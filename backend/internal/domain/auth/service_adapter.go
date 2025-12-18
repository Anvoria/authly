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

// GetClientID returns the service client_id
func (a *serviceInfoAdapter) GetClientID() string {
	return a.service.ClientID
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

func (a *serviceRepositoryAdapter) FindByDomain(ctx context.Context, domain string) (ServiceInfo, error) {
	service, err := a.cache.GetByDomain(ctx, domain)
	if err != nil {
		return nil, err
	}
	return &serviceInfoAdapter{service: service}, nil
}

func (a *serviceRepositoryAdapter) FindByClientID(ctx context.Context, clientID string) (ServiceInfo, error) {
	service, err := a.cache.GetByClientID(ctx, clientID)
	if err != nil {
		return nil, err
	}
	return &serviceInfoAdapter{service: service}, nil
}
