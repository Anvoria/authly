package config

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// EnvironmentType represents the application environment
type EnvironmentType string

const (
	EnvironmentDevelopment EnvironmentType = "development"
	EnvironmentProduction  EnvironmentType = "production"
)

// String returns the string representation of the environment type
func (e EnvironmentType) String() string {
	return string(e)
}

// IsValid checks if the environment type is valid
func (e EnvironmentType) IsValid() bool {
	switch e {
	case EnvironmentDevelopment, EnvironmentProduction:
		return true
	default:
		return false
	}
}

// Environment holds the environment variables
type Environment struct {
	Environment EnvironmentType `env:"ENVIRONMENT"`
	ConfigPath  string          `env:"CONFIG_PATH"`
	JWTSecret   string          `env:"JWT_SECRET"`
	PrivateKey  string          `env:"PRIVATE_KEY"`
}

// LoadEnv loads the environment variables
// It first tries to load from .env file, then falls back to system environment variables
func LoadEnv() *Environment {
	_ = godotenv.Load()

	envStr := getEnv("ENVIRONMENT", string(EnvironmentDevelopment))
	envStr = strings.TrimSpace(envStr)
	envStr = strings.ToLower(envStr)
	envType := EnvironmentType(envStr)

	// Validate and default to development if invalid
	if !envType.IsValid() {
		envType = EnvironmentDevelopment
	}

	return &Environment{
		Environment: envType,
		ConfigPath:  getEnv("CONFIG_PATH", "config.yaml"),
		JWTSecret:   getEnv("JWT_SECRET", ""),
		PrivateKey:  getEnv("PRIVATE_KEY", ""),
	}
}

// getEnv gets the environment variable with a default value
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value != "" {
		return value
	}
	return defaultValue
}

// LoadRSAPrivateKey loads RSA private key from PEM string
// If privateKeyPEM is empty and environment is production, returns an error
// If privateKeyPEM is empty and environment is development, generates a new key
func LoadRSAPrivateKey(privateKeyPEM string, env EnvironmentType) (*rsa.PrivateKey, error) {
	if privateKeyPEM == "" {
		if env == EnvironmentProduction {
			return nil, fmt.Errorf("private key is required in production environment")
		}

		privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return nil, fmt.Errorf("failed to generate private key: %w", err)
		}
		return privateKey, nil
	}

	block, _ := pem.Decode([]byte(privateKeyPEM))
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	var key *rsa.PrivateKey
	var err error

	if key, err = x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}

	pkcs8Key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	rsaKey, ok := pkcs8Key.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("key is not an RSA private key")
	}

	return rsaKey, nil
}
