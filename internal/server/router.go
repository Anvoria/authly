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

// SetupRoutes sets up the routes for the application
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
	authGroup.Post("/register", authHandler.Register)

	app.Get("/.well-known/jwks.json", auth.JWKSHandler(keyStore))

	return nil
}
