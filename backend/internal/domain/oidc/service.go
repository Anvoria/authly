package oidc

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/Anvoria/authly/internal/domain/auth"
	"github.com/Anvoria/authly/internal/domain/permission"
	svc "github.com/Anvoria/authly/internal/domain/service"
	"github.com/Anvoria/authly/internal/domain/session"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ServiceInterface defines the interface for OIDC operations
type ServiceInterface interface {
	Authorize(req *AuthorizeRequest, userID uuid.UUID) (*AuthorizeResponse, error)
	ExchangeCode(req *TokenRequest, sessionID uuid.UUID, refreshSecret string) (*TokenResponse, error)
}

// Service handles OIDC operations
type Service struct {
	serviceRepo       svc.Repository
	codeRepo          Repository
	codeLifetime      time.Duration
	authService       *auth.Service
	sessionService    session.Service
	permissionService permission.ServiceInterface
}

// NewService creates a new OIDC service
func NewService(serviceRepo svc.Repository, codeRepo Repository, authService *auth.Service, sessionService session.Service, permissionService permission.ServiceInterface) ServiceInterface {
	return &Service{
		serviceRepo:       serviceRepo,
		codeRepo:          codeRepo,
		codeLifetime:      10 * time.Minute,
		authService:       authService,
		sessionService:    sessionService,
		permissionService: permissionService,
	}
}

// Authorize validates the authorization request and generates an authorization code
func (s *Service) Authorize(req *AuthorizeRequest, userID uuid.UUID) (*AuthorizeResponse, error) {
	// Validate response_type
	if req.ResponseType != "code" {
		return nil, ErrInvalidResponseType
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

	// Validate redirect_uri
	if !s.isValidRedirectURI(service.RedirectURIs, req.RedirectURI) {
		return nil, ErrInvalidRedirectURI
	}

	// Validate scopes
	requestedScopes := strings.Fields(req.Scope)
	if !s.isValidScopes(service.AllowedScopes, requestedScopes) {
		return nil, ErrInvalidScope
	}

	// Validate PKCE if provided
	if req.CodeChallenge != "" {
		if err := s.validatePKCE(req.CodeChallenge, req.CodeChallengeMethod); err != nil {
			return nil, err
		}
	}

	// Generate authorization code
	code, err := s.generateAuthorizationCode()
	if err != nil {
		return nil, fmt.Errorf("failed to generate authorization code: %w", err)
	}

	// Create authorization code record
	authCode := &AuthorizationCode{
		Code:          code,
		ClientID:      req.ClientID,
		UserID:        userID,
		RedirectURI:   req.RedirectURI,
		Scopes:        strings.Join(requestedScopes, " "),
		CodeChallenge: req.CodeChallenge,
		ChallengeMeth: req.CodeChallengeMethod,
		ExpiresAt:     time.Now().Add(s.codeLifetime),
		Used:          false,
	}

	if err := s.codeRepo.Create(authCode); err != nil {
		return nil, fmt.Errorf("failed to save authorization code: %w", err)
	}

	return &AuthorizeResponse{
		Code:  code,
		State: req.State,
	}, nil
}

// isValidRedirectURI checks if the redirect_uri is allowed for the service
func (s *Service) isValidRedirectURI(allowedURIs []string, redirectURI string) bool {
	return slices.Contains(allowedURIs, redirectURI)
}

// isValidScopes checks if all requested scopes are allowed
func (s *Service) isValidScopes(allowedScopes []string, requestedScopes []string) bool {
	allowedMap := make(map[string]bool)
	for _, scope := range allowedScopes {
		allowedMap[scope] = true
	}

	for _, scope := range requestedScopes {
		if !allowedMap[scope] {
			return false
		}
	}
	return true
}

// validatePKCE validates the PKCE parameters
func (s *Service) validatePKCE(codeChallenge, codeChallengeMethod string) error {
	if codeChallenge == "" {
		return ErrInvalidCodeChallenge
	}

	if codeChallengeMethod != "S256" {
		return ErrInvalidCodeChallengeMethod
	}

	// Validate code_challenge format (base64url encoded SHA256 hash)
	// Should be 43 characters (base64url encoded 32-byte hash)
	if len(codeChallenge) != 43 {
		return ErrInvalidCodeChallenge
	}

	_, err := base64.RawURLEncoding.DecodeString(codeChallenge)
	if err != nil {
		return ErrInvalidCodeChallenge
	}

	return nil
}

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

	// Validate client_secret if provided (for confidential clients)
	if req.ClientSecret != "" {
		if service.ClientSecret != req.ClientSecret {
			return nil, ErrInvalidClientSecret
		}
	}

	if slices.Contains(strings.Fields(authCode.Scopes), "openid") {
		// TODO: Generate ID token if openid scope is present
		slog.Error("OpenID scope is not supported yet")
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
		return nil, fmt.Errorf("failed to mark code as used: %w", err)
	}

	oidcScopes := strings.Fields(authCode.Scopes)

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
		RefreshToken: refreshSecret,
		Scope:        authCode.Scopes,
	}, nil
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

// verifyCodeVerifier verifies code_verifier against code_challenge using the specified method
func (s *Service) verifyCodeVerifier(codeVerifier, codeChallenge, method string) error {
	if method != "S256" {
		return ErrInvalidCodeChallengeMethod
	}

	// Compute SHA256 hash of code_verifier
	hash := sha256.Sum256([]byte(codeVerifier))
	// Encode to base64url
	computedChallenge := base64.RawURLEncoding.EncodeToString(hash[:])

	// Compare with stored code_challenge
	if computedChallenge != codeChallenge {
		return ErrInvalidCodeVerifier
	}

	return nil
}
