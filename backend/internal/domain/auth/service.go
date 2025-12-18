package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
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

// GenerateAccessToken generates an OIDC-compliant access token
// scopes: OIDC scope strings (e.g., ["openid", "profile]")
// audience: resource server identifier (e.g., "api:clientID" or clientID)
// permissions: optional permissions map for internal authorization
func (s *Service) GenerateAccessToken(sub, sid string, scopes []string, audience string, permissions map[string]uint64, pver int) (string, error) {
	now := time.Now()
	exp := now.Add(15 * time.Minute)

	accessTokenScopes := filterAccessTokenScopes(scopes)

	token, err := jwt.NewBuilder().
		Subject(sub).
		Audience([]string{audience}).
		Issuer(s.issuer).
		IssuedAt(now).
		Expiration(exp).
		Claim("sid", sid).
		Claim("scope", strings.Join(accessTokenScopes, " ")). // Scopes for access token (filtered)
		Claim("requested_scopes", strings.Join(scopes, " ")). // All requested scopes (for userinfo)
		Claim("pver", pver).
		Build()
	if err != nil {
		return "", err
	}

	if len(permissions) > 0 {
		if err := token.Set("permissions", permissions); err != nil {
			return "", fmt.Errorf("failed to set permissions claim: %w", err)
		}
	}

	claims := &AccessTokenClaims{
		Sid:   sid,
		Token: token,
	}

	return s.KeyStore.Sign(claims)
}

// filterAccessTokenScopes removes OIDC scopes that don't belong in access token
// filterAccessTokenScopes filters out OIDC scopes that are intended for ID tokens or userinfo ("openid", "profile", "email") from the provided scope list.
// It returns a new slice containing only the scopes appropriate for inclusion in an access token.
func filterAccessTokenScopes(scopes []string) []string {
	filtered := make([]string, 0, len(scopes))
	for _, scope := range scopes {
		if scope != "openid" && scope != "profile" && scope != "email" {
			filtered = append(filtered, scope)
		}
	}
	return filtered
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

	sid, secret, err := s.Sessions.Create(u.ID, userAgent, ip, 24*time.Hour)
	if err != nil {
		return nil, err
	}

	return &LoginResponse{
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
