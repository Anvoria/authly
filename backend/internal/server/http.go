package server

import (
	"errors"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/Anvoria/authly/internal/cache"
	"github.com/Anvoria/authly/internal/config"
	"github.com/Anvoria/authly/internal/database"
	"github.com/Anvoria/authly/internal/migrations"
	"github.com/Anvoria/authly/internal/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/limiter"
)

// Start initializes logging, configures the Fiber app (including CORS), connects to the database and Redis, runs migrations, loads environment configuration, registers routes, and starts listening on the configured address.
// It returns an error if any startup step fails.
func Start(cfg *config.Config) error {
	initLogger(cfg.Logging.Level)

	app := fiber.New(fiber.Config{
		BodyLimit: 10 * 1024 * 1024, // 10MB
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			if apiErr, ok := err.(*utils.APIError); ok {
				return utils.ErrorResponse(c, apiErr)
			}

			var e *fiber.Error
			if errors.As(err, &e) {
				return utils.ErrorResponse(c, utils.NewAPIError(
					"HTTP_ERROR",
					e.Message,
					e.Code,
				))
			}

			return utils.ErrorResponse(c, utils.NewAPIError(
				"INTERNAL_SERVER_ERROR",
				"An unexpected error occurred",
				fiber.StatusInternalServerError,
			))
		},
	})

	// Use Helmet for security headers
	app.Use(helmet.New())

	// Configure Rate Limiting
	app.Use(limiter.New(limiter.Config{
		Max:        cfg.Server.RateLimit.Max,
		Expiration: time.Duration(cfg.Server.RateLimit.Expiration) * time.Second,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return utils.ErrorResponse(c, utils.NewAPIError(
				"TOO_MANY_REQUESTS",
				"Too many requests, please try again later.",
				fiber.StatusTooManyRequests,
			))
		},
	}))

	// Configure CORS
	app.Use(cors.New(cors.Config{
		AllowOrigins:     strings.Join(cfg.Server.AllowedOrigins, ","),
		AllowMethods:     "GET,POST,PUT,DELETE,PATCH,OPTIONS",
		AllowHeaders:     "Origin,Content-Type,Accept,Authorization",
		AllowCredentials: true,
		ExposeHeaders:    "Content-Length",
		MaxAge:           3600,
	}))

	if err := database.ConnectDB(cfg); err != nil {
		slog.Error("Failed to connect to database", "error", err)
		return err
	}
	slog.Info("Database connected successfully")

	if err := cache.ConnectRedis(&cfg.Redis); err != nil {
		slog.Error("Failed to connect to Redis", "error", err)
		return err
	}

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
	slog.Info("Server starting",
		"address", addr,
		"app", cfg.App.Name,
		"version", cfg.App.Version,
	)
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
