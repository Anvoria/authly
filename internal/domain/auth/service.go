package auth

import (
	"time"

	"github.com/Anvoria/authly/internal/domain/session"
	"github.com/Anvoria/authly/internal/domain/user"
	"github.com/lestrrat-go/jwx/v3/jwt"
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
	now := time.Now()
	exp := now.Add(15 * time.Minute)

	// Build token
	token, err := jwt.NewBuilder().
		Subject(sub).
		Audience(aud).
		Issuer(s.issuer).
		IssuedAt(now).
		Expiration(exp).
		Claim("sid", sid).
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
		return nil, ErrInvalidCredentials
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
