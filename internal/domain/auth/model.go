package auth

import "github.com/golang-jwt/jwt/v5"

// AccessTokenClaims are the claims for the access token
type AccessTokenClaims struct {
	Sid string `json:"sid"`
	jwt.RegisteredClaims
}

// Identity represents the identity of a user
type Identity struct {
	UserID    string
	SessionID string
}
