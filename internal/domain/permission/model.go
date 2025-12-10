package permission

import (
	"time"

	"github.com/Anvoria/authly/internal/database"
	"github.com/google/uuid"
)

// Permission represents a permission/scope for a service
type Permission struct {
	database.BaseModel
	ServiceID uuid.UUID `gorm:"column:service_id;type:uuid;not null"`
	Resource  *string   `gorm:"column:resource;size:100"` // NULL = global service permission
	Bit       uint8     `gorm:"column:bit;not null"`
	Name      string    `gorm:"column:name;not null;size:100"`
	Active    bool      `gorm:"column:active;default:true"`
}

func (Permission) TableName() string {
	return "permissions"
}

// UserPermission represents a user's combined permissions for a service
type UserPermission struct {
	database.BaseModel
	UserID      uuid.UUID `gorm:"column:user_id;type:uuid;not null"`
	ServiceID   uuid.UUID `gorm:"column:service_id;type:uuid;not null"`
	Resource    *string   `gorm:"column:resource;size:100"`
	Bitmask     uint64    `gorm:"column:bitmask;not null;default:0"`
	PermissionV int       `gorm:"column:permission_v;not null;default:1"`
}

func (UserPermission) TableName() string {
	return "user_permissions"
}

// PermissionResponse represents a safe response for a permission
type PermissionResponse struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	ServiceID uuid.UUID `json:"service_id"`
	Resource  *string   `json:"resource,omitempty"`
	Bit       uint8     `json:"bit"`
	Name      string    `json:"name"`
	Active    bool      `json:"active"`
}

// ToResponse converts a Permission to a safe response
func (p *Permission) ToResponse() *PermissionResponse {
	return &PermissionResponse{
		ID:        p.ID,
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
		ServiceID: p.ServiceID,
		Resource:  p.Resource,
		Bit:       p.Bit,
		Name:      p.Name,
		Active:    p.Active,
	}
}

// Common permission bits
const (
	BitRead   uint8 = 0 // Read access
	BitWrite  uint8 = 1 // Write access
	BitDelete uint8 = 2 // Delete access
	BitAdmin  uint8 = 3 // Admin access
)

// Authly system management permission bits
const (
	BitManageServices    uint8 = 4 // Manage services (create, update, delete)
	BitManagePermissions uint8 = 5 // Manage permissions
	BitManageUsers       uint8 = 6 // Manage users
	BitManageRoles       uint8 = 7 // Manage roles (for future use)
	BitSystemAdmin       uint8 = 8 // Full system administration access
)

// Common permission names (can be extended per service)
const (
	PermRead   = "read"
	PermWrite  = "write"
	PermDelete = "delete"
	PermAdmin  = "admin"
)

// Authly system management permission names
const (
	PermManageServices    = "manage_services"
	PermManagePermissions = "manage_permissions"
	PermManageUsers       = "manage_users"
	PermManageRoles       = "manage_roles"
	PermSystemAdmin       = "system_admin"
)

// SetBit sets the specified bit position in bitmask and returns the resulting bitmask.
// If bit is greater than 63 the original bitmask is returned unchanged.
func SetBit(bitmask uint64, bit uint8) uint64 {
	if bit > 63 {
		return bitmask // return the bitmask unchanged if the bit is greater than 63
	}
	return bitmask | (1 << bit)
}

// ClearBit clears the specified bit in bitmask and returns the resulting mask.
// If bit is greater than 63 the original bitmask is returned unchanged.
func ClearBit(bitmask uint64, bit uint8) uint64 {
	if bit > 63 {
		return bitmask // return the bitmask unchanged if the bit is greater than 63
	}
	return bitmask &^ (1 << bit)
}

// HasBit reports whether the specified bit index is set in bitmask.
// If bit is greater than 63, it returns false.
func HasBit(bitmask uint64, bit uint8) bool {
	if bit > 63 {
		return false
	}
	return (bitmask & (1 << bit)) != 0
}

// HasAny reports whether any of the specified bit positions are set in the given bitmask.
// If no bit positions are provided, it returns false.
func HasAny(bitmask uint64, bits ...uint8) bool {
	for _, bit := range bits {
		if HasBit(bitmask, bit) {
			return true
		}
	}
	return false
}

// HasAll reports whether all specified bits are set in the given bitmask.
// HasAll reports whether all of the specified bit positions are set in bitmask.
// If no bits are provided it returns true. Bits with an index greater than 63 are treated as not set.
func HasAll(bitmask uint64, bits ...uint8) bool {
	for _, bit := range bits {
		if !HasBit(bitmask, bit) {
			return false
		}
	}
	return true
}