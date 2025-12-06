package server

import (
	"log/slog"
	"os"

	"github.com/Anvoria/authly/internal/config"
	"github.com/Anvoria/authly/internal/database"
	"github.com/Anvoria/authly/internal/migrations"
	"github.com/gofiber/fiber/v2"
)

// Start initializes and starts the HTTP server
func Start(cfg *config.Config) error {
	initLogger(cfg.Logging.Level)

	app := fiber.New()

	if err := database.ConnectDB(cfg); err != nil {
		slog.Error("Failed to connect to database", "error", err)
		return err
	}
	slog.Info("Database connected successfully")

	if err := migrations.RunMigrations(cfg); err != nil {
		slog.Error("Failed to run migrations", "error", err)
		return err
	}
	slog.Info("Migrations completed successfully")

	envConfig := config.LoadEnv()
	slog.Info("Environment loaded", "environment", envConfig.Environment.String())
	if err := SetupRoutes(app, envConfig, cfg); err != nil {
		slog.Error("Failed to setup routes", "error", err)
		return err
	}

	addr := cfg.Server.Address()
	slog.Info("Server starting", "address", addr)
	if err := app.Listen(addr); err != nil {
		slog.Error("Failed to start server", "error", err)
		return err
	}

	return nil
}

func initLogger(level string) {
	var logLevel slog.Level
	switch level {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: logLevel,
	}

	handler := slog.NewTextHandler(os.Stdout, opts)
	slog.SetDefault(slog.New(handler))
}
