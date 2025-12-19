package session

import (
	"time"

	"github.com/Anvoria/authly/internal/database"
)

type Session struct {
	database.BaseModel

	UserID         string    `gorm:"column:user_id;type:uuid;not null;index"`
	RefreshHash    string    `gorm:"column:refresh_hash;not null"`
	RefreshVersion int       `gorm:"column:refresh_version;default:1"`
	ExpiresAt      time.Time `gorm:"column:expires_at;not null"`
	Revoked        bool      `gorm:"column:revoked;default:false"`
	GrantedScopes  string    `gorm:"column:granted_scopes;type:text"` // space-separated scopes

	IPAddress string `gorm:"column:ip_address;type:text"`
	UserAgent string `gorm:"column:user_agent;type:text"`
	Device    string `gorm:"column:device;type:text"`

	LastUsedAt time.Time `gorm:"column:last_used_at"`
}

func (Session) TableName() string {
	return "sessions"
}
