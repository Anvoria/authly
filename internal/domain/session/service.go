package session

import (
	"errors"
	"time"

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

type Service interface {
	Create(userID uuid.UUID, userAgent, ip string, ttl time.Duration) (sessionID uuid.UUID, secret string, err error)
	Validate(sessionID uuid.UUID, secret string) (*Session, error)
	Rotate(sessionID uuid.UUID, oldSecret string, ttl time.Duration) (newSecret string, err error)
	Revoke(sessionID uuid.UUID) error
}
