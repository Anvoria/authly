package oidc

import (
	"strings"

	"github.com/Anvoria/authly/internal/domain/auth"
	"github.com/Anvoria/authly/internal/domain/permission"
	"github.com/Anvoria/authly/internal/domain/session"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// SessionMiddleware returns a Fiber middleware that authenticates requests using a session cookie.
// It validates the session and attaches the resolved Identity to the request context.
// SessionMiddleware returns a Fiber middleware that authenticates requests using a session cookie
// and, when successful, attaches the resolved Identity and scopes to the request context.
//
// The middleware reads the "session" cookie expecting the format "sessionID:secret", validates the
// session via the provided session service, builds permission scopes and version via the provided
// permission service (defaulting to an empty scope map and permission version 1 on error), and
// stores the resulting *auth.Identity and scopes map in the request locals under auth.IdentityKey
// and auth.ScopesKey. If any step fails, the request proceeds without an authenticated identity.
func SessionMiddleware(sessionService session.Service, permissionService permission.ServiceInterface) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get session cookie
		sessionCookie := c.Cookies("session")
		if sessionCookie == "" {
			return c.Next()
		}

		parts := strings.SplitN(sessionCookie, ":", 2) // sessionID:secret
		if len(parts) != 2 {
			return c.Next()
		}

		sessionIDStr := parts[0]
		secret := parts[1]

		if sessionIDStr == "" || secret == "" {
			return c.Next()
		}

		// Parse session ID as UUID
		sessionID, err := uuid.Parse(sessionIDStr)
		if err != nil {
			return c.Next()
		}

		// Validate session
		sess, err := sessionService.Validate(sessionID, secret)
		if err != nil {
			return c.Next()
		}

		// Build scopes from user permissions
		scopes, err := permissionService.BuildScopes(sess.UserID)
		if err != nil {
			scopes = make(map[string]uint64)
		}

		pver, err := permissionService.GetPermissionVersion(sess.UserID)
		if err != nil {
			pver = 1
		}

		identity := &auth.Identity{
			UserID:      sess.UserID,
			SessionID:   sess.ID.String(),
			PermissionV: pver,
			Scopes:      scopes,
		}

		c.Locals(auth.IdentityKey, identity)
		c.Locals(auth.ScopesKey, scopes)

		return c.Next()
	}
}
