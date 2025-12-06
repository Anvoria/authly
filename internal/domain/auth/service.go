package auth

import (
	"errors"
	"time"

	"github.com/Anvoria/authly/internal/domain/permission"
	"github.com/Anvoria/authly/internal/domain/session"
	"github.com/Anvoria/authly/internal/domain/user"
	"github.com/lestrrat-go/jwx/v3/jwt"
	"gorm.io/gorm"
)

// LoginResponse represents the response from a successful login
type LoginResponse struct {
	AccessToken  string             `json:"access_token"`
	RefreshToken string             `json:"refresh_token"`
	RefreshSID   string             `json:"refresh_sid"`
	User         *user.UserResponse `json:"user"`
}

// AuthService defines the interface for authentication operations
type AuthService interface {
	Login(username, password, userAgent, ip string) (*LoginResponse, error)
	Register(req user.RegisterRequest) (*user.UserResponse, error)
	IsTokenRevoked(claims *AccessTokenClaims) (bool, error)
}

// Service handles authentication operations
type Service struct {
	Users             user.Repository
	Sessions          session.Service
	PermissionService permission.ServiceInterface
	KeyStore          *KeyStore
	issuer            string
}

// NewService constructs a Service configured with the provided user repository, session service, permission service, key store, and issuer.
func NewService(users user.Repository, sessions session.Service, permService permission.ServiceInterface, keyStore *KeyStore, issuer string) *Service {
	return &Service{
		Users:             users,
		Sessions:          sessions,
		PermissionService: permService,
		KeyStore:          keyStore,
		issuer:            issuer,
	}
}

// BuildAudience builds audience list from scopes
// Returns list of service codes that user has access to
// BuildAudience extracts unique service codes from the given scope keys.
// Each key is expected in the form "service" or "service:resource"; the returned
// slice contains each service code at most once in no particular order.
func BuildAudience(scopes map[string]uint64) []string {
	audMap := make(map[string]bool)
	for scopeKey := range scopes {
		// Extract service code from scope key (format: "service" or "service:resource")
		serviceCode := scopeKey
		// Find first colon to extract service code
		for i := 0; i < len(scopeKey); i++ {
			if scopeKey[i] == ':' {
				serviceCode = scopeKey[:i]
				break
			}
		}
		if serviceCode != "" {
			audMap[serviceCode] = true
		}
	}

	aud := make([]string, 0, len(audMap))
	for serviceCode := range audMap {
		aud = append(aud, serviceCode)
	}
	return aud
}

func (s *Service) GenerateAccessToken(sub, sid string, scopes map[string]uint64, pver int) (string, error) {
	now := time.Now()
	exp := now.Add(15 * time.Minute)

	// Build audience from scopes
	aud := BuildAudience(scopes)

	// Build token
	token, err := jwt.NewBuilder().
		Subject(sub).
		Audience(aud).
		Issuer(s.issuer).
		IssuedAt(now).
		Expiration(exp).
		Claim("sid", sid).
		Claim("scopes", scopes).
		Claim("pver", pver).
		Build()
	if err != nil {
		return "", err
	}

	claims := &AccessTokenClaims{
		Sid:   sid,
		Token: token,
	}

	return s.KeyStore.Sign(claims)
}

func (s *Service) Login(username, password, userAgent, ip string) (*LoginResponse, error) {
	u, err := s.Users.FindByUsername(username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if !user.VerifyPassword(password, u.Password) {
		return nil, ErrInvalidCredentials
	}

	// Build scopes from user permissions
	scopes, err := s.PermissionService.BuildScopes(u.ID.String())
	if err != nil {
		return nil, err
	}

	// Get permission version (for cache invalidation)
	pver, err := s.PermissionService.GetPermissionVersion(u.ID.String())
	if err != nil {
		// Default to 1 if error
		pver = 1
	}

	sid, secret, err := s.Sessions.Create(u.ID, userAgent, ip, 24*time.Hour)
	if err != nil {
		return nil, err
	}

	access, err := s.GenerateAccessToken(u.ID.String(), sid.String(), scopes, pver)
	if err != nil {
		return nil, err
	}
	return &LoginResponse{
		AccessToken:  access,
		RefreshToken: secret,
		RefreshSID:   sid.String(),
		User:         u.ToResponse(),
	}, nil
}

func (s *Service) Register(req user.RegisterRequest) (*user.UserResponse, error) {
	if req.Email != "" {
		if _, err := s.Users.FindByEmail(req.Email); err == nil {
			return nil, user.ErrEmailExists
		}
	}

	if _, err := s.Users.FindByUsername(req.Username); err == nil {
		return nil, user.ErrUsernameExists
	}

	hashedPassword, err := user.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	newUser := &user.User{
		Username:  req.Username,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Email:     req.Email,
		Password:  hashedPassword,
		IsActive:  true,
	}

	if err := s.Users.Create(newUser); err != nil {
		return nil, err
	}

	return newUser.ToResponse(), nil
}

// IsTokenRevoked checks if a token has been revoked
// This is a stub implementation - in production, this should check Redis/cache
// for revoked tokens (e.g., by jti claim or user:last_logout_at)
func (s *Service) IsTokenRevoked(claims *AccessTokenClaims) (bool, error) {
	// TODO: Implement Redis-based revocation check
	// For now, return false (not revoked)
	// In production, check:
	// 1. Redis key: "token:revoked:{jti}" or "user:logout:{userID}"
	// 2. Compare token issued_at with user's last_logout_at
	return false, nil
}