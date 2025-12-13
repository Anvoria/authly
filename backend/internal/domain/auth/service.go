package auth

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/Anvoria/authly/internal/cache"
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
	revocationCache   *cache.TokenRevocationCache
}

// NewService constructs a Service configured with the provided user repository, session service, permission service, key store, issuer, and revocation cache.
func NewService(users user.Repository, sessions session.Service, permService permission.ServiceInterface, keyStore *KeyStore, issuer string, revocationCache *cache.TokenRevocationCache) *Service {
	return &Service{
		Users:             users,
		Sessions:          sessions,
		PermissionService: permService,
		KeyStore:          keyStore,
		issuer:            issuer,
		revocationCache:   revocationCache,
	}
}

// BuildAudience builds audience list from scopes
// Returns list of client_ids that user has access to
// BuildAudience extracts unique client_ids from the given scope keys.
// Each key is expected in the form "client_id" or "client_id:resource"; the returned
// slice contains each client_id at most once in no particular order.
func BuildAudience(scopes map[string]uint64) []string {
	audMap := make(map[string]bool)
	for scopeKey := range scopes {
		// Extract client_id from scope key (format: "client_id" or "client_id:resource")
		clientID := scopeKey
		// Find first colon to extract client_id
		for i := 0; i < len(scopeKey); i++ {
			if scopeKey[i] == ':' {
				clientID = scopeKey[:i]
				break
			}
		}
		if clientID != "" {
			audMap[clientID] = true
		}
	}

	aud := make([]string, 0, len(audMap))
	for clientID := range audMap {
		aud = append(aud, clientID)
	}
	return aud
}

func (s *Service) GenerateAccessToken(sub, sid string, scopes map[string]uint64, pver int) (string, error) {
	now := time.Now()
	exp := now.Add(15 * time.Minute)

	// Build audience from scopes
	// aud := BuildAudience(scopes)
	aud := []string{"authly_authly_00000000"}

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

	if req.Username == "" {
		return nil, user.ErrUsernameRequired
	}

	if req.Password == "" {
		return nil, user.ErrPasswordRequired
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

// IsTokenRevoked checks if a token has been revoked by checking Redis cache
// It uses the session ID (sid) from the token claims to check if the session is revoked
func (s *Service) IsTokenRevoked(claims *AccessTokenClaims) (bool, error) {
	if s.revocationCache == nil {
		slog.Warn("Token revocation cache not available, skipping revocation check")
		return false, nil
	}

	sessionID := claims.GetSid()
	if sessionID == "" {
		return false, nil
	}

	ctx := context.Background()
	revoked, err := s.revocationCache.IsSessionRevoked(ctx, sessionID)
	if err != nil {
		slog.Warn("Failed to check token revocation in Redis", "error", err, "session_id", sessionID)
		return false, nil
	}

	return revoked, nil
}
