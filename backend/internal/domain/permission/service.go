package permission

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ServiceInterface defines the interface for permission operations
type ServiceInterface interface {
	// BuildScopes builds a map of client_ids (with optional resource) to bitmasks for a user
	BuildScopes(userID string) (map[string]uint64, error)

	// GetUserPermission gets a user's permission bitmask for a service and resource
	// resource can be empty string for global service permissions
	GetUserPermission(userID, serviceID, resource string) (uint64, error)

	// GrantPermission grants a permission bit to a user for a service and resource
	// resource can be empty string for global service permissions
	GrantPermission(userID, serviceID, resource string, bit uint8) error

	// RevokePermission revokes a permission bit from a user for a service and resource
	// resource can be empty string for global service permissions
	RevokePermission(userID, serviceID, resource string, bit uint8) error

	// HasPermission checks if a user has a specific permission bit for a service and resource
	// resource can be empty string for global service permissions
	HasPermission(userID, serviceID, resource string, bit uint8) (bool, error)

	// IncrementPermissionVersion increments permission version (invalidates tokens)
	IncrementPermissionVersion(userID string) error

	// GetPermissionVersion gets the current permission version for a user
	GetPermissionVersion(userID string) (int, error)
}

// serviceImpl implements ServiceInterface
type serviceImpl struct {
	repo        Repository
	serviceRepo ServiceRepository
}

// ServiceRepository interface for getting client_ids by ID
// This should be implemented by domain/service.Repository
type ServiceRepository interface {
	FindByID(id string) (*ServiceModel, error)
}

// ServiceModel represents a service (to avoid circular dependency)
type ServiceModel struct {
	ID       uuid.UUID
	ClientID string
}

// NewService returns a ServiceInterface backed by the provided Repository and ServiceRepository.
// NewService creates a ServiceInterface backed by the provided Repository and ServiceRepository implementations.
// The returned service implementation uses repo for user permission data access and serviceRepo to resolve service metadata such as client IDs.
func NewService(repo Repository, serviceRepo ServiceRepository) ServiceInterface {
	return &serviceImpl{
		repo:        repo,
		serviceRepo: serviceRepo,
	}
}

// BuildScopes builds a map of client_ids (with optional resource) to bitmasks for a user
func (s *serviceImpl) BuildScopes(userID string) (map[string]uint64, error) {
	userPerms, err := s.repo.FindUserPermissionsByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user permissions: %w", err)
	}

	scopes := make(map[string]uint64)
	for _, userPerm := range userPerms {
		// Get service client_id
		service, err := s.serviceRepo.FindByID(userPerm.ServiceID.String())
		if err != nil {
			// Skip if service not found (it might have been deleted)
			continue
		}

		// Only include if the bitmask is greater than 0
		if userPerm.Bitmask > 0 {
			// Build scope key: "client_id:resource" or "client_id" if resource is NULL
			scopeKey := service.ClientID
			if userPerm.Resource != nil && *userPerm.Resource != "" {
				scopeKey = fmt.Sprintf("%s:%s", service.ClientID, *userPerm.Resource)
			}
			scopes[scopeKey] = userPerm.Bitmask
		}
	}

	return scopes, nil
}

// GetUserPermission gets a user's permission bitmask for a service and resource
// resource can be empty string for global service permissions
func (s *serviceImpl) GetUserPermission(userID, serviceID, resource string) (uint64, error) {
	var resourcePtr *string
	if resource != "" {
		resourcePtr = &resource
	}

	userPerm, err := s.repo.FindUserPermission(userID, serviceID, resourcePtr)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, nil // there are no permissions = 0 bitmask
		}
		return 0, err
	}
	return userPerm.Bitmask, nil
}

// GrantPermission grants a permission bit to a user for a service and resource
// resource can be empty string for global service permissions
func (s *serviceImpl) GrantPermission(userID, serviceID, resource string, bit uint8) error {
	if bit > 63 {
		return fmt.Errorf("invalid bit position: %d (must be 0-63)", bit)
	}

	var resourcePtr *string
	if resource != "" {
		resourcePtr = &resource
	}

	userPerm, err := s.repo.FindUserPermission(userID, serviceID, resourcePtr)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Create new user permission
			userIDUUID, err := uuid.Parse(userID)
			if err != nil {
				return fmt.Errorf("invalid user ID: %w", err)
			}
			serviceIDUUID, err := uuid.Parse(serviceID)
			if err != nil {
				return fmt.Errorf("invalid service ID: %w", err)
			}

			userPerm = &UserPermission{
				UserID:      userIDUUID,
				ServiceID:   serviceIDUUID,
				Resource:    resourcePtr,
				Bitmask:     SetBit(0, bit),
				PermissionV: 1,
			}
			if err := s.repo.CreateUserPermission(userPerm); err != nil {
				return fmt.Errorf("failed to create user permission: %w", err)
			}
		} else {
			return fmt.Errorf("failed to get user permission: %w", err)
		}
	} else {
		// Update existing permission
		userPerm.Bitmask = SetBit(userPerm.Bitmask, bit)
		if err := s.repo.UpdateUserPermission(userPerm); err != nil {
			return fmt.Errorf("failed to update user permission: %w", err)
		}
	}

	if err := s.repo.IncrementPermissionVersion(userID); err != nil {
		return fmt.Errorf("failed to increment permission version: %w", err)
	}

	return nil
}

// RevokePermission revokes a permission bit from a user for a service and resource
// resource can be empty string for global service permissions
func (s *serviceImpl) RevokePermission(userID, serviceID, resource string, bit uint8) error {
	if bit > 63 {
		return fmt.Errorf("invalid bit position: %d (must be 0-63)", bit)
	}

	var resourcePtr *string
	if resource != "" {
		resourcePtr = &resource
	}

	userPerm, err := s.repo.FindUserPermission(userID, serviceID, resourcePtr)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return fmt.Errorf("failed to get user permission: %w", err)
	}

	userPerm.Bitmask = ClearBit(userPerm.Bitmask, bit)
	if err := s.repo.UpdateUserPermission(userPerm); err != nil {
		return fmt.Errorf("failed to update user permission: %w", err)
	}

	// Increment permission version to invalidate cached tokens
	if err := s.repo.IncrementPermissionVersion(userID); err != nil {
		return fmt.Errorf("failed to increment permission version: %w", err)
	}

	return nil
}

// HasPermission checks if a user has a specific permission bit for a service and resource
// resource can be empty string for global service permissions
func (s *serviceImpl) HasPermission(userID, serviceID, resource string, bit uint8) (bool, error) {
	bitmask, err := s.GetUserPermission(userID, serviceID, resource)
	if err != nil {
		return false, err
	}
	return HasBit(bitmask, bit), nil
}

// IncrementPermissionVersion increments permission version (invalidates tokens)
func (s *serviceImpl) IncrementPermissionVersion(userID string) error {
	return s.repo.IncrementPermissionVersion(userID)
}

// GetPermissionVersion gets the current permission version for a user
func (s *serviceImpl) GetPermissionVersion(userID string) (int, error) {
	userPerms, err := s.repo.FindUserPermissionsByUserID(userID)
	if err != nil {
		return 1, err // Default to 1 if error
	}
	if len(userPerms) == 0 {
		return 1, nil // Default to 1 if no permissions
	}
	// All permissions should have the same version, so return the first one
	return userPerms[0].PermissionV, nil
}
