package oidc

import (
	"time"

	"gorm.io/gorm"
)

// Repository interface for authorization code operations
type Repository interface {
	Create(code *AuthorizationCode) error
	FindByCode(code string) (*AuthorizationCode, error)
	MarkAsUsed(code string) error
	DeleteExpired() error
}

// repository struct for authorization code operations
type repository struct {
	db *gorm.DB
}

// NewRepository returns a Repository backed by the provided GORM DB handle and configured to persist authorization codes.
func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

// Create creates a new authorization code
func (r *repository) Create(code *AuthorizationCode) error {
	return r.db.Create(code).Error
}

// FindByCode finds an authorization code by code string
func (r *repository) FindByCode(code string) (*AuthorizationCode, error) {
	var authCode AuthorizationCode
	err := r.db.Where("code = ? AND used = false AND expires_at > ?", code, time.Now()).First(&authCode).Error
	if err != nil {
		return nil, err
	}
	return &authCode, nil
}

// MarkAsUsed marks an authorization code as used
func (r *repository) MarkAsUsed(code string) error {
	result := r.db.Model(&AuthorizationCode{}).
		Where("code = ? AND used = false AND expires_at > ?", code, time.Now()).
		Update("used", true)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

// DeleteExpired deletes expired authorization codes
func (r *repository) DeleteExpired() error {
	return r.db.Where("expires_at < ?", time.Now()).Delete(&AuthorizationCode{}).Error
}
