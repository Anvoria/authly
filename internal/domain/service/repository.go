package service

import "gorm.io/gorm"

// Repository interface for service operations
type Repository interface {
	Create(service *Service) error
	FindByID(id string) (*Service, error)
	FindByCode(code string) (*Service, error)
	FindAll() ([]*Service, error)
	FindActive() ([]*Service, error)
	Update(service *Service) error
	Delete(id string) error
}

// repository struct for service operations
type repository struct {
	db *gorm.DB
}

// NewRepository creates a new service repository
func NewRepository(db *gorm.DB) Repository {
	return &repository{db}
}

// Create creates a new service
func (r *repository) Create(service *Service) error {
	return r.db.Create(service).Error
}

// FindByID gets a service by ID
func (r *repository) FindByID(id string) (*Service, error) {
	var service Service
	if err := r.db.Where("id = ?", id).First(&service).Error; err != nil {
		return nil, err
	}
	return &service, nil
}

// FindByCode gets a service by code
func (r *repository) FindByCode(code string) (*Service, error) {
	var service Service
	if err := r.db.Where("code = ?", code).First(&service).Error; err != nil {
		return nil, err
	}
	return &service, nil
}

// FindAll gets all services
func (r *repository) FindAll() ([]*Service, error) {
	var services []*Service
	if err := r.db.Find(&services).Error; err != nil {
		return nil, err
	}
	return services, nil
}

// FindActive gets all active services
func (r *repository) FindActive() ([]*Service, error) {
	var services []*Service
	if err := r.db.Where("active = ?", true).Find(&services).Error; err != nil {
		return nil, err
	}
	return services, nil
}

// Update updates a service
func (r *repository) Update(service *Service) error {
	if err := r.db.Save(service).Error; err != nil {
		return err
	}
	return nil
}

// Delete deletes a service (soft delete)
func (r *repository) Delete(id string) error {
	if err := r.db.Delete(&Service{}, id).Error; err != nil {
		return err
	}
	return nil
}
