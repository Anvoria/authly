package session

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"crypto/rand"
	"crypto/sha3"
	"encoding/base64"
	"strings"

	"github.com/Anvoria/authly/internal/cache"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	// ErrInvalidSession is returned when the session is invalid
	ErrInvalidSession = errors.New("invalid session")
	// ErrInvalidSecret is returned when the session secret is invalid
	ErrInvalidSecret = errors.New("invalid session secret")
	// ErrExpiredSession is returned when the session has expired
	ErrExpiredSession = errors.New("session expired")
	// ErrReplayDetected is returned when a replay attack is detected
	ErrReplayDetected = errors.New("replay detected")
)

// Service interface for session operations
type Service interface {
	Create(userID uuid.UUID, userAgent, ip string, scopes []string, ttl time.Duration) (sessionID uuid.UUID, secret string, err error)
	Validate(sessionID uuid.UUID, secret string) (*Session, error)
	Rotate(sessionID uuid.UUID, oldSecret string, ttl time.Duration) (newSecret string, err error)
	Revoke(sessionID uuid.UUID) error
	RevokeAllUserSessions(userID uuid.UUID) error
	Exists(sessionID uuid.UUID) (bool, error)
	UpdateScopes(sessionID uuid.UUID, scopes []string) error
}

// service struct for session operations
type service struct {
	repo            Repository
	revocationCache *cache.TokenRevocationCache
}

// NewService creates a session Service that uses the provided Repository and does not configure a revocation cache.
func NewService(repo Repository) Service {
	return &service{repo: repo}
}

// NewServiceWithCache creates a Service configured with the provided repository and an optional token revocation cache.
// If revocationCache is nil the service will operate without a revocation cache.
func NewServiceWithCache(repo Repository, revocationCache *cache.TokenRevocationCache) Service {
	return &service{repo: repo, revocationCache: revocationCache}
}

// generateSecret generates a random secret for the session
func generateSecret() (string, error) {
	b := make([]byte, 48)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.RawStdEncoding.EncodeToString(b), nil
}

// hashSecret hashes the secret using SHA-3-256
func hashSecret(secret string) string {
	h := sha3.Sum256([]byte(secret))
	return base64.RawStdEncoding.EncodeToString(h[:])
}

// Create creates a new session
func (s *service) Create(userID uuid.UUID, userAgent, ip string, scopes []string, ttl time.Duration) (uuid.UUID, string, error) {
	secret, err := generateSecret()
	if err != nil {
		return uuid.Nil, "", err
	}

	sess := &Session{
		UserID:        userID.String(),
		RefreshHash:   hashSecret(secret),
		ExpiresAt:     time.Now().UTC().Add(ttl),
		UserAgent:     userAgent,
		IPAddress:     ip,
		GrantedScopes: strings.Join(scopes, " "),
		LastUsedAt:    time.Now().UTC(),
	}

	sess.ID = uuid.New()

	if err := s.repo.Create(sess); err != nil {
		return uuid.Nil, "", err
	}

	return sess.ID, secret, nil
}

// Validate validates a session
func (s *service) Validate(id uuid.UUID, secret string) (*Session, error) {
	sess, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidSession
		}
		return nil, err
	}

	if sess.Revoked {
		return nil, ErrInvalidSession
	}

	if time.Now().UTC().After(sess.ExpiresAt) {
		return nil, ErrExpiredSession
	}

	if hashSecret(secret) != sess.RefreshHash {
		return nil, ErrInvalidSecret
	}

	if err := s.repo.UpdateLastUsed(id, time.Now().UTC()); err != nil {
		return nil, err
	}

	return sess, nil
}

// Rotate rotates a session by generating a new secret and updating the session hash
func (s *service) Rotate(id uuid.UUID, oldSecret string, ttl time.Duration) (string, error) {
	_, err := s.Validate(id, oldSecret)
	if err != nil {
		return "", err
	}

	newSecret, err := generateSecret()
	if err != nil {
		return "", err
	}

	success, err := s.repo.UpdateHash(id, hashSecret(oldSecret), hashSecret(newSecret), time.Now().UTC().Add(ttl))
	if err != nil {
		return "", err
	}

	if !success {
		return "", ErrReplayDetected
	}

	return newSecret, nil
}

// Revoke revokes a session
func (s *service) Revoke(id uuid.UUID) error {
	// Get session info before revoking to get ExpiresAt for Redis TTL
	sess, err := s.repo.FindByIDForRevoke(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrInvalidSession
		}
		return err
	}

	// Revoke in database
	if err := s.repo.Revoke(id); err != nil {
		return err
	}

	// Store revocation in Redis cache if available
	if s.revocationCache != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		// Calculate TTL: time until session expires (or minimum 1 hour if already expired)
		ttl := time.Until(sess.ExpiresAt)
		if ttl <= 0 {
			ttl = 1 * time.Hour
		}
		if err := s.revocationCache.RevokeSession(ctx, id.String(), ttl); err != nil {
			slog.Warn("Failed to store session revocation in Redis", "error", err, "session_id", id.String())
		}
	}

	return nil
}

// RevokeAllUserSessions revokes all sessions for a specific user
func (s *service) RevokeAllUserSessions(userID uuid.UUID) error {
	sessions, err := s.repo.FindSessionsByUserID(userID)
	if err != nil {
		return fmt.Errorf("failed to get sessions for user %s: %w", userID, err)
	}

	for _, sess := range sessions {
		if err := s.Revoke(sess.ID); err != nil {
			slog.Warn("Failed to revoke session", "error", err, "session_id", sess.ID.String(), "user_id", userID.String())
		}
	}

	return nil
}

// Exists checks if a session exists and is valid (not revoked, not expired)
func (s *service) Exists(id uuid.UUID) (bool, error) {
	sess, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}

	if sess.Revoked {
		return false, nil
	}

	if time.Now().UTC().After(sess.ExpiresAt) {
		return false, nil
	}

	return true, nil
}

// UpdateScopes updates the granted scopes for a session
func (s *service) UpdateScopes(sessionID uuid.UUID, scopes []string) error {
	return s.repo.UpdateScopes(sessionID, strings.Join(scopes, " "))
}
