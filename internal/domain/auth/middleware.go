package auth

import (
	"context"
	"log/slog"
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

// ServiceRepository defines the interface needed to find services by domain or code
type ServiceRepository interface {
	FindByDomain(ctx context.Context, domain string) (ServiceInfo, error)
	FindByCode(ctx context.Context, code string) (ServiceInfo, error)
}

// ServiceInfo provides information about a service needed for AUD verification
type ServiceInfo interface {
	GetCode() string
	IsActive() bool
}

// AuthMiddleware returns a Fiber middleware that authenticates requests using a bearer token
// from the Authorization header, enforces issuer/expiration/audience rules, checks revocation,
// and attaches the resolved Identity and scopes to the request context.
//
// If a non-empty issuer is provided the middleware validates the token issuer matches it.
// When a ServiceRepository is supplied the middleware validates that each service code in the
// token's audience claim corresponds to a valid, active service. This validation is based
// solely on the cryptographically signed token claims and does not rely on client-controlled
// headers like Origin or Referer, which can be spoofed. On successful validation the middleware
// stores the Identity under IdentityKey and the scopes map under ScopesKey in the Fiber context
// before calling the next handler.
func AuthMiddleware(keyStore *KeyStore, svc AuthService, issuer string, serviceRepo ServiceRepository) fiber.Handler {
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

		aud := claims.Audience()
		if len(aud) == 0 {
			return utils.ErrorResponse(c, ErrTokenValidationError.Error(), fiber.StatusUnauthorized)
		}

		if serviceRepo != nil {
			ctx := c.UserContext()
			validAudience := false
			for _, serviceCode := range aud {
				service, err := serviceRepo.FindByCode(ctx, serviceCode)
				if err != nil {
					slog.Debug("service not found for audience code", "code", serviceCode, "error", err)
					continue
				}

				if !service.IsActive() {
					slog.Debug("service is not active", "code", serviceCode)
					continue
				}

				validAudience = true
				break
			}

			if !validAudience {
				slog.Error("token audience does not contain any valid active service", "audience", aud)
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

// RequireScope returns a middleware that requires the specified scope to be present and have a non-zero bitmask in the request's scopes stored under ScopesKey.
// The middleware responds with HTTP 403 Forbidden if scopes are missing, the scope is absent, or its bitmask is zero.
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

// RequirePermission returns a middleware that allows the request only when the specified scope contains the given permission bit.
//
// The middleware checks the request's scopes (stored under ScopesKey in the context) and responds with HTTP 403 Forbidden if:
// - the scopes map is missing or invalid,
// - the required scope is not present or has a zero bitmask,
// - the required permission bit is not set in the scope's bitmask.
func RequirePermission(requiredScope string, requiredBit uint8) fiber.Handler {
	return func(c *fiber.Ctx) error {
		scopes, ok := c.Locals(ScopesKey).(map[string]uint64)
		if !ok || scopes == nil {
			return utils.ErrorResponse(c, ErrUnauthorized.Error(), fiber.StatusForbidden)
		}

		if requiredBit >= 64 {
			slog.Error("invalid bit position: must be 0-63", "bit", requiredBit)
			return utils.ErrorResponse(c, ErrTokenValidationError.Error(), fiber.StatusForbidden)
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
// GetIdentity retrieves the Identity stored in the Fiber context under IdentityKey.
// It returns the *Identity or nil if no identity is present or the stored value is not an *Identity.
func GetIdentity(c *fiber.Ctx) *Identity {
	identity, ok := c.Locals(IdentityKey).(*Identity)
	if !ok {
		return nil
	}
	return identity
}

// GetScopes returns the scopes map stored in the Fiber context under ScopesKey.
// If no scopes are present or the stored value has a different type, it returns an empty map.
func GetScopes(c *fiber.Ctx) map[string]uint64 {
	scopes, ok := c.Locals(ScopesKey).(map[string]uint64)
	if !ok {
		return make(map[string]uint64)
	}
	return scopes
}
