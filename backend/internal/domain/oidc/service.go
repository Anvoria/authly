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
	"github.com/Anvoria/authly/internal/domain/user"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ValidateAuthorizationRequestResponse represents the response from authorization request validation
type ValidateAuthorizationRequestResponse struct {
	Valid            bool        `json:"valid"`
	Client           *ClientInfo `json:"client,omitempty"`
	Error            string      `json:"error,omitempty"`
	ErrorDescription string      `json:"error_description,omitempty"`
}

// ClientInfo represents client information for validation response
type ClientInfo struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	LogoURL       *string  `json:"logo_url,omitempty"`
	RedirectURIs  []string `json:"redirect_uris"`
	AllowedScopes []string `json:"allowed_scopes"`
	Active        bool     `json:"active"`
}

// ServiceInterface defines the interface for OIDC operations
type ServiceInterface interface {
	Authorize(req *AuthorizeRequest, userID uuid.UUID) (*AuthorizeResponse, error)
	ExchangeCode(req *TokenRequest, sessionID uuid.UUID, refreshSecret string) (*TokenResponse, error)
	GetUserInfo(userID string, scopes []string) (map[string]interface{}, error)
	ValidateAuthorizationRequest(req *AuthorizeRequest) *ValidateAuthorizationRequestResponse
}

// Service handles OIDC operations
type Service struct {
	serviceRepo       svc.Repository
	codeRepo          Repository
	codeLifetime      time.Duration
	authService       *auth.Service
	sessionService    session.Service
	permissionService permission.ServiceInterface
	userService       user.Service
}

// NewService creates a new OIDC service
func NewService(serviceRepo svc.Repository, codeRepo Repository, authService *auth.Service, sessionService session.Service, permissionService permission.ServiceInterface, userService user.Service) ServiceInterface {
	return &Service{
		serviceRepo:       serviceRepo,
		codeRepo:          codeRepo,
		codeLifetime:      10 * time.Minute,
		authService:       authService,
		sessionService:    sessionService,
		permissionService: permissionService,
		userService:       userService,
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

	if codeChallengeMethod != "s256" && codeChallengeMethod != "S256" && codeChallengeMethod != "" {
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
	if method != "s256" && method != "S256" {
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

// GetUserInfo returns user information based on requested scopes
// Only returns claims that are allowed by the scopes
func (s *Service) GetUserInfo(userID string, scopes []string) (map[string]interface{}, error) {
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
		claims["name"] = fmt.Sprintf("%s %s", u.FirstName, u.LastName)
		claims["preferred_username"] = u.Username
		claims["given_name"] = u.FirstName
		claims["family_name"] = u.LastName
	}

	if scopeSet["email"] {
		if u.Email != "" {
			claims["email"] = u.Email
			claims["email_verified"] = false
		}
	}

	return claims, nil
}

// ValidateAuthorizationRequest validates an OAuth2/OIDC authorization request without requiring authentication
// Returns a response indicating if the request is valid and includes client information if valid
func (s *Service) ValidateAuthorizationRequest(req *AuthorizeRequest) *ValidateAuthorizationRequestResponse {
	// Validate response_type
	if req.ResponseType == "" {
		return &ValidateAuthorizationRequestResponse{
			Valid:            false,
			Error:            "invalid_request",
			ErrorDescription: "response_type is required",
		}
	}
	if req.ResponseType != "code" {
		return &ValidateAuthorizationRequestResponse{
			Valid:            false,
			Error:            "unsupported_response_type",
			ErrorDescription: "Only 'code' response_type is supported",
		}
	}

	// Validate client_id
	if req.ClientID == "" {
		return &ValidateAuthorizationRequestResponse{
			Valid:            false,
			Error:            "invalid_request",
			ErrorDescription: "client_id is required",
		}
	}

	// Find service by client_id
	service, err := s.serviceRepo.FindByClientID(req.ClientID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &ValidateAuthorizationRequestResponse{
				Valid:            false,
				Error:            "invalid_client",
				ErrorDescription: "Client not found",
			}
		}
		return &ValidateAuthorizationRequestResponse{
			Valid:            false,
			Error:            "server_error",
			ErrorDescription: "Failed to validate client",
		}
	}

	// Check if service is active
	if !service.Active {
		return &ValidateAuthorizationRequestResponse{
			Valid:            false,
			Error:            "unauthorized_client",
			ErrorDescription: "Client is not active",
			Client: &ClientInfo{
				ID:            service.ID.String(),
				Name:          service.Name,
				RedirectURIs:  service.RedirectURIs,
				AllowedScopes: service.AllowedScopes,
				Active:        service.Active,
			},
		}
	}

	// Validate redirect_uri
	if req.RedirectURI == "" {
		return &ValidateAuthorizationRequestResponse{
			Valid:            false,
			Error:            "invalid_request",
			ErrorDescription: "redirect_uri is required",
			Client: &ClientInfo{
				ID:            service.ID.String(),
				Name:          service.Name,
				RedirectURIs:  service.RedirectURIs,
				AllowedScopes: service.AllowedScopes,
				Active:        service.Active,
			},
		}
	}
	if !s.isValidRedirectURI(service.RedirectURIs, req.RedirectURI) {
		return &ValidateAuthorizationRequestResponse{
			Valid:            false,
			Error:            "invalid_redirect_uri",
			ErrorDescription: "The redirect_uri is not allowed for this client",
			Client: &ClientInfo{
				ID:            service.ID.String(),
				Name:          service.Name,
				RedirectURIs:  service.RedirectURIs,
				AllowedScopes: service.AllowedScopes,
				Active:        service.Active,
			},
		}
	}

	// Validate scopes
	if req.Scope == "" {
		return &ValidateAuthorizationRequestResponse{
			Valid:            false,
			Error:            "invalid_request",
			ErrorDescription: "scope is required",
			Client: &ClientInfo{
				ID:            service.ID.String(),
				Name:          service.Name,
				RedirectURIs:  service.RedirectURIs,
				AllowedScopes: service.AllowedScopes,
				Active:        service.Active,
			},
		}
	}
	requestedScopes := strings.Fields(req.Scope)
	if !s.isValidScopes(service.AllowedScopes, requestedScopes) {
		return &ValidateAuthorizationRequestResponse{
			Valid:            false,
			Error:            "invalid_scope",
			ErrorDescription: "One or more requested scopes are not allowed",
			Client: &ClientInfo{
				ID:            service.ID.String(),
				Name:          service.Name,
				RedirectURIs:  service.RedirectURIs,
				AllowedScopes: service.AllowedScopes,
				Active:        service.Active,
			},
		}
	}

	// Validate PKCE if provided
	if req.CodeChallenge != "" {
		if err := s.validatePKCE(req.CodeChallenge, req.CodeChallengeMethod); err != nil {
			var errorCode string
			var errorDesc string
			switch err {
			case ErrInvalidCodeChallenge:
				errorCode = "invalid_code_challenge"
				errorDesc = "Invalid code_challenge format"
			case ErrInvalidCodeChallengeMethod:
				errorCode = "unsupported_code_challenge_method"
				errorDesc = "Only 's256' or 'S256' code_challenge_method is supported"
			default:
				errorCode = "invalid_request"
				errorDesc = err.Error()
			}
			return &ValidateAuthorizationRequestResponse{
				Valid:            false,
				Error:            errorCode,
				ErrorDescription: errorDesc,
				Client: &ClientInfo{
					ID:            service.ID.String(),
					Name:          service.Name,
					RedirectURIs:  service.RedirectURIs,
					AllowedScopes: service.AllowedScopes,
					Active:        service.Active,
				},
			}
		}
	}

	// All validations passed
	return &ValidateAuthorizationRequestResponse{
		Valid: true,
		Client: &ClientInfo{
			ID:            service.ID.String(),
			Name:          service.Name,
			RedirectURIs:  service.RedirectURIs,
			AllowedScopes: service.AllowedScopes,
			Active:        service.Active,
		},
	}
}
