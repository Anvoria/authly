package user

import "gorm.io/gorm"

// Repository interface for user operations
type Repository interface {
	Create(user *User) error
	GetByID(id string) (*User, error)
	GetByEmail(email string) (*User, error)
	GetByUsername(username string) (*User, error)
	Update(user *User) error
	Delete(id string) error
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

// GetByID gets a user by ID
func (r *repository) GetByID(id string) (*User, error) {
	var user User
	if err := r.db.Where("id = ?", id).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// GetByEmail gets a user by email
func (r *repository) GetByEmail(email string) (*User, error) {
	var user User
	if err := r.db.Where("email = ?", email).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// GetByUsername gets a user by username
func (r *repository) GetByUsername(username string) (*User, error) {
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
