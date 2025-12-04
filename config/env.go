package config

import (
	"os"
)

type Environment struct {
	ConfigPath string `env:"CONFIG_PATH" default:"config.yaml"`
	JWTSecret string `env:"JWT_SECRET" default:""`
}

func LoadEnv() *Environment {
	return &Environment{
		ConfigPath: getEnv("CONFIG_PATH", "config.yaml"),
		JWTSecret: getEnv("JWT_SECRET", ""),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
