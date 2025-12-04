package config

import (
	"os"
)

// Environment holds the environment variables
type Environment struct {
	ConfigPath string `env:"CONFIG_PATH" default:"config.yaml"`
	JWTSecret string `env:"JWT_SECRET" default:""`
}

// LoadEnv loads the environment variables
func LoadEnv() *Environment {
	return &Environment{
		ConfigPath: getEnv("CONFIG_PATH", "config.yaml"),
		JWTSecret: getEnv("JWT_SECRET", ""),
	}
}

// getEnv gets the environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
