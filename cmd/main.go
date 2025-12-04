package main

import (
	"log/slog"
	"os"

	"github.com/Anvoria/authly/config"
	"github.com/gofiber/fiber/v2"
)

func main() {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config.yaml"
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		slog.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	initLogger(cfg.Logging.Level)

	app := fiber.New()

	setupRoutes(app)

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

func setupRoutes(app *fiber.App) {
	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message": "Authly API",
			"status":  "running",
		})
	})
}
