package auth

import (
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/Anvoria/authly/internal/utils"
)

const (
	// IdentityKey is the key used to store the identity in Fiber context
	IdentityKey = "identity"
	// ScopesKey is the key used to store scopes in Fiber context
	ScopesKey = "scopes"
)

// AuthMiddleware returns a Fiber middleware that validates an incoming Bearer access token.
// It checks issuer and optionally validates audience if expectedAudience is provided.
// For routes like /user/info, pass empty expectedAudience to allow any logged-in user.
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

		// Validate issuer and expiration
		iss := claims.Issuer()
		if issuer != "" && iss != issuer {
			slog.Error("token issuer mismatch", "expected", issuer, "got", iss)
			return utils.ErrorResponse(c, ErrTokenExpiredOrInvalid.Error(), fiber.StatusUnauthorized)
		}

		exp := claims.Expiration()
		if exp.IsZero() {
			return utils.ErrorResponse(c, ErrTokenExpiredOrInvalid.Error(), fiber.StatusUnauthorized)
		}
		if time.Now().After(exp) {
			return utils.ErrorResponse(c, ErrTokenExpiredOrInvalid.Error(), fiber.StatusUnauthorized)
		}

		if len(expectedAudience) > 0 {
			aud := claims.Audience()
			audMatch := false
			for _, expected := range expectedAudience {
				if slices.Contains(aud, expected) {
					audMatch = true
					break
				}
			}
			if !audMatch {
				slog.Error("token audience mismatch", "expected", expectedAudience, "got", aud)
				return utils.ErrorResponse(c, ErrTokenExpiredOrInvalid.Error(), fiber.StatusUnauthorized)
			}
		}

		revoked, err := svc.IsTokenRevoked(claims)
		if err != nil {
			return utils.ErrorResponse(c, ErrTokenValidationError.Error(), fiber.StatusInternalServerError)
		}
		if revoked {
			return utils.ErrorResponse(c, ErrTokenRevoked.Error(), fiber.StatusUnauthorized)
		}

		scopes := claims.GetScopes()

		identity := &Identity{
			UserID:      claims.Subject(),
			SessionID:   claims.GetSid(),
			PermissionV: claims.GetPermissionV(),
			Scopes:      scopes,
		}

		c.Locals(IdentityKey, identity)
		c.Locals(ScopesKey, scopes)

		return c.Next()
	}
}

// RequireScope returns a middleware that requires a specific scope (service:resource or service)
func RequireScope(requiredScope string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		scopes, ok := c.Locals(ScopesKey).(map[string]uint64)
		if !ok || scopes == nil {
			return utils.ErrorResponse(c, ErrUnauthorized.Error(), fiber.StatusForbidden)
		}

		bitmask, exists := scopes[requiredScope]
		if !exists || bitmask == 0 {
			return utils.ErrorResponse(c, ErrUnauthorized.Error(), fiber.StatusForbidden)
		}

		return c.Next()
	}
}

// RequirePermission returns a middleware that requires a specific permission bit for a scope
func RequirePermission(requiredScope string, requiredBit uint8) fiber.Handler {
	return func(c *fiber.Ctx) error {
		scopes, ok := c.Locals(ScopesKey).(map[string]uint64)
		if !ok || scopes == nil {
			return utils.ErrorResponse(c, ErrUnauthorized.Error(), fiber.StatusForbidden)
		}

		bitmask, exists := scopes[requiredScope]
		if !exists || bitmask == 0 {
			return utils.ErrorResponse(c, ErrUnauthorized.Error(), fiber.StatusForbidden)
		}

		if (bitmask & (1 << requiredBit)) == 0 {
			return utils.ErrorResponse(c, ErrUnauthorized.Error(), fiber.StatusForbidden)
		}

		return c.Next()
	}
}

// GetIdentity retrieves the *Identity stored in the current Fiber context under IdentityKey.
// It returns the Identity pointer, or nil if no identity is present or the stored value is not an *Identity.
func GetIdentity(c *fiber.Ctx) *Identity {
	identity, ok := c.Locals(IdentityKey).(*Identity)
	if !ok {
		return nil
	}
	return identity
}

// GetScopes retrieves the scopes map stored in the current Fiber context under ScopesKey.
func GetScopes(c *fiber.Ctx) map[string]uint64 {
	scopes, ok := c.Locals(ScopesKey).(map[string]uint64)
	if !ok {
		return make(map[string]uint64)
	}
	return scopes
}
