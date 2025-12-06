package utils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Anvoria/authly/internal/config"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
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

// SetupTestDB creates a database connection suitable for tests.
// Defaults to an in-memory SQLite database to avoid external dependencies,
// but can be switched to Postgres by setting TEST_DB_DRIVER=postgres.
func SetupTestDB(t *testing.T, models ...any) *gorm.DB {
	driver := os.Getenv("TEST_DB_DRIVER")
	if driver == "" {
		driver = "sqlite"
	}

	var (
		db  *gorm.DB
		err error
	)

	switch driver {
	case "postgres":
		cfg := LoadTestConfig(t)
		dsn := cfg.Database.DSN()
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	default:
		db, err = gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
			SkipDefaultTransaction: true,
		})
	}

	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	if len(models) > 0 {
		migrator := db.Migrator()
		for _, model := range models {
			if migrator.HasTable(model) {
				if err := migrator.DropTable(model); err != nil {
					t.Fatalf("Failed to reset test database schema: %v", err)
				}
			}
		}
		if err := migrator.AutoMigrate(models...); err != nil {
			t.Fatalf("Failed to migrate test database: %v", err)
		}
	}

	return db
}
