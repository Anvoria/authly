package server

import (
	"fmt"
	"log/slog"

	"github.com/Anvoria/authly/internal/cache"
	"github.com/Anvoria/authly/internal/config"
	"github.com/Anvoria/authly/internal/database"
	"github.com/Anvoria/authly/internal/domain/auth"
	"github.com/Anvoria/authly/internal/domain/oidc"
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
// SetupRoutes configures all HTTP routes, repositories, caches, services, authentication, and OpenID Connect endpoints on the provided Fiber app.
// It initializes repositories (user, session, service, permission), caches, core services, key store, authentication and OIDC services and handlers, then registers routes under /v1 (including /auth, /oauth and well-known endpoints).
// Returns an error if cryptographic keys cannot be loaded or the configured active key is not found.
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

	issuer := cfg.Server.Domain

	// Initialize auth service
	authService := auth.NewService(userRepo, sessionService, permissionService, keyStore, issuer, tokenRevocationCache)
	authHandler := auth.NewHandler(authService, userService)

	// Setup auth routes
	authGroup := api.Group("/auth")
	authGroup.Post("/login", authHandler.Login)
	authGroup.Post("/register", authHandler.Register)

	authSessionGroup := api.Group("/auth")
	authSessionGroup.Use(oidc.SessionMiddleware(sessionService, permissionService))
	authSessionGroup.Get("/me", authHandler.Me)

	authServiceRepoAdapter := auth.NewServiceRepositoryAdapter(serviceCache)

	// Initialize OIDC repositories and services
	authCodeRepo := oidc.NewRepository(database.DB)
	oidcService := oidc.NewService(serviceRepo, authCodeRepo, authService, sessionService, permissionService, userService)
	oidcHandler := oidc.NewHandler(oidcService)

	oauthGroup := api.Group("/oauth")
	oauthGroup.Use(oidc.SessionMiddleware(sessionService, permissionService))
	oauthGroup.Get("/authorize", oidcHandler.Authorize)
	oauthGroup.Post("/authorize/confirm", oidcHandler.ConfirmAuthorization)
	oauthGroup.Post("/token", oidcHandler.Token)

	oauthGroup.Get("/authorize/validate", oidcHandler.ValidateAuthorization)

	oauthGroupProtected := api.Group("/oauth")
	oauthGroupProtected.Use(auth.AuthMiddleware(keyStore, authService, issuer, authServiceRepoAdapter))
	oauthGroupProtected.Get("/userinfo", oidcHandler.UserInfo)

	// Setup well-known endpoints
	app.Get("/.well-known/jwks.json", auth.JWKSHandler(keyStore))
	app.Get("/.well-known/openid-configuration", oidc.OpenIDConfigurationHandler(issuer))
	return nil
}
