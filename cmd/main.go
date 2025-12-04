package main

import (
	"log/slog"
	"os"

	"github.com/Anvoria/authly/internal/config"
	"github.com/Anvoria/authly/internal/database"
	"github.com/Anvoria/authly/internal/router"
	"github.com/gofiber/fiber/v2"
)

func main() {
	envConfig := config.LoadEnv()

	cfg, err := config.Load(envConfig.ConfigPath)
	if err != nil {
		slog.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	initLogger(cfg.Logging.Level)

	app := fiber.New()

	if err := database.ConnectDB(cfg); err != nil {
		slog.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	slog.Info("Database connected successfully")

	router.SetupRoutes(app)

	addr := cfg.Server.Address()
	slog.Info("Server starting", "address", addr)
	if err := app.Listen(addr); err != nil {
		slog.Error("Failed to start server", "error", err)
		os.Exit(1)
	}
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
