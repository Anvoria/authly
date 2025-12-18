package cache

import (
	"context"
	"fmt"
	"time"
)

const (
	// TokenRevocationPrefix is the prefix for revoked token cache keys
	TokenRevocationPrefix = "token:revoked:"
	// SessionRevocationPrefix is the prefix for revoked session cache keys
	SessionRevocationPrefix = "session:revoked:"
)

// TokenRevocationCache provides caching for token revocation checks
type TokenRevocationCache struct{}

// NewTokenRevocationCache creates a new TokenRevocationCache instance
func NewTokenRevocationCache() *TokenRevocationCache {
	return &TokenRevocationCache{}
}

// IsSessionRevoked checks if a session is revoked using Redis cache
// Returns true if session is revoked, false if not revoked or if Redis is unavailable
func (c *TokenRevocationCache) IsSessionRevoked(ctx context.Context, sessionID string) (bool, error) {
	if RedisClient == nil {
		return false, fmt.Errorf("redis client not initialized")
	}

	cacheKey := SessionRevocationPrefix + sessionID
	exists, err := RedisClient.Exists(ctx, cacheKey).Result()
	if err != nil {
		return false, err
	}

	return exists > 0, nil
}

// RevokeSession marks a session as revoked in Redis cache
func (c *TokenRevocationCache) RevokeSession(ctx context.Context, sessionID string, ttl time.Duration) error {
	if RedisClient == nil {
		return fmt.Errorf("redis client not initialized")
	}

	cacheKey := SessionRevocationPrefix + sessionID
	return RedisClient.Set(ctx, cacheKey, "1", ttl).Err()
}
