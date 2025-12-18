package auth

import (
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwt"
)

// AccessTokenClaims are the claims for the access token
type AccessTokenClaims struct {
	Sid   string
	Token jwt.Token
}

// Helper methods to access token claims
func (c *AccessTokenClaims) Subject() string {
	sub, _ := c.Token.Subject()
	return sub
}

func (c *AccessTokenClaims) Audience() []string {
	aud, _ := c.Token.Audience()
	return aud
}

func (c *AccessTokenClaims) Issuer() string {
	iss, _ := c.Token.Issuer()
	return iss
}

func (c *AccessTokenClaims) IssuedAt() time.Time {
	iat, _ := c.Token.IssuedAt()
	return iat
}

func (c *AccessTokenClaims) Expiration() time.Time {
	exp, _ := c.Token.Expiration()
	return exp
}

// GetSid returns the session ID from the token claims
// It extracts the "sid" claim from the token, with fallback to the stored Sid field
func (c *AccessTokenClaims) GetSid() string {
	var sid any
	if c.Token.Get("sid", &sid) == nil {
		if s, ok := sid.(string); ok {
			c.Sid = s
			return s
		}
	}
	return c.Sid
}

// GetPermissions extracts permissions from the token claims
// Permissions are stored in the "permissions" claim, not "scopes"
func (c *AccessTokenClaims) GetPermissions() map[string]uint64 {
	var permissions map[string]uint64
	var permissionsAny any
	if c.Token.Get("permissions", &permissionsAny) == nil {
		if p, ok := permissionsAny.(map[string]uint64); ok {
			permissions = p
		} else if p, ok := permissionsAny.(map[string]any); ok {
			permissions = make(map[string]uint64)
			for k, v := range p {
				if u, ok := v.(uint64); ok {
					permissions[k] = u
				} else if f, ok := v.(float64); ok {
					permissions[k] = uint64(f)
				}
			}
		}
	}
	if permissions == nil {
		permissions = make(map[string]uint64)
	}
	return permissions
}

// GetScopes is deprecated - use GetPermissions instead
// Kept for backward compatibility but returns permissions from "permissions" claim
func (c *AccessTokenClaims) GetScopes() map[string]uint64 {
	return c.GetPermissions()
}

// GetScopeString extracts OIDC scope string from token claims
func (c *AccessTokenClaims) GetScopeString() string {
	var scope string
	if c.Token.Get("scope", &scope) == nil {
		return scope
	}
	return ""
}

// GetRequestedScopes extracts all requested scopes from token claims
// Returns the "requested_scopes" claim (all scopes including openid, profile, email)
func (c *AccessTokenClaims) GetRequestedScopes() []string {
	var requestedScopes string
	if c.Token.Get("requested_scopes", &requestedScopes) == nil {
		if requestedScopes != "" {
			return strings.Fields(requestedScopes)
		}
	}
	scope := c.GetScopeString()
	if scope != "" {
		return strings.Fields(scope)
	}
	return []string{}
}

// GetPermissionV extracts permission version from the token claims
func (c *AccessTokenClaims) GetPermissionV() int {
	var pver int
	var pverAny any
	if c.Token.Get("pver", &pverAny) == nil {
		if p, ok := pverAny.(int); ok {
			pver = p
		} else if p, ok := pverAny.(float64); ok {
			pver = int(p)
		}
	}
	return pver
}

// Validate validates standard JWT claims
func (c *AccessTokenClaims) Validate(issuer string, expectedAudience []string) error {
	exp := c.Expiration()
	if exp.IsZero() {
		return errors.New("token missing expiration claim")
	}
	if time.Now().After(exp) {
		return errors.New("token expired")
	}

	iss := c.Issuer()
	if issuer != "" && iss != issuer {
		return errors.New("token issuer mismatch")
	}

	aud := c.Audience()
	if len(expectedAudience) > 0 {
		audMatch := false
		for _, expected := range expectedAudience {
			if slices.Contains(aud, expected) {
				audMatch = true
			}
			if audMatch {
				break
			}
		}
		if !audMatch {
			return errors.New("token audience mismatch")
		}
	}

	return nil
}

// Identity represents the identity of a user
type Identity struct {
	UserID      string
	SessionID   string
	PermissionV int
	Scopes      map[string]uint64
}
