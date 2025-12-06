package migrations

import (
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/Anvoria/authly/internal/config"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// RunMigrations runs all database migrations using golang-migrate
func RunMigrations(cfg *config.Config) error {
	// Get the directory where this file is located
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return fmt.Errorf("failed to get current file path")
	}
	migrationsDir := filepath.Dir(filename)
	migrationsPath := filepath.Join(migrationsDir)
	migrationsURL := fmt.Sprintf("file://%s", migrationsPath)

	m, err := migrate.New(migrationsURL, cfg.Database.URL())
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil {
		// If already at latest version, that's not an error
		if err == migrate.ErrNoChange {
			return nil
		}
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}
