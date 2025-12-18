package service

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CacheInvalidator defines interface for cache invalidation
type CacheInvalidator interface {
	InvalidateByDomain(ctx context.Context, domain string) error
}

// ServiceInterface defines the interface for service operations
type ServiceInterface interface {
	Create(slug, name, description, domain string, redirectURIs []string, allowedScopes []string) (*Service, error)
	FindByID(id string) (*Service, error)
	FindByClientID(clientID string) (*Service, error)
	FindByDomain(domain string) (*Service, error)
	FindAll() ([]*Service, error)
	FindActive() ([]*Service, error)
	Update(id string, name, description, domain *string, active *bool) (*Service, error)
	Delete(id string) error
}

// service implements ServiceInterface
type serviceImpl struct {
	repo  Repository
	cache CacheInvalidator
}

// NewService creates a ServiceInterface that uses the provided repository and optional cache invalidator.
// NewService constructs a ServiceInterface that uses the provided repository and optional cache invalidator.
// If cache is nil, cache invalidation is disabled for the returned service.
func NewService(repo Repository, cache CacheInvalidator) ServiceInterface {
	return &serviceImpl{repo: repo, cache: cache}
}

// Create creates a new service
func (s *serviceImpl) Create(slug, name, description, domain string, redirectURIs []string, allowedScopes []string) (*Service, error) {
	clientID := GenerateClientID(slug)

	// Check if client_id already exists
	_, err := s.repo.FindByClientID(clientID)
	if err == nil {
		return nil, ErrServiceClientIDExists
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// Check if domain already exists
	if domain != "" {
		_, err := s.repo.FindByDomain(domain)
		if err == nil {
			return nil, ErrServiceDomainExists
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}

	svc := &Service{
		ClientID:      clientID,
		ClientSecret:  GenerateClientSecret(),
		RedirectURIs:  redirectURIs,
		AllowedScopes: allowedScopes,
		Name:          name,
		Description:   description,
		Domain:        domain,
		Active:        true,
		IsSystem:      false,
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

// FindByClientID gets a service by client_id
func (s *serviceImpl) FindByClientID(clientID string) (*Service, error) {
	svc, err := s.repo.FindByClientID(clientID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrServiceNotFound
		}
		return nil, err
	}
	return svc, nil
}

// FindByDomain gets a service by domain
func (s *serviceImpl) FindByDomain(domain string) (*Service, error) {
	svc, err := s.repo.FindByDomain(domain)
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
func (s *serviceImpl) Update(id string, name, description, domain *string, active *bool) (*Service, error) {
	svc, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrServiceNotFound
		}
		return nil, err
	}

	oldDomain := svc.Domain

	if name != nil {
		svc.Name = *name
	}
	if description != nil {
		svc.Description = *description
	}
	if domain != nil {
		svc.Domain = *domain
	}
	if active != nil {
		svc.Active = *active
	}

	if err := s.repo.Update(svc); err != nil {
		return nil, err
	}

	// Invalidate cache if domain changed or if cache is available
	if s.cache != nil {
		ctx := context.Background()
		// Invalidate old domain cache
		if oldDomain != "" {
			_ = s.cache.InvalidateByDomain(ctx, oldDomain)
		}
		// Invalidate new domain cache
		if svc.Domain != "" {
			_ = s.cache.InvalidateByDomain(ctx, svc.Domain)
		}
	}

	return svc, nil
}

// Delete deletes a service
func (s *serviceImpl) Delete(id string) error {
	return s.repo.Delete(id)
}

// GenerateClientID produces a client identifier in the form "authly_<slug>_<8hex>".
// The final segment is the first eight hexadecimal characters of a newly generated UUID.
func GenerateClientID(slug string) string {
	return fmt.Sprintf("authly_%s_%s", slug, uuid.New().String()[:8])
}

// GenerateClientSecret produces a 32-character secret suitable for use as an OAuth client secret.
// The secret is a standard base64 (A–Z, a–z, 0–9, +, /) representation derived from a UUID, trimmed to 32 characters with no padding.
func GenerateClientSecret() string {
	return base64.RawStdEncoding.EncodeToString([]byte(uuid.New().String()))[:32]
}
