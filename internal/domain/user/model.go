package user

import (
	"time"

	"github.com/Anvoria/authly/internal/database"
	"github.com/google/uuid"
)

type User struct {
	database.BaseModel
	Username  string `gorm:"column:username;unique;not null"`
	FirstName string `gorm:"column:first_name;not null"`
	LastName  string `gorm:"column:last_name;not null"`
	Email     string `gorm:"column:email;unique"`
	Password  string `gorm:"column:password;not null"`
	IsActive  bool   `gorm:"column:is_active;default:true"`
}

func (User) TableName() string {
	return "users"
}

// UserResponse represents a safe user response
type UserResponse struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Username  string    `json:"username"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Email     string    `json:"email"`
	IsActive  bool      `json:"is_active"`
}

// ToResponse converts a User to UserResponse, excluding sensitive fields
func (u *User) ToResponse() *UserResponse {
	return &UserResponse{
		ID:        u.ID,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
		Username:  u.Username,
		FirstName: u.FirstName,
		LastName:  u.LastName,
		Email:     u.Email,
		IsActive:  u.IsActive,
	}
}
