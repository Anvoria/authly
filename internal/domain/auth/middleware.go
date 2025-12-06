package auth

import (
	"log/slog"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/Anvoria/authly/internal/utils"
)

const (
	// IdentityKey is the key used to store the identity in Fiber context
	IdentityKey = "identity"
)

// AuthMiddleware verifies the access token
func AuthMiddleware(keyStore *KeyStore, svc AuthService, issuer string, expectedAudience []string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return utils.ErrorResponse(c, ErrMissingAuthorizationHeader.Error(), fiber.StatusUnauthorized)
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			return utils.ErrorResponse(c, ErrInvalidAuthorizationHeader.Error(), fiber.StatusUnauthorized)
		}

		token := parts[1]
		if token == "" {
			return utils.ErrorResponse(c, ErrMissingToken.Error(), fiber.StatusUnauthorized)
		}

		claims, err := keyStore.Verify(token)
		if err != nil {
			return utils.ErrorResponse(c, ErrInvalidToken.Error(), fiber.StatusUnauthorized)
		}

		if err := claims.Validate(issuer, expectedAudience); err != nil {
			slog.Error("token validation error", "error", err)
			return utils.ErrorResponse(c, ErrTokenExpiredOrInvalid.Error(), fiber.StatusUnauthorized)
		}

		revoked, err := svc.IsTokenRevoked(claims)
		if err != nil {
			return utils.ErrorResponse(c, ErrTokenValidationError.Error(), fiber.StatusInternalServerError)
		}
		if revoked {
			return utils.ErrorResponse(c, ErrTokenRevoked.Error(), fiber.StatusUnauthorized)
		}

		identity := &Identity{
			UserID:      claims.Subject(),
			SessionID:   claims.GetSid(),
			PermissionV: claims.GetPermissionV(),
		}

		c.Locals(IdentityKey, identity)

		return c.Next()
	}
}

// GetIdentity extracts the identity from Fiber context
func GetIdentity(c *fiber.Ctx) *Identity {
	identity, ok := c.Locals(IdentityKey).(*Identity)
	if !ok {
		return nil
	}
	return identity
}
