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
	// ErrUserNotFound is returned when user is not found
	ErrUserNotFound = errors.New("user not found")
)

// Service interface for user operations
type Service interface {
	Register(req RegisterRequest) (*User, error)
	GetUserInfo(userID string) (*User, error)
	FindByUsername(username string) (*User, error)
	VerifyPassword(u *User, password string) bool

	// Management methods
	ListUsers(limit, offset int) ([]*User, int64, error)
	UpdateUser(id string, email *string, username *string, isActive *bool) (*User, error)
	DeleteUser(id string) error
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

func (s *service) FindByUsername(username string) (*User, error) {
	return s.repo.FindByUsername(username)
}

func (s *service) VerifyPassword(u *User, password string) bool {
	return s.repo.VerifyPassword(u, password)
}

func (s *service) ListUsers(limit, offset int) ([]*User, int64, error) {
	return s.repo.FindAll(limit, offset)
}

func (s *service) UpdateUser(id string, email *string, username *string, isActive *bool) (*User, error) {
	user, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	updates := false
	if email != nil && *email != user.Email {
		if *email != "" {
			existing, err := s.repo.FindByEmail(*email)
			if err == nil && existing.ID != user.ID {
				return nil, ErrEmailExists
			}
		}
		user.Email = *email
		updates = true
	}

	if username != nil && *username != user.Username {
		if *username != "" {
			existing, err := s.repo.FindByUsername(*username)
			if err == nil && existing.ID != user.ID {
				return nil, ErrUsernameExists
			}
		}
		user.Username = *username
		updates = true
	}

	if isActive != nil && *isActive != user.IsActive {
		user.IsActive = *isActive
		updates = true
	}

	if updates {
		if err := s.repo.Update(user); err != nil {
			return nil, err
		}
	}

	return user, nil
}

func (s *service) DeleteUser(id string) error {
	user, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}
	if user == nil {
		return ErrUserNotFound
	}
	return s.repo.Delete(id)
}
