package oidc

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"
)

// generateAuthorizationCode generates a cryptographically random authorization code
func (s *Service) generateAuthorizationCode() (string, error) {
	// Generate 32 random bytes
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	// Encode to base64url (URL-safe base64)
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

// verifyCodeVerifier verifies code_verifier against code_challenge using the specified method
func (s *Service) verifyCodeVerifier(codeVerifier, codeChallenge, method string) error {
	if method != "s256" && method != "S256" {
		return ErrInvalidCodeChallengeMethod
	}

	// Compute SHA256 hash of code_verifier
	hash := sha256.Sum256([]byte(codeVerifier))
	// Encode to base64url
	computedChallenge := base64.RawURLEncoding.EncodeToString(hash[:])

	// Compare with stored code_challenge using constant-time comparison to prevent timing attacks
	if len(computedChallenge) != len(codeChallenge) || subtle.ConstantTimeCompare([]byte(computedChallenge), []byte(codeChallenge)) != 1 {
		return ErrInvalidCodeVerifier
	}

	return nil
}

// filterPermissionsForClient filters permissions map to only include permissions for the given client
// This is for internal authorization, not OIDC scopes
func (s *Service) filterPermissionsForClient(allPermissions map[string]uint64, clientID string) map[string]uint64 {
	clientPermissions := make(map[string]uint64)
	for scopeKey, bitmask := range allPermissions {
		// Include if it's for this client (format: "clientID" or "clientID:resource")
		if scopeKey == clientID || strings.HasPrefix(scopeKey, clientID+":") {
			clientPermissions[scopeKey] = bitmask
		}
	}
	return clientPermissions
}

// GetUserInfo returns user information based on requested scopes
// Only returns claims that are allowed by the scopes
func (s *Service) GetUserInfo(userID string, scopes []string) (map[string]any, error) {
	u, err := s.userService.GetUserInfo(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	scopeSet := make(map[string]bool)
	for _, scope := range scopes {
		scopeSet[scope] = true
	}

	claims := make(map[string]any)

	claims["sub"] = u.ID.String()

	// profile scope claims
	if scopeSet["profile"] {
		claims["name"] = strings.TrimSpace(fmt.Sprintf("%s %s", u.FirstName, u.LastName))
		claims["preferred_username"] = u.Username
		claims["given_name"] = u.FirstName
		claims["family_name"] = u.LastName
		claims["created_at"] = u.CreatedAt
		claims["updated_at"] = u.UpdatedAt
		claims["active"] = u.IsActive
	}

	if scopeSet["email"] {
		if u.Email != "" {
			claims["email"] = u.Email
			claims["email_verified"] = false
		}
	}

	return claims, nil
}
