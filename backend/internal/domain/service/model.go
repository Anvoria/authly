package service

import (
	"time"

	"github.com/Anvoria/authly/internal/database"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

// DefaultAuthlyServiceID is the fixed UUID for the default authly system service
const DefaultAuthlyServiceID = "00000000-0000-0000-0000-000000000001"

// DefaultAuthlyClientID is the client_id for the default authly service
const DefaultAuthlyClientID = "authly_authly_00000000"

// Service represents a service in the system
type Service struct {
	database.BaseModel

	// OIDC
	ClientID     string `gorm:"column:client_id;not null;unique"`
	ClientSecret string `gorm:"column:client_secret;not null"`

	// Metadata
	Name        string `gorm:"column:name;not null;size:255"`
	Description string `gorm:"column:description;type:text"`

	// Security
	Domain        string         `gorm:"column:domain;unique;size:255"`
	RedirectURIs  pq.StringArray `gorm:"type:text[]"`
	AllowedScopes pq.StringArray `gorm:"type:text[]"`

	// Flags
	Active   bool `gorm:"column:active;default:true"`
	IsSystem bool `gorm:"column:is_system;default:false"`
}

func (Service) TableName() string {
	return "services"
}

// ServiceResponse represents a safe service response
type ServiceResponse struct {
	ID            uuid.UUID      `json:"id"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	ClientID      string         `json:"client_id"`
	ClientSecret  string         `json:"client_secret"`
	RedirectURIs  pq.StringArray `json:"redirect_uris"`
	AllowedScopes pq.StringArray `json:"allowed_scopes"`
	Active        bool           `json:"active"`
	IsSystem      bool           `json:"is_system"`
	Name          string         `json:"name"`
	Description   string         `json:"description"`
	Domain        string         `json:"domain"`
}

// ToResponse converts a Service to ServiceResponse
func (s *Service) ToResponse() *ServiceResponse {
	return &ServiceResponse{
		ID:            s.ID,
		CreatedAt:     s.CreatedAt,
		UpdatedAt:     s.UpdatedAt,
		ClientID:      s.ClientID,
		ClientSecret:  s.ClientSecret,
		RedirectURIs:  s.RedirectURIs,
		AllowedScopes: s.AllowedScopes,
		Name:          s.Name,
		Description:   s.Description,
		Domain:        s.Domain,
		Active:        s.Active,
		IsSystem:      s.IsSystem,
	}
}
