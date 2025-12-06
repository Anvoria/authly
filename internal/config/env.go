package config

import (
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
}

// LoadEnv loads the environment variables
// LoadEnv loads application environment settings into an Environment struct.
// It attempts to read a .env file, then reads ENVIRONMENT, CONFIG_PATH, and JWT_SECRET from the environment,
// normalizes and validates ENVIRONMENT (defaults to "development" when invalid), and returns a pointer to the populated Environment
// (ConfigPath defaults to "config.yaml", JWTSecret defaults to an empty string).
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
	}
}

// getEnv retrieves the environment variable named by key and returns defaultValue when the variable is unset or empty.
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value != "" {
		return value
	}
	return defaultValue
}