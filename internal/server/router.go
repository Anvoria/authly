package server

import (
	"fmt"
	"log/slog"

	"github.com/Anvoria/authly/internal/config"
	"github.com/Anvoria/authly/internal/database"
	"github.com/Anvoria/authly/internal/domain/auth"
	"github.com/Anvoria/authly/internal/domain/session"
	"github.com/Anvoria/authly/internal/domain/user"
	"github.com/gofiber/fiber/v2"
)

// SetupRoutes configures the application's HTTP routes and authentication infrastructure.
// It creates the /v1 API group, initializes repositories and services, loads the configured key store and active key,
// registers the /v1/auth/login route and the /.well-known/jwks.json JWKS endpoint.
// Returns an error if the key store cannot be loaded or the configured active key is not found.
func SetupRoutes(app *fiber.App, envConfig *config.Environment, cfg *config.Config) error {
	api := app.Group("/v1")

	// Initialize repositories
	userRepo := user.NewRepository(database.DB)
	sessionRepo := session.NewRepository(database.DB)

	// Initialize services
	sessionService := session.NewService(sessionRepo)

	keyStore, err := auth.LoadKeys(cfg.Auth.KeysPath, cfg.Auth.ActiveKID)
	if err != nil {
		return fmt.Errorf("failed to load keys: %w", err)
	}

	activeKey, err := keyStore.GetActiveKey()
	if err != nil {
		return fmt.Errorf("active key with KID %s not found in key store: %w", cfg.Auth.ActiveKID, err)
	}

	keyID, _ := activeKey.KeyID()
	slog.Info("Active key loaded", "key", cfg.Auth.ActiveKID, "key_id", keyID)

	// Initialize auth service
	authService := auth.NewService(userRepo, sessionService, keyStore, cfg.App.Name)
	authHandler := auth.NewHandler(authService)

	// Setup auth routes
	authGroup := api.Group("/auth")
	authGroup.Post("/login", authHandler.Login)

	app.Get("/.well-known/jwks.json", auth.JWKSHandler(keyStore))

	return nil
}