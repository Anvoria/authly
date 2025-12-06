package auth

import (
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

// Identity represents the identity of a user
type Identity struct {
	UserID    string
	SessionID string
}
