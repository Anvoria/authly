package utils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Anvoria/authly/internal/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// FindProjectRoot finds the project root directory by looking for go.mod file
func FindProjectRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return wd, nil
		}
		dir = parent
	}
}

// LoadTestConfig loads configuration for testing
// Config path can be overridden with TEST_CONFIG_PATH env variable
// Defaults to config.yaml in project root
func LoadTestConfig(t *testing.T) *config.Config {
	projectRoot, err := FindProjectRoot()
	if err != nil {
		t.Fatalf("Failed to find project root: %v", err)
	}

	configPath := os.Getenv("TEST_CONFIG_PATH")
	if configPath == "" {
		configPath = "config.yaml"
	}

	// If path is not absolute, make it relative to project root
	if !filepath.IsAbs(configPath) {
		configPath = filepath.Join(projectRoot, configPath)
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config from %s: %v", configPath, err)
	}

	return cfg
}

// SetupTestDB creates a PostgreSQL database connection for testing
// Loads database configuration from config file and auto-migrates provided models
func SetupTestDB(t *testing.T, models ...any) *gorm.DB {
	cfg := LoadTestConfig(t)
	dsn := cfg.Database.DSN()

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Auto migrate provided models
	if len(models) > 0 {
		if err := db.AutoMigrate(models...); err != nil {
			t.Fatalf("Failed to migrate test database: %v", err)
		}
	}

	return db
}
