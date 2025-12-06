package server

import (
	"time"

	"github.com/Anvoria/authly/internal/config"
	"github.com/Anvoria/authly/internal/database"
	"github.com/Anvoria/authly/internal/domain/auth"
	"github.com/Anvoria/authly/internal/domain/session"
	"github.com/Anvoria/authly/internal/domain/user"
	"github.com/gofiber/fiber/v2"
)

// SetupRoutes sets up the routes for the application
func SetupRoutes(app *fiber.App, envConfig *config.Environment) error {
	api := app.Group("/v1")

	// Initialize repositories
	userRepo := user.NewRepository(database.DB)
	sessionRepo := session.NewRepository(database.DB)

	// Initialize services
	sessionService := session.NewService(sessionRepo)

	tokenGenerator := auth.NewTokenGenerator(privateKey, "authly", 15*time.Minute)

	// Initialize auth service
	authService := auth.NewService(userRepo, sessionService, tokenGenerator)
	authHandler := auth.NewHandler(authService)

	// Setup auth routes
	auth := api.Group("/auth")
	auth.Post("/login", authHandler.Login)

	return nil
}
