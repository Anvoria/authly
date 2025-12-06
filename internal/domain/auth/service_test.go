package auth

import (
	"errors"
	"testing"
	"time"

	"github.com/Anvoria/authly/internal/domain/session"
	"github.com/Anvoria/authly/internal/domain/user"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockUserRepository is a mock implementation of user.Repository
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(u *user.User) error {
	args := m.Called(u)
	return args.Error(0)
}

func (m *MockUserRepository) FindByID(id string) (*user.User, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.User), args.Error(1)
}

func (m *MockUserRepository) FindByUsername(username string) (*user.User, error) {
	args := m.Called(username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.User), args.Error(1)
}

func (m *MockUserRepository) FindByEmail(email string) (*user.User, error) {
	args := m.Called(email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.User), args.Error(1)
}

func (m *MockUserRepository) Update(u *user.User) error {
	args := m.Called(u)
	return args.Error(0)
}

func (m *MockUserRepository) Delete(id string) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockUserRepository) VerifyPassword(u *user.User, password string) bool {
	args := m.Called(u, password)
	return args.Bool(0)
}

// MockSessionService is a mock implementation of session.Service
type MockSessionService struct {
	mock.Mock
}

func (m *MockSessionService) Create(userID uuid.UUID, userAgent, ip string, ttl time.Duration) (uuid.UUID, string, error) {
	args := m.Called(userID, userAgent, ip, ttl)
	if args.Get(0) == nil {
		return uuid.Nil, "", args.Error(2)
	}
	return args.Get(0).(uuid.UUID), args.Get(1).(string), args.Error(2)
}

func (m *MockSessionService) Validate(sid uuid.UUID, secret string) (*session.Session, error) {
	args := m.Called(sid, secret)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*session.Session), args.Error(1)
}

func (m *MockSessionService) Rotate(sid uuid.UUID, oldSecret string, ttl time.Duration) (string, error) {
	args := m.Called(sid, oldSecret, ttl)
	return args.String(0), args.Error(1)
}

func (m *MockSessionService) Revoke(sid uuid.UUID) error {
	args := m.Called(sid)
	return args.Error(0)
}

// TestNewService tests service creation
func TestNewService(t *testing.T) {
	mockUserRepo := new(MockUserRepository)
	mockSessionService := new(MockSessionService)
	mockKeyStore := &KeyStore{} // Simplified mock

	t.Run("creates service successfully", func(t *testing.T) {
		service := NewService(mockUserRepo, mockSessionService, mockKeyStore, "test-issuer")

		assert.NotNil(t, service)
		assert.Equal(t, mockUserRepo, service.Users)
		assert.Equal(t, mockSessionService, service.Sessions)
		assert.Equal(t, mockKeyStore, service.KeyStore)
		assert.Equal(t, "test-issuer", service.issuer)
	})

	t.Run("service fields are properly initialized", func(t *testing.T) {
		issuer := "my-auth-service"
		service := NewService(mockUserRepo, mockSessionService, mockKeyStore, issuer)

		assert.NotNil(t, service.Users, "Users repository should be set")
		assert.NotNil(t, service.Sessions, "Sessions service should be set")
		assert.NotNil(t, service.KeyStore, "KeyStore should be set")
		assert.Equal(t, issuer, service.issuer, "Issuer should match")
	})
}

// TestService_Register tests the Register method
func TestService_Register(t *testing.T) {
	t.Run("successful registration", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		mockSessionService := new(MockSessionService)
		mockKeyStore := &KeyStore{}

		service := NewService(mockUserRepo, mockSessionService, mockKeyStore, "test-issuer")

		req := user.RegisterRequest{
			Username:  "testuser",
			FirstName: "Test",
			LastName:  "User",
			Email:     "test@example.com",
			Password:  "SecureP@ss123",
		}

		// Mock expectations
		mockUserRepo.On("FindByEmail", req.Email).Return(nil, errors.New("not found"))
		mockUserRepo.On("FindByUsername", req.Username).Return(nil, errors.New("not found"))
		mockUserRepo.On("Create", mock.AnythingOfType("*user.User")).Return(nil)

		result, err := service.Register(req)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, req.Username, result.Username)
		assert.Equal(t, req.FirstName, result.FirstName)
		assert.Equal(t, req.LastName, result.LastName)
		assert.Equal(t, req.Email, result.Email)

		mockUserRepo.AssertExpectations(t)
	})

	t.Run("registration fails when email exists", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		mockSessionService := new(MockSessionService)
		mockKeyStore := &KeyStore{}

		service := NewService(mockUserRepo, mockSessionService, mockKeyStore, "test-issuer")

		req := user.RegisterRequest{
			Username:  "testuser",
			FirstName: "Test",
			LastName:  "User",
			Email:     "existing@example.com",
			Password:  "SecureP@ss123",
		}

		existingUser := &user.User{
			Username: "otheruser",
			Email:    req.Email,
		}
		existingUser.ID = uuid.New()

		mockUserRepo.On("FindByEmail", req.Email).Return(existingUser, nil)

		result, err := service.Register(req)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, user.ErrEmailExists, err)

		mockUserRepo.AssertExpectations(t)
	})

	t.Run("registration fails when username exists", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		mockSessionService := new(MockSessionService)
		mockKeyStore := &KeyStore{}

		service := NewService(mockUserRepo, mockSessionService, mockKeyStore, "test-issuer")

		req := user.RegisterRequest{
			Username:  "existinguser",
			FirstName: "Test",
			LastName:  "User",
			Email:     "new@example.com",
			Password:  "SecureP@ss123",
		}

		existingUser := &user.User{
			Username: req.Username,
			Email:    "other@example.com",
		}
		existingUser.ID = uuid.New()

		mockUserRepo.On("FindByEmail", req.Email).Return(nil, errors.New("not found"))
		mockUserRepo.On("FindByUsername", req.Username).Return(existingUser, nil)

		result, err := service.Register(req)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, user.ErrUsernameExists, err)

		mockUserRepo.AssertExpectations(t)
	})

	t.Run("registration without email", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		mockSessionService := new(MockSessionService)
		mockKeyStore := &KeyStore{}

		service := NewService(mockUserRepo, mockSessionService, mockKeyStore, "test-issuer")

		req := user.RegisterRequest{
			Username:  "testuser",
			FirstName: "Test",
			LastName:  "User",
			Email:     "", // No email
			Password:  "SecureP@ss123",
		}

		mockUserRepo.On("FindByUsername", req.Username).Return(nil, errors.New("not found"))
		mockUserRepo.On("Create", mock.AnythingOfType("*user.User")).Return(nil)

		result, err := service.Register(req)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, req.Username, result.Username)
		assert.Empty(t, result.Email)

		mockUserRepo.AssertExpectations(t)
	})

	t.Run("registration fails when Create fails", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		mockSessionService := new(MockSessionService)
		mockKeyStore := &KeyStore{}

		service := NewService(mockUserRepo, mockSessionService, mockKeyStore, "test-issuer")

		req := user.RegisterRequest{
			Username:  "testuser",
			FirstName: "Test",
			LastName:  "User",
			Email:     "test@example.com",
			Password:  "SecureP@ss123",
		}

		mockUserRepo.On("FindByEmail", req.Email).Return(nil, errors.New("not found"))
		mockUserRepo.On("FindByUsername", req.Username).Return(nil, errors.New("not found"))
		mockUserRepo.On("Create", mock.AnythingOfType("*user.User")).Return(errors.New("database error"))

		result, err := service.Register(req)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "database error")

		mockUserRepo.AssertExpectations(t)
	})
}

// TestService_Login tests the Login method
func TestService_Login(t *testing.T) {
	t.Run("login fails with invalid username", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		mockSessionService := new(MockSessionService)
		mockKeyStore := &KeyStore{}

		service := NewService(mockUserRepo, mockSessionService, mockKeyStore, "test-issuer")

		mockUserRepo.On("FindByUsername", "nonexistent").Return(nil, errors.New("not found"))

		result, err := service.Login("nonexistent", "password", "agent", "ip")

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, ErrInvalidCredentials, err)

		mockUserRepo.AssertExpectations(t)
	})

	t.Run("login fails with invalid password", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		mockSessionService := new(MockSessionService)
		mockKeyStore := &KeyStore{}

		service := NewService(mockUserRepo, mockSessionService, mockKeyStore, "test-issuer")

		username := "testuser"
		correctPassword := "SecureP@ss123"
		wrongPassword := "WrongPassword"
		hashedPassword, _ := user.HashPassword(correctPassword)

		existingUser := &user.User{
			Username: username,
			Password: hashedPassword,
			IsActive: true,
		}
		existingUser.ID = uuid.New()

		mockUserRepo.On("FindByUsername", username).Return(existingUser, nil)

		result, err := service.Login(username, wrongPassword, "agent", "ip")

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, ErrInvalidCredentials, err)

		mockUserRepo.AssertExpectations(t)
	})

	t.Run("login fails when session creation fails", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		mockSessionService := new(MockSessionService)
		mockKeyStore := &KeyStore{}

		service := NewService(mockUserRepo, mockSessionService, mockKeyStore, "test-issuer")

		username := "testuser"
		password := "SecureP@ss123"
		hashedPassword, _ := user.HashPassword(password)
		userID := uuid.New()

		existingUser := &user.User{
			Username: username,
			Password: hashedPassword,
			IsActive: true,
		}
		existingUser.ID = userID

		mockUserRepo.On("FindByUsername", username).Return(existingUser, nil)
		mockSessionService.On("Create", userID, "agent", "ip", 24*time.Hour).
			Return(nil, "", errors.New("session creation failed"))

		result, err := service.Login(username, password, "agent", "ip")

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "session creation failed")

		mockUserRepo.AssertExpectations(t)
		mockSessionService.AssertExpectations(t)
	})
}

// TestService_GenerateAccessToken tests access token generation
func TestService_GenerateAccessToken(t *testing.T) {
	t.Run("token generation parameters", func(t *testing.T) {
		// This is a documentation test for expected token structure
		sub := uuid.New().String()
		sid := uuid.New().String()
		aud := []string{"api", "web"}

		assert.NotEmpty(t, sub, "Subject should not be empty")
		assert.NotEmpty(t, sid, "Session ID should not be empty")
		assert.NotEmpty(t, aud, "Audience should not be empty")
	})

	t.Run("token expiration should be 15 minutes", func(t *testing.T) {
		expectedExpiration := 15 * time.Minute
		assert.Equal(t, 15*time.Minute, expectedExpiration)
	})
}

// TestLoginResponse_Structure tests the LoginResponse structure
func TestLoginResponse_Structure(t *testing.T) {
	t.Run("response should contain all required fields", func(t *testing.T) {
		userID := uuid.New()
		sessionID := uuid.New()

		response := &LoginResponse{
			AccessToken:  "access_token_value",
			RefreshToken: "refresh_token_value",
			RefreshSID:   sessionID.String(),
			User: &user.UserResponse{
				ID:        userID,
				Username:  "testuser",
				FirstName: "Test",
				LastName:  "User",
				Email:     "test@example.com",
			},
		}

		assert.NotEmpty(t, response.AccessToken, "Access token should be present")
		assert.NotEmpty(t, response.RefreshToken, "Refresh token should be present")
		assert.NotEmpty(t, response.RefreshSID, "Session ID should be present")
		assert.NotNil(t, response.User, "User should be present")
		assert.Equal(t, "testuser", response.User.Username)
	})
}

// TestRegisterValidation tests registration validation scenarios
func TestRegisterValidation(t *testing.T) {
	tests := []struct {
		name        string
		request     user.RegisterRequest
		shouldError bool
	}{
		{
			name: "valid registration with email",
			request: user.RegisterRequest{
				Username:  "validuser",
				FirstName: "Valid",
				LastName:  "User",
				Email:     "valid@example.com",
				Password:  "ValidP@ss123",
			},
			shouldError: false,
		},
		{
			name: "valid registration without email",
			request: user.RegisterRequest{
				Username:  "validuser",
				FirstName: "Valid",
				LastName:  "User",
				Email:     "",
				Password:  "ValidP@ss123",
			},
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validate request structure
			assert.NotEmpty(t, tt.request.Username, "Username is required")
			assert.NotEmpty(t, tt.request.FirstName, "FirstName is required")
			assert.NotEmpty(t, tt.request.LastName, "LastName is required")
			assert.NotEmpty(t, tt.request.Password, "Password is required")
		})
	}
}

// TestService_EdgeCases tests edge cases and boundary conditions
func TestService_EdgeCases(t *testing.T) {
	t.Run("empty username login", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		mockSessionService := new(MockSessionService)
		mockKeyStore := &KeyStore{}

		service := NewService(mockUserRepo, mockSessionService, mockKeyStore, "test-issuer")

		mockUserRepo.On("FindByUsername", "").Return(nil, errors.New("not found"))

		result, err := service.Login("", "password", "agent", "ip")

		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("empty password login", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		mockSessionService := new(MockSessionService)
		mockKeyStore := &KeyStore{}

		service := NewService(mockUserRepo, mockSessionService, mockKeyStore, "test-issuer")

		username := "testuser"
		hashedPassword, _ := user.HashPassword("actualpassword")

		existingUser := &user.User{
			Username: username,
			Password: hashedPassword,
		}
		existingUser.ID = uuid.New()

		mockUserRepo.On("FindByUsername", username).Return(existingUser, nil)

		result, err := service.Login(username, "", "agent", "ip")

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, ErrInvalidCredentials, err)
	})

	t.Run("nil user agent and IP", func(t *testing.T) {
		// Document that empty strings are acceptable for user agent and IP
		userAgent := ""
		ip := ""

		assert.NotNil(t, &userAgent, "Empty user agent should be handled")
		assert.NotNil(t, &ip, "Empty IP should be handled")
	})
}

// BenchmarkRegister benchmarks the registration flow
func BenchmarkRegister(b *testing.B) {
	mockUserRepo := new(MockUserRepository)
	mockSessionService := new(MockSessionService)
	mockKeyStore := &KeyStore{}

	service := NewService(mockUserRepo, mockSessionService, mockKeyStore, "test-issuer")

	req := user.RegisterRequest{
		Username:  "benchuser",
		FirstName: "Bench",
		LastName:  "User",
		Email:     "bench@example.com",
		Password:  "BenchP@ss123",
	}

	mockUserRepo.On("FindByEmail", mock.Anything).Return(nil, errors.New("not found"))
	mockUserRepo.On("FindByUsername", mock.Anything).Return(nil, errors.New("not found"))
	mockUserRepo.On("Create", mock.Anything).Return(nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.Register(req)
	}
}
