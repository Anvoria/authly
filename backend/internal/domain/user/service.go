package user

import "errors"

var (
	// ErrEmailExists is returned when trying to register with an email that already exists
	ErrEmailExists = errors.New("email already exists")
	// ErrUsernameExists is returned when trying to register with a username that already exists
	ErrUsernameExists = errors.New("username already exists")
	// ErrUsernameRequired is returned when trying to register with an empty username
	ErrUsernameRequired = errors.New("username is required")
	// ErrPasswordRequired is returned when trying to register with an empty password
	ErrPasswordRequired = errors.New("password is required")
)

// Service interface for user operations
type Service interface {
	Register(req RegisterRequest) (*User, error)
	GetUserInfo(userID string) (*User, error)
}

// service struct for user operations
type service struct {
	repo Repository
}

// NewService creates a new user service
func NewService(repo Repository) Service {
	return &service{repo}
}

// Register registers a new user
func (s *service) Register(req RegisterRequest) (*User, error) {
	hashedPassword, err := HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	if req.Username == "" {
		return nil, ErrUsernameRequired
	}

	// Only check email uniqueness if email is provided (not empty)
	if req.Email != "" {
		if _, err := s.repo.FindByEmail(req.Email); err == nil {
			return nil, ErrEmailExists
		}
	}

	if _, err := s.repo.FindByUsername(req.Username); err == nil {
		return nil, ErrUsernameExists
	}

	user := &User{
		Username:  req.Username,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Email:     req.Email,
		Password:  hashedPassword,
		IsActive:  true,
	}

	if err := s.repo.Create(user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *service) GetUserInfo(userID string) (*User, error) {
	return s.repo.FindByID(userID)
}
