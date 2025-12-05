package user

import "errors"

// RegisterRequest represents the input for user registration
type RegisterRequest struct {
	Username  string
	Email     string
	Password  string
	FirstName string
	LastName  string
}

// Service interface for user operations
type Service interface {
	Register(req RegisterRequest) (*User, error)
	VerifyPassword(u *User, password string) bool
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

	if _, err := s.repo.GetByEmail(req.Email); err == nil {
		return nil, errors.New("email already exists")
	}

	if _, err := s.repo.GetByUsername(req.Username); err == nil {
		return nil, errors.New("username already exists")
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

// VerifyPassword verifies if the provided password matches the user's hashed password
func (s *service) VerifyPassword(u *User, password string) bool {
	return VerifyPassword(password, u.Password)
}
