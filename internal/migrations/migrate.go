package migrations

import (
	"embed"
	"fmt"

	"github.com/Anvoria/authly/internal/config"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed *.sql
var migrations embed.FS

// RunMigrations runs all database migrations using golang-migrate
func RunMigrations(cfg *config.Config) error {
	// Create iofs source from embedded migrations
	source, err := iofs.New(migrations, ".")
	if err != nil {
		return fmt.Errorf("failed to create migration source: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", source, cfg.Database.URL())
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
