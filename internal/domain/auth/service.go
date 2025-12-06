package auth

import (
	"time"

	"github.com/Anvoria/authly/internal/domain/session"
	"github.com/Anvoria/authly/internal/domain/user"
	"github.com/golang-jwt/jwt/v5"
)

// LoginResponse represents the response from a successful login
type LoginResponse struct {
	AccessToken  string     `json:"access_token"`
	RefreshToken string     `json:"refresh_token"`
	RefreshSID   string     `json:"refresh_sid"`
	User         *user.User `json:"user"`
}

// Service handles authentication operations
type Service struct {
	Users    user.Repository
	Sessions session.Service
	KeyStore *KeyStore
	issuer   string
}

// NewService creates a new auth service
func NewService(users user.Repository, sessions session.Service, keyStore *KeyStore, issuer string) *Service {
	return &Service{
		Users:    users,
		Sessions: sessions,
		KeyStore: keyStore,
		issuer:   issuer,
	}
}

func (s *Service) GenerateAccessToken(sub, sid string, aud []string) (string, error) {
	claims := &AccessTokenClaims{
		Sid: sid,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   sub,
			Audience:  aud,
			Issuer:    s.issuer,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
		},
	}
	return s.KeyStore.Sign(claims)
}

func (s *Service) Login(email, password, userAgent, ip string) (*LoginResponse, error) {
	u, err := s.Users.GetByEmail(email)
	if err != nil {
		return nil, err
	}

	if !user.VerifyPassword(password, u.Password) {
		return nil, ErrInvalidCredentials
	}

	// TODO: Scopes

	sid, secret, err := s.Sessions.Create(u.ID, userAgent, ip, 24*time.Hour)
	if err != nil {
		return nil, err
	}

	access, err := s.GenerateAccessToken(u.ID.String(), sid.String(), []string{"api"})
	if err != nil {
		return nil, err
	}
	return &LoginResponse{
		AccessToken:  access,
		RefreshToken: secret,
		RefreshSID:   sid.String(),
		User:         u,
	}, nil
}
