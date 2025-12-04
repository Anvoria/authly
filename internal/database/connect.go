package database

import (
	"fmt"
	"log"

	"github.com/Anvoria/authly/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// DB is the global database connection instance
var DB *gorm.DB

// ConnectDB connects to the database using configuration from YAML
func ConnectDB(cfg *config.Config) error {
	var err error

	dsn := cfg.Database.DSN()

	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to connect database: %w", err)
	}

	log.Println("Connection opened to database")
	return nil
}
