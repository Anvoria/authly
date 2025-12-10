package auth

import (
	"context"

	"github.com/Anvoria/authly/internal/cache"
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

// NewServiceRepositoryAdapter creates a ServiceRepository backed by the provided ServiceCache.
// The returned repository adapts services stored in the cache to the package's ServiceRepository interface.
func NewServiceRepositoryAdapter(cache *cache.ServiceCache) ServiceRepository {
	return &serviceRepositoryAdapter{cache: cache}
}

type serviceRepositoryAdapter struct {
	cache *cache.ServiceCache
}

func (a *serviceRepositoryAdapter) FindByDomain(domain string) (ServiceInfo, error) {
	ctx := context.Background()
	service, err := a.cache.GetByDomain(ctx, domain)
	if err != nil {
		return nil, err
	}
	return &serviceInfoAdapter{service: service}, nil
}