package service

import (
	"errors"

	"gorm.io/gorm"
)

// ServiceInterface defines the interface for service operations
type ServiceInterface interface {
	Create(code, name, description string) (*Service, error)
	FindByID(id string) (*Service, error)
	FindByCode(code string) (*Service, error)
	FindAll() ([]*Service, error)
	FindActive() ([]*Service, error)
	Update(id string, name, description *string, active *bool) (*Service, error)
	Delete(id string) error
}

// service implements ServiceInterface
type serviceImpl struct {
	repo Repository
}

// NewService creates a ServiceInterface that uses the provided Repository for persistence.
func NewService(repo Repository) ServiceInterface {
	return &serviceImpl{repo}
}

// Create creates a new service
func (s *serviceImpl) Create(code, name, description string) (*Service, error) {
	// Check if code already exists
	_, err := s.repo.FindByCode(code)
	if err == nil {
		return nil, ErrServiceCodeExists
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	svc := &Service{
		Code:        code,
		Name:        name,
		Description: description,
		Active:      true,
		IsSystem:    false,
	}

	if err := s.repo.Create(svc); err != nil {
		return nil, err
	}

	return svc, nil
}

// FindByID gets a service by ID
func (s *serviceImpl) FindByID(id string) (*Service, error) {
	svc, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrServiceNotFound
		}
		return nil, err
	}
	return svc, nil
}

// FindByCode gets a service by code
func (s *serviceImpl) FindByCode(code string) (*Service, error) {
	svc, err := s.repo.FindByCode(code)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrServiceNotFound
		}
		return nil, err
	}
	return svc, nil
}

// FindAll gets all services
func (s *serviceImpl) FindAll() ([]*Service, error) {
	return s.repo.FindAll()
}

// FindActive gets all active services
func (s *serviceImpl) FindActive() ([]*Service, error) {
	return s.repo.FindActive()
}

// Update updates a service
// Only non-nil fields will be updated
func (s *serviceImpl) Update(id string, name, description *string, active *bool) (*Service, error) {
	svc, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrServiceNotFound
		}
		return nil, err
	}

	if name != nil {
		svc.Name = *name
	}
	if description != nil {
		svc.Description = *description
	}
	if active != nil {
		svc.Active = *active
	}

	if err := s.repo.Update(svc); err != nil {
		return nil, err
	}

	return svc, nil
}

// Delete deletes a service
func (s *serviceImpl) Delete(id string) error {
	return s.repo.Delete(id)
}