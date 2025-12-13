package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/Anvoria/authly/internal/domain/service"
)

const (
	// ServiceCachePrefix is the prefix for service cache keys
	ServiceCachePrefix = "service:domain:"
	// ServiceCacheCodePrefix is the prefix for service cache keys by code
	ServiceCacheCodePrefix = "service:code:"
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

// cachedServiceInfo represents cached service information
type cachedServiceInfo struct {
	Code   string `json:"code"`
	Active bool   `json:"active"`
}

// GetByDomain retrieves a service by domain, using cache if available
func (c *ServiceCache) GetByDomain(ctx context.Context, domain string) (*service.Service, error) {
	cacheKey := ServiceCachePrefix + domain

	if RedisClient == nil {
		return nil, fmt.Errorf("redis client not initialized")
	}

	// Try to get from Redis cache
	cached, err := RedisClient.Get(ctx, cacheKey).Result()
	if err == nil {
		var info cachedServiceInfo
		if err := json.Unmarshal([]byte(cached), &info); err == nil {
			slog.Debug("Service cache hit from Redis", "domain", domain, "key", cacheKey)
			return &service.Service{
				Code:   info.Code,
				Active: info.Active,
			}, nil
		}
	}

	slog.Debug("Service cache miss, fetching from database", "domain", domain)

	svc, err := c.repo.FindByDomain(domain)
	if err != nil {
		return nil, err
	}

	info := cachedServiceInfo{
		Code:   svc.Code,
		Active: svc.Active,
	}
	data, err := json.Marshal(info)
	if err == nil {
		if err := RedisClient.Set(ctx, cacheKey, data, ServiceCacheTTL).Err(); err != nil {
			slog.Warn("Failed to store service in Redis cache", "domain", domain, "error", err)
		} else {
			slog.Debug("Service cached in Redis", "domain", domain, "key", cacheKey, "ttl", ServiceCacheTTL)
		}
	}

	return svc, nil
}

// GetByCode retrieves a service by code, using cache if available
func (c *ServiceCache) GetByCode(ctx context.Context, code string) (*service.Service, error) {
	cacheKey := ServiceCacheCodePrefix + code

	if RedisClient == nil {
		return nil, fmt.Errorf("redis client not initialized")
	}

	// Try to get from Redis cache
	cached, err := RedisClient.Get(ctx, cacheKey).Result()
	if err == nil {
		var info cachedServiceInfo
		if err := json.Unmarshal([]byte(cached), &info); err == nil {
			slog.Debug("Service cache hit from Redis", "code", code, "key", cacheKey)
			return &service.Service{
				Code:   info.Code,
				Active: info.Active,
			}, nil
		}
	}

	slog.Debug("Service cache miss, fetching from database", "code", code)

	svc, err := c.repo.FindByCode(code)
	if err != nil {
		return nil, err
	}

	info := cachedServiceInfo{
		Code:   svc.Code,
		Active: svc.Active,
	}
	data, err := json.Marshal(info)
	if err == nil {
		if err := RedisClient.Set(ctx, cacheKey, data, ServiceCacheTTL).Err(); err != nil {
			slog.Warn("Failed to store service in Redis cache", "code", code, "error", err)
		} else {
			slog.Debug("Service cached in Redis", "code", code, "key", cacheKey, "ttl", ServiceCacheTTL)
		}
	}

	return svc, nil
}

// InvalidateByDomain removes a service from Redis cache by domain
func (c *ServiceCache) InvalidateByDomain(ctx context.Context, domain string) error {
	cacheKey := ServiceCachePrefix + domain

	if RedisClient == nil {
		return fmt.Errorf("redis client not initialized")
	}

	err := RedisClient.Del(ctx, cacheKey).Err()
	if err == nil {
		slog.Debug("Service cache invalidated in Redis", "domain", domain, "key", cacheKey)
	}
	return err
}
