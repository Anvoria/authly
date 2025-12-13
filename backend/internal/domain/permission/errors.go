package permission

import "errors"

var (
	// ErrPermissionNotFound is returned when a permission is not found
	ErrPermissionNotFound = errors.New("permission not found")

	// ErrUserPermissionNotFound is returned when a user permission is not found
	ErrUserPermissionNotFound = errors.New("user permission not found")

	// ErrInvalidBitPosition is returned when a bit position is invalid (must be 0-63)
	ErrInvalidBitPosition = errors.New("invalid bit position (must be 0-63)")

	// ErrPermissionAlreadyExists is returned when trying to create a duplicate permission
	ErrPermissionAlreadyExists = errors.New("permission already exists for this service and bit")
)
