package role

import (
	"errors"

	"gorm.io/gorm"
)

var (
	ErrRoleNotFound = errors.New("role not found")
)

// Repository defines the interface for role persistence
type Repository interface {
	Create(role *Role) error
	FindByID(id string) (*Role, error)
	FindByName(serviceID, name string) (*Role, error)
	FindByServiceID(serviceID string) ([]*Role, error)
	FindDefaultByServiceID(serviceID string) (*Role, error)
	FindAllDefaults() ([]*Role, error)
	Update(role *Role) error
	Delete(id string) error
}

type repository struct {
	db *gorm.DB
}

// NewRepository returns a Repository backed by the provided GORM database instance.
func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(role *Role) error {
	return r.db.Create(role).Error
}

func (r *repository) FindByID(id string) (*Role, error) {
	var role Role
	if err := r.db.Where("id = ?", id).First(&role).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRoleNotFound
		}
		return nil, err
	}
	return &role, nil
}

func (r *repository) FindByName(serviceID, name string) (*Role, error) {
	var role Role
	if err := r.db.Where("service_id = ? AND name = ?", serviceID, name).First(&role).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRoleNotFound
		}
		return nil, err
	}
	return &role, nil
}

func (r *repository) FindByServiceID(serviceID string) ([]*Role, error) {
	var roles []*Role
	if err := r.db.Where("service_id = ?", serviceID).Order("priority DESC").Find(&roles).Error; err != nil {
		return nil, err
	}
	return roles, nil
}

func (r *repository) FindDefaultByServiceID(serviceID string) (*Role, error) {
	var role Role
	if err := r.db.Where("service_id = ? AND is_default = ?", serviceID, true).First(&role).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRoleNotFound
		}
		return nil, err
	}
	return &role, nil
}

func (r *repository) FindAllDefaults() ([]*Role, error) {
	var roles []*Role
	if err := r.db.Where("is_default = ?", true).Find(&roles).Error; err != nil {
		return nil, err
	}
	return roles, nil
}

func (r *repository) Update(role *Role) error {
	return r.db.Save(role).Error
}

func (r *repository) Delete(id string) error {
	return r.db.Delete(&Role{}, id).Error
}