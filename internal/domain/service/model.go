package service

import (
	"time"

	"github.com/Anvoria/authly/internal/database"
	"github.com/google/uuid"
)

// DefaultAuthlyServiceID is the fixed UUID for the default authly system service
const DefaultAuthlyServiceID = "00000000-0000-0000-0000-000000000001"

// DefaultAuthlyServiceCode is the code for the default authly service
const DefaultAuthlyServiceCode = "authly"

// Service represents a service in the system
type Service struct {
	database.BaseModel
	Code        string `gorm:"column:code;unique;not null;size:50"`
	Name        string `gorm:"column:name;not null;size:255"`
	Description string `gorm:"column:description;type:text"`
	Active      bool   `gorm:"column:active;default:true"`
	IsSystem    bool   `gorm:"column:is_system;default:false"`
}

func (Service) TableName() string {
	return "services"
}

// ServiceResponse represents a safe service response
type ServiceResponse struct {
	ID          uuid.UUID `json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Active      bool      `json:"active"`
	IsSystem    bool      `json:"is_system"`
}

// ToResponse converts a Service to ServiceResponse
func (s *Service) ToResponse() *ServiceResponse {
	return &ServiceResponse{
		ID:          s.ID,
		CreatedAt:   s.CreatedAt,
		UpdatedAt:   s.UpdatedAt,
		Code:        s.Code,
		Name:        s.Name,
		Description: s.Description,
		Active:      s.Active,
		IsSystem:    s.IsSystem,
	}
}
