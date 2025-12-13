package oidc

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	svc "github.com/Anvoria/authly/internal/domain/service"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ServiceInterface defines the interface for OIDC operations
type ServiceInterface interface {
	Authorize(req *AuthorizeRequest, userID uuid.UUID) (*AuthorizeResponse, error)
}

// Service handles OIDC operations
type Service struct {
	serviceRepo  svc.Repository
	codeRepo     Repository
	codeLifetime time.Duration
}

// NewService creates a new OIDC service
func NewService(serviceRepo svc.Repository, codeRepo Repository) ServiceInterface {
	return &Service{
		serviceRepo:  serviceRepo,
		codeRepo:     codeRepo,
		codeLifetime: 10 * time.Minute,
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
	for _, allowed := range allowedURIs {
		if allowed == redirectURI {
			return true
		}
	}
	return false
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
