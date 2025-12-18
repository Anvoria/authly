package cache

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/Anvoria/authly/internal/domain/service"
	"github.com/google/uuid"
)

const (
	// ServiceCachePrefix is the prefix for service cache keys
	ServiceCachePrefix = "service:domain:"
	// ServiceCacheClientIDPrefix is the prefix for service cache keys by client_id
	ServiceCacheClientIDPrefix = "service:client_id:"
	// ServiceCacheTTL is the time-to-live for cached service data
	ServiceCacheTTL = 1 * time.Hour
)

// CacheInvalidator defines interface for cache invalidation
type CacheInvalidator interface {
	InvalidateByDomain(ctx context.Context, domain string) error
}

// ServiceCache provides caching for service lookups by domain
type ServiceCache struct {
	repo service.Repository
}

// NewServiceCache creates a ServiceCache that uses the provided repository for backend lookups.
func NewServiceCache(repo service.Repository) *ServiceCache {
	return &ServiceCache{repo: repo}
}

// GetByDomain retrieves a service by domain, using cache if available
func (c *ServiceCache) GetByDomain(ctx context.Context, domain string) (*service.Service, error) {
	cacheKey := ServiceCachePrefix + domain

	if RedisClient != nil {
		// Try to get from Redis cache
		cached, err := RedisClient.Get(ctx, cacheKey).Result()
		if err == nil {
			var svc service.Service
			if err := json.Unmarshal([]byte(cached), &svc); err == nil {
				if svc.ID != uuid.Nil {
					slog.Debug("Service cache hit from Redis", "domain", domain, "key", cacheKey)
					return &svc, nil
				}
			}
		}
	} else {
		slog.Warn("Redis client not initialized, skipping cache", "domain", domain)
	}

	slog.Debug("Service cache miss, fetching from database", "domain", domain)

	svc, err := c.repo.FindByDomain(domain)
	if err != nil {
		return nil, err
	}

	if RedisClient != nil {
		data, err := json.Marshal(svc)
		if err == nil {
			if err := RedisClient.Set(ctx, cacheKey, data, ServiceCacheTTL).Err(); err != nil {
				slog.Warn("Failed to store service in Redis cache", "domain", domain, "error", err)
			} else {
				slog.Debug("Service cached in Redis", "domain", domain, "key", cacheKey, "ttl", ServiceCacheTTL)
			}
		}
	}

	return svc, nil
}

// GetByClientID retrieves a service by client_id, using cache if available
func (c *ServiceCache) GetByClientID(ctx context.Context, clientID string) (*service.Service, error) {
	cacheKey := ServiceCacheClientIDPrefix + clientID

	if RedisClient != nil {
		cached, err := RedisClient.Get(ctx, cacheKey).Result()
		if err == nil {
			var svc service.Service
			if err := json.Unmarshal([]byte(cached), &svc); err == nil {
				if svc.ID != uuid.Nil {
					slog.Debug("Service cache hit from Redis", "client_id", clientID, "key", cacheKey)
					return &svc, nil
				}
			}
		}
	} else {
		slog.Warn("Redis client not initialized, skipping cache", "client_id", clientID)
	}

	slog.Debug("Service cache miss, fetching from database", "client_id", clientID)

	svc, err := c.repo.FindByClientID(clientID)
	if err != nil {
		return nil, err
	}

	if RedisClient != nil {
		data, err := json.Marshal(svc)
		if err == nil {
			if err := RedisClient.Set(ctx, cacheKey, data, ServiceCacheTTL).Err(); err != nil {
				slog.Warn("Failed to store service in Redis cache", "client_id", clientID, "error", err)
			} else {
				slog.Debug("Service cached in Redis", "client_id", clientID, "key", cacheKey, "ttl", ServiceCacheTTL)
			}
		}
	}

	return svc, nil
}

// InvalidateByDomain removes a service from Redis cache by domain
func (c *ServiceCache) InvalidateByDomain(ctx context.Context, domain string) error {
	cacheKey := ServiceCachePrefix + domain

	if RedisClient == nil {
		slog.Warn("Redis client not initialized, skipping cache invalidation", "domain", domain)
		return nil
	}

	// Try to get service to find client_id for complete invalidation
	// We ignore error here because even if we fail to find the service/client_id,
	// we still want to at least invalidate the domain key
	svc, _ := c.repo.FindByDomain(domain)

	keys := []string{cacheKey}
	if svc != nil && svc.ClientID != "" {
		keys = append(keys, ServiceCacheClientIDPrefix+svc.ClientID)
	}

	err := RedisClient.Del(ctx, keys...).Err()
	if err == nil {
		slog.Debug("Service cache invalidated in Redis", "domain", domain, "keys", keys)
	}
	return err
}
