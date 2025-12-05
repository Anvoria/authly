package session

import (
	"errors"
	"fmt"
	"time"

	"crypto/rand"
	"crypto/sha3"
	"encoding/base64"

	"github.com/google/uuid"
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
	Create(userID uuid.UUID, userAgent, ip string, ttl time.Duration) (sessionID uuid.UUID, secret string, err error)
	Validate(sessionID uuid.UUID, secret string) (*Session, error)
	Rotate(sessionID uuid.UUID, oldSecret string, ttl time.Duration) (newSecret string, err error)
	Revoke(sessionID uuid.UUID) error
}

// service struct for session operations
type service struct {
	repo Repository
}

// NewService creates a new session service
func NewService(repo Repository) Service {
	return &service{repo}
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
func (s *service) Create(userID uuid.UUID, userAgent, ip string, ttl time.Duration) (uuid.UUID, string, error) {
	secret, err := generateSecret()
	if err != nil {
		return uuid.Nil, "", err
	}

	sess := &Session{
		UserID:      userID.String(),
		RefreshHash: hashSecret(secret),
		ExpiresAt:   time.Now().UTC().Add(ttl),
		UserAgent:   userAgent,
		IPAddress:   ip,
		LastUsedAt:  time.Now().UTC(),
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
		return nil, ErrInvalidSession
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
		return nil, fmt.Errorf("failed to update session usage: %w", err)
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
	return s.repo.Revoke(id)
}
