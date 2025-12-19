package oidc

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/Anvoria/authly/internal/domain/session"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ExchangeCode exchanges an authorization code for access and refresh tokens
// sessionID and refreshSecret should come from the existing session cookie
func (s *Service) ExchangeCode(req *TokenRequest, sessionID uuid.UUID, refreshSecret string) (*TokenResponse, error) {
	// Validate grant_type
	if req.GrantType != "authorization_code" {
		return nil, ErrInvalidGrant
	}

	// Find and validate authorization code
	authCode, err := s.codeRepo.FindByCode(req.Code)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidCode
		}
		return nil, fmt.Errorf("failed to find authorization code: %w", err)
	}

	// Check if code is already used
	if authCode.Used {
		return nil, ErrInvalidCode
	}

	// Check if code is expired
	if time.Now().After(authCode.ExpiresAt) {
		return nil, ErrInvalidCode
	}

	// Find service by client_id
	service, err := s.serviceRepo.FindByClientID(req.ClientID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidClientID
		}
		return nil, fmt.Errorf("failed to find service: %w", err)
	}

	// Validate client_id matches
	if authCode.ClientID != req.ClientID {
		return nil, ErrInvalidClientID
	}

	// Validate redirect_uri matches
	if authCode.RedirectURI != req.RedirectURI {
		return nil, ErrInvalidRedirectURI
	}

	// Validate client_secret for confidential clients
	if service.ClientSecret != "" {
		if req.ClientSecret != service.ClientSecret {
			return nil, ErrInvalidClientSecret
		}
	}

	// Validate PKCE if code_challenge was provided
	if authCode.CodeChallenge != "" {
		if req.CodeVerifier == "" {
			return nil, ErrInvalidCodeVerifier
		}

		// Verify code_verifier against code_challenge
		if err := s.verifyCodeVerifier(req.CodeVerifier, authCode.CodeChallenge, authCode.ChallengeMeth); err != nil {
			return nil, err
		}
	}

	sess, err := s.sessionService.Validate(sessionID, refreshSecret)
	if err != nil {
		return nil, fmt.Errorf("invalid session: %w", err)
	}

	userIDFromSession, err := uuid.Parse(sess.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid session user ID: %w", err)
	}

	if userIDFromSession != authCode.UserID {
		return nil, fmt.Errorf("session user mismatch: session belongs to different user")
	}

	// Mark code as used
	if err := s.codeRepo.MarkAsUsed(req.Code); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidCode
		}
		return nil, fmt.Errorf("failed to mark code as used: %w", err)
	}

	oidcScopes := strings.Fields(authCode.Scopes)

	// Update session with granted scopes (binds session to these OIDC scopes)
	if err := s.sessionService.UpdateScopes(sessionID, oidcScopes); err != nil {
		return nil, fmt.Errorf("failed to update session scopes: %w", err)
	}

	// Generate ID Token if openid scope is present
	var idToken string
	if slices.Contains(oidcScopes, "openid") {
		userInfo, err := s.GetUserInfo(authCode.UserID.String(), oidcScopes)
		if err != nil {
			return nil, fmt.Errorf("failed to get user info for id token: %w", err)
		}

		idToken, err = s.authService.GenerateIDToken(
			authCode.UserID.String(),
			req.ClientID,
			authCode.Nonce,
			sess.CreatedAt,
			userInfo,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to generate id token: %w", err)
		}
	}

	permissions, err := s.permissionService.BuildScopes(authCode.UserID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to build permissions: %w", err)
	}

	clientPermissions := s.filterPermissionsForClient(permissions, req.ClientID)

	pver, err := s.permissionService.GetPermissionVersion(authCode.UserID.String())
	if err != nil {
		pver = 1
	}

	audience := req.ClientID

	accessToken, err := s.authService.GenerateAccessToken(
		authCode.UserID.String(),
		sessionID.String(),
		oidcScopes,
		audience,
		clientPermissions,
		pver,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	return &TokenResponse{
		AccessToken:  accessToken,
		TokenType:    "Bearer",
		ExpiresIn:    900, // 15 minutes in seconds
		RefreshToken: fmt.Sprintf("%s:%s", sessionID.String(), refreshSecret),
		Scope:        authCode.Scopes,
		IDToken:      idToken,
	}, nil
}

// RefreshToken refreshes the access token using a refresh token
func (s *Service) RefreshToken(req *TokenRequest) (*TokenResponse, error) {
	// Validate grant_type
	if req.GrantType != "refresh_token" {
		return nil, ErrInvalidGrant
	}

	// Parse refresh_token (format: sessionID:secret)
	parts := strings.SplitN(req.RefreshToken, ":", 2)
	if len(parts) != 2 {
		return nil, ErrInvalidGrant
	}
	sessionIDStr := parts[0]
	refreshSecret := parts[1]

	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		return nil, ErrInvalidGrant
	}

	// Find service by client_id
	service, err := s.serviceRepo.FindByClientID(req.ClientID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidClientID
		}
		return nil, fmt.Errorf("failed to find service: %w", err)
	}

	// Check if service is active
	if !service.Active {
		return nil, ErrClientNotActive
	}

	// Validate client_secret if provided
	if service.ClientSecret != "" && req.ClientSecret != "" {
		if service.ClientSecret != req.ClientSecret {
			return nil, ErrInvalidClientSecret
		}
	}

	// Validate session first to get UserID and GrantedScopes
	sess, err := s.sessionService.Validate(sessionID, refreshSecret)
	if err != nil {
		if errors.Is(err, session.ErrInvalidSession) || errors.Is(err, session.ErrInvalidSecret) || errors.Is(err, session.ErrExpiredSession) {
			return nil, ErrInvalidGrant
		}
		return nil, fmt.Errorf("failed to validate session: %w", err)
	}

	// Validate Scopes against originally granted scopes (RFC 6749 Section 6)
	var requestedScopes []string
	grantedScopes := strings.Fields(sess.GrantedScopes)

	if req.Scope != "" {
		requestedScopes = strings.Fields(req.Scope)
		// Check if requested scopes are a subset of granted scopes (Downscoping)
		if !s.isValidScopes(grantedScopes, requestedScopes) {
			return nil, ErrInvalidScope
		}
	} else {
		// If no scope requested, default to originally granted scopes
		requestedScopes = grantedScopes
	}

	userID, err := uuid.Parse(sess.UserID)
	if err != nil {
		return nil, ErrInvalidGrant
	}

	// Rotate session (Refresh Token Rotation)
	// We use a default TTL of 7 days (168 hours) for refreshed sessions
	newSecret, err := s.sessionService.Rotate(sessionID, refreshSecret, 168*time.Hour)
	if err != nil {
		if errors.Is(err, session.ErrReplayDetected) {
			// Revoke session if replay detected
			_ = s.sessionService.Revoke(sessionID)
			return nil, ErrInvalidGrant
		}
		return nil, ErrInvalidGrant
	}

	permissions, err := s.permissionService.BuildScopes(userID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to build permissions: %w", err)
	}

	clientPermissions := s.filterPermissionsForClient(permissions, req.ClientID)

	pver, err := s.permissionService.GetPermissionVersion(userID.String())
	if err != nil {
		pver = 1
	}

	audience := req.ClientID

	accessToken, err := s.authService.GenerateAccessToken(
		userID.String(),
		sessionID.String(),
		requestedScopes,
		audience,
		clientPermissions,
		pver,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	return &TokenResponse{
		AccessToken:  accessToken,
		TokenType:    "Bearer",
		ExpiresIn:    900, // 15 minutes in seconds
		RefreshToken: fmt.Sprintf("%s:%s", sessionID.String(), newSecret),
		Scope:        strings.Join(requestedScopes, " "),
	}, nil
}

// ClientCredentialsGrant handles the client credentials flow (Machine-to-Machine)
func (s *Service) ClientCredentialsGrant(req *TokenRequest) (*TokenResponse, error) {
	if req.GrantType != "client_credentials" {
		return nil, ErrInvalidGrant
	}

	// Find service by client_id
	service, err := s.serviceRepo.FindByClientID(req.ClientID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidClientID
		}
		return nil, fmt.Errorf("failed to find service: %w", err)
	}

	// Check if service is active
	if !service.Active {
		return nil, ErrClientNotActive
	}

	// Validate client_secret (mandatory for client_credentials)
	if service.ClientSecret == "" || req.ClientSecret != service.ClientSecret {
		return nil, ErrInvalidClientSecret
	}

	// Validate scopes
	// For client_credentials, scopes must be pre-registered (AllowedScopes)
	requestedScopes := strings.Fields(req.Scope)
	if len(requestedScopes) > 0 {
		if !s.isValidScopes(service.AllowedScopes, requestedScopes) {
			return nil, ErrInvalidScope
		}
	} else {
		// Default to allowed scopes if none requested
		requestedScopes = service.AllowedScopes
	}

	// For Client Credentials, the "user" is the Service itself.
	// We use the Service ID as the Subject (sub).
	subject := service.ID.String()

	// Build permissions based on the Service/Client ID
	// For Client Credentials, the "user" is the Service itself.
	permissions, err := s.permissionService.BuildServiceScopes(req.ClientID)
	if err != nil {
		return nil, fmt.Errorf("failed to build service permissions: %w", err)
	}

	// Client Credentials tokens usually don't have Refresh Tokens.

	accessToken, err := s.authService.GenerateAccessToken(
		subject,
		"service-session", // No interactive session
		requestedScopes,
		req.ClientID,
		permissions,
		1,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	return &TokenResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   3600, // 1 hour
		Scope:       strings.Join(requestedScopes, " "),
	}, nil
}

// PasswordGrant handles the resource owner password credentials flow
func (s *Service) PasswordGrant(req *TokenRequest) (*TokenResponse, error) {
	if req.GrantType != "password" {
		return nil, ErrInvalidGrant
	}

	if req.Username == "" || req.Password == "" {
		return nil, ErrInvalidGrant // Missing credentials
	}

	// Authenticate User
	u, err := s.userService.FindByUsername(req.Username)
	if err != nil {
		// Avoid leaking user existence
		return nil, ErrInvalidGrant
	}

	if !s.userService.VerifyPassword(u, req.Password) {
		return nil, ErrInvalidGrant
	}

	if !u.IsActive {
		return nil, ErrInvalidGrant // User disabled
	}

	// Validate Client
	service, err := s.serviceRepo.FindByClientID(req.ClientID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidClientID
		}
		return nil, fmt.Errorf("failed to find service: %w", err)
	}

	// Check if service is active
	if !service.Active {
		return nil, ErrClientNotActive
	}

	// Validate client_secret for confidential clients
	if service.ClientSecret != "" {
		if req.ClientSecret != service.ClientSecret {
			return nil, ErrInvalidClientSecret
		}
	}

	// Validate Scopes
	requestedScopes := strings.Fields(req.Scope)
	if len(requestedScopes) > 0 {
		if !s.isValidScopes(service.AllowedScopes, requestedScopes) {
			return nil, ErrInvalidScope
		}
	} else {
		requestedScopes = service.AllowedScopes
	}

	// Create a new session (Password grant acts like a login)
	sessionID, secret, err := s.sessionService.Create(u.ID, req.UserAgent, req.IPAddress, requestedScopes, 24*time.Hour)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}
	// Generate ID Token if openid scope is present
	var idToken string
	if slices.Contains(requestedScopes, "openid") {
		userInfo, err := s.GetUserInfo(u.ID.String(), requestedScopes)
		if err != nil {
			return nil, fmt.Errorf("failed to get user info: %w", err)
		}

		idToken, err = s.authService.GenerateIDToken(
			u.ID.String(),
			req.ClientID,
			"", // No nonce in password flow
			time.Now(),
			userInfo,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to generate id token: %w", err)
		}
	}

	// Permissions
	permissions, err := s.permissionService.BuildScopes(u.ID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to build permissions: %w", err)
	}
	clientPermissions := s.filterPermissionsForClient(permissions, req.ClientID)

	pver, err := s.permissionService.GetPermissionVersion(u.ID.String())
	if err != nil {
		pver = 1
	}

	accessToken, err := s.authService.GenerateAccessToken(
		u.ID.String(),
		sessionID.String(),
		requestedScopes,
		req.ClientID,
		clientPermissions,
		pver,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	return &TokenResponse{
		AccessToken:  accessToken,
		TokenType:    "Bearer",
		ExpiresIn:    900,
		RefreshToken: fmt.Sprintf("%s:%s", sessionID.String(), secret),
		Scope:        strings.Join(requestedScopes, " "),
		IDToken:      idToken,
	}, nil
}
