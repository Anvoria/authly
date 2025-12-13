package user

import "gorm.io/gorm"

// Repository interface for user operations
type Repository interface {
	Create(user *User) error
	FindByID(id string) (*User, error)
	FindByEmail(email string) (*User, error)
	FindByUsername(username string) (*User, error)
	Update(user *User) error
	Delete(id string) error
	VerifyPassword(u *User, password string) bool
}

// repository struct for user operations
type repository struct {
	db *gorm.DB
}

// NewRepository creates a new user repository
func NewRepository(db *gorm.DB) Repository {
	return &repository{db}
}

// Create creates a new user
func (r *repository) Create(user *User) error {
	return r.db.Create(user).Error
}

// FindByID gets a user by ID
func (r *repository) FindByID(id string) (*User, error) {
	var user User
	if err := r.db.Where("id = ?", id).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// FindByEmail gets a user by email
func (r *repository) FindByEmail(email string) (*User, error) {
	var user User
	if err := r.db.Where("email = ?", email).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// FindByUsername gets a user by username
func (r *repository) FindByUsername(username string) (*User, error) {
	var user User
	if err := r.db.Where("username = ?", username).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// Update updates a user
func (r *repository) Update(user *User) error {
	if err := r.db.Save(user).Error; err != nil {
		return err
	}
	return nil
}

// Delete deletes a user
func (r *repository) Delete(id string) error {
	if err := r.db.Delete(&User{}, id).Error; err != nil {
		return err
	}
	return nil
}

// VerifyPassword verifies if the provided password matches the user's hashed password
func (r *repository) VerifyPassword(u *User, password string) bool {
	return VerifyPassword(password, u.Password)
}
