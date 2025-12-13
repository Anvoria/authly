package server

import (
	"fmt"
	"log/slog"

	"github.com/Anvoria/authly/internal/cache"
	"github.com/Anvoria/authly/internal/config"
	"github.com/Anvoria/authly/internal/database"
	"github.com/Anvoria/authly/internal/domain/auth"
	perm "github.com/Anvoria/authly/internal/domain/permission"
	svc "github.com/Anvoria/authly/internal/domain/service"
	"github.com/Anvoria/authly/internal/domain/session"
	"github.com/Anvoria/authly/internal/domain/user"
	"github.com/gofiber/fiber/v2"
)

// SetupRoutes configures HTTP routes, repositories, services, authentication, and middleware on the provided Fiber app.
//
// It mounts the API under "/v1", registers authentication endpoints ("/auth/login", "/auth/register"),
// exposes the JWKS at "/.well-known/jwks.json", and creates a protected route for "/user/info" that requires authentication.
// The function also initializes repositories and services required by authentication and permission checks.
// SetupRoutes configures HTTP routes, repositories, caches, services, authentication and middleware on the provided Fiber app.
// It mounts the API under "/v1", registers authentication endpoints under "/v1/auth" (POST /login, POST /register),
// exposes the JWKS at "/.well-known/jwks.json", and protects GET "/v1/user/info" with authentication middleware.
// The function loads cryptographic keys from cfg.Auth.KeysPath and requires the configured active KID to be present;
// it returns an error if keys cannot be loaded or the active key is not found.
func SetupRoutes(app *fiber.App, envConfig *config.Environment, cfg *config.Config) error {
	api := app.Group("/v1")

	// Initialize repositories
	userRepo := user.NewRepository(database.DB)
	sessionRepo := session.NewRepository(database.DB)
	serviceRepo := svc.NewRepository(database.DB)
	permissionRepo := perm.NewRepository(database.DB)

	// Initialize cache
	serviceCache := cache.NewServiceCache(serviceRepo)
	tokenRevocationCache := cache.NewTokenRevocationCache()

	// Initialize services
	sessionService := session.NewServiceWithCache(sessionRepo, tokenRevocationCache)
	serviceRepoAdapter := perm.NewServiceRepositoryAdapter(serviceRepo)
	permissionService := perm.NewService(permissionRepo, serviceRepoAdapter)
	userService := user.NewService(userRepo)

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
	authService := auth.NewService(userRepo, sessionService, permissionService, keyStore, cfg.App.Name, tokenRevocationCache)
	authHandler := auth.NewHandler(authService, userService)

	// Setup auth routes
	authGroup := api.Group("/auth")
	authGroup.Post("/login", authHandler.Login)
	authGroup.Post("/register", authHandler.Register)

	protectedGroup := api.Group("")
	authServiceRepoAdapter := auth.NewServiceRepositoryAdapter(serviceCache)
	protectedGroup.Use(auth.AuthMiddleware(keyStore, authService, cfg.App.Name, authServiceRepoAdapter))
	protectedGroup.Get("/user/info", authHandler.GetUserInfo)

	app.Get("/.well-known/jwks.json", auth.JWKSHandler(keyStore))

	return nil
}
