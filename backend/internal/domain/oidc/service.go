package oidc

import (
	"errors"
	"fmt"
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
	RefreshToken(req *TokenRequest) (*TokenResponse, error)
	ClientCredentialsGrant(req *TokenRequest) (*TokenResponse, error)
	PasswordGrant(req *TokenRequest) (*TokenResponse, error)
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

// NewService creates a new ServiceInterface wired with the provided repositories and supporting services.
// The returned Service is configured with a 10-minute authorization code lifetime.
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
		Nonce:         req.Nonce,
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
