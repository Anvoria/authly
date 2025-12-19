package session

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository interface {
	Create(sess *Session) error
	FindByID(id uuid.UUID) (*Session, error)
	FindByIDForRevoke(id uuid.UUID) (*Session, error)
	UpdateHash(id uuid.UUID, oldHash, newHash string, newExpiry time.Time) (bool, error)
	Revoke(id uuid.UUID) error
	UpdateLastUsed(id uuid.UUID, t time.Time) error
	FindSessionsByUserID(userID uuid.UUID) ([]Session, error)
	UpdateScopes(id uuid.UUID, scopes string) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db}
}

func (r *repository) Create(sess *Session) error {
	return r.db.Create(sess).Error
}

func (r *repository) FindByID(id uuid.UUID) (*Session, error) {
	var sess Session
	err := r.db.Where("id = ? AND revoked = false", id).First(&sess).Error
	if err != nil {
		return nil, err
	}
	return &sess, nil
}

func (r *repository) FindByIDForRevoke(id uuid.UUID) (*Session, error) {
	var sess Session
	err := r.db.Where("id = ?", id).First(&sess).Error
	if err != nil {
		return nil, err
	}
	return &sess, nil
}

func (r *repository) UpdateHash(id uuid.UUID, oldHash, newHash string, newExpiry time.Time) (bool, error) {
	res := r.db.Model(&Session{}).
		Where("id = ? AND refresh_hash = ? AND revoked = false", id, oldHash).
		Updates(map[string]any{
			"refresh_hash": newHash,
			"expires_at":   newExpiry,
			"last_used_at": time.Now().UTC(),
		})

	if res.Error != nil {
		return false, res.Error
	}

	return res.RowsAffected == 1, nil
}

func (r *repository) Revoke(id uuid.UUID) error {
	return r.db.Model(&Session{}).
		Where("id = ? AND revoked = false", id).
		Update("revoked", true).Error
}

func (r *repository) UpdateLastUsed(id uuid.UUID, t time.Time) error {
	return r.db.Model(&Session{}).
		Where("id = ? AND revoked = false", id).
		Update("last_used_at", t).Error
}

func (r *repository) FindSessionsByUserID(userID uuid.UUID) ([]Session, error) {
	var sessions []Session
	err := r.db.Where("user_id = ? AND revoked = false", userID.String()).Find(&sessions).Error
	if err != nil {
		return nil, err
	}
	return sessions, nil
}

func (r *repository) UpdateScopes(id uuid.UUID, scopes string) error {
	return r.db.Model(&Session{}).
		Where("id = ?", id).
		Update("granted_scopes", scopes).Error
}
