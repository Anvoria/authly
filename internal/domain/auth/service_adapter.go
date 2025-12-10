package auth

import (
	svc "github.com/Anvoria/authly/internal/domain/service"
)

// serviceInfoAdapter adapts service.Service to ServiceInfo interface
type serviceInfoAdapter struct {
	service *svc.Service
}

// GetCode returns the service code
func (a *serviceInfoAdapter) GetCode() string {
	return a.service.Code
}

// IsActive returns whether the service is active
func (a *serviceInfoAdapter) IsActive() bool {
	return a.service.Active
}

// NewServiceRepositoryAdapter creates a ServiceRepository adapter from service.Repository
func NewServiceRepositoryAdapter(repo svc.Repository) ServiceRepository {
	return &serviceRepositoryAdapter{repo: repo}
}

type serviceRepositoryAdapter struct {
	repo svc.Repository
}

func (a *serviceRepositoryAdapter) FindByDomain(domain string) (ServiceInfo, error) {
	service, err := a.repo.FindByDomain(domain)
	if err != nil {
		return nil, err
	}
	return &serviceInfoAdapter{service: service}, nil
}
