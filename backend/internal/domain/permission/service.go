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

	// HasAnyPermission checks if a user has any permission for a service (across all resources)
	HasAnyPermission(userID, serviceID string) (bool, error)

	// IncrementPermissionVersion increments permission version (invalidates tokens)
	IncrementPermissionVersion(userID string) error

	// GetPermissionVersion gets the current permission version for a user
	GetPermissionVersion(userID string) (int, error)

	// BuildServiceScopes builds a map of client_ids (with optional resource) to bitmasks for a service (client)
	BuildServiceScopes(clientID string) (map[string]uint64, error)

	// GetServicePermission gets a service's permission bitmask for a target service and resource
	// resource can be empty string for global service permissions
	GetServicePermission(clientID, targetServiceID, resource string) (uint64, error)

	// GrantServicePermission grants a permission bit to a service for a target service and resource
	// resource can be empty string for global service permissions
	GrantServicePermission(clientID, targetServiceID, resource string, bit uint8) error

	// RevokeServicePermission revokes a permission bit from a service for a target service and resource
	// resource can be empty string for global service permissions
	RevokeServicePermission(clientID, targetServiceID, resource string, bit uint8) error

	// HasServicePermission checks if a service has a specific permission bit for a target service and resource
	// resource can be empty string for global service permissions
	HasServicePermission(clientID, targetServiceID, resource string, bit uint8) (bool, error)

	// Permission Management
	CreatePermission(serviceID, name string, bit uint8, resource *string) (*Permission, error)
	ListPermissions(serviceID string, limit, offset int) ([]*Permission, error)
	UpdatePermission(id, name string, active bool) (*Permission, error)
	DeletePermission(id string) error
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

// HasAnyPermission checks if a user has any permission for a service (across all resources)
func (s *serviceImpl) HasAnyPermission(userID, serviceID string) (bool, error) {
	userPerms, err := s.repo.FindUserPermissionsByUserIDAndServiceID(userID, serviceID)
	if err != nil {
		return false, err
	}

	for _, perm := range userPerms {
		if perm.Bitmask > 0 {
			return true, nil
		}
	}

	return false, nil
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

// BuildServiceScopes builds a map of client_ids (with optional resource) to bitmasks for a service (client)
func (s *serviceImpl) BuildServiceScopes(clientID string) (map[string]uint64, error) {
	servicePerms, err := s.repo.FindServicePermissionsByClientID(clientID)
	if err != nil {
		return nil, fmt.Errorf("failed to get service permissions: %w", err)
	}

	scopes := make(map[string]uint64)
	for _, servicePerm := range servicePerms {
		// Get target service client_id
		targetService, err := s.serviceRepo.FindByID(servicePerm.TargetServiceID.String())
		if err != nil {
			// Skip if service not found
			continue
		}

		if servicePerm.Bitmask > 0 {
			scopeKey := targetService.ClientID
			if servicePerm.Resource != nil && *servicePerm.Resource != "" {
				scopeKey = fmt.Sprintf("%s:%s", targetService.ClientID, *servicePerm.Resource)
			}
			scopes[scopeKey] = servicePerm.Bitmask
		}
	}

	return scopes, nil
}

// GetServicePermission gets a service's permission bitmask for a target service and resource
func (s *serviceImpl) GetServicePermission(clientID, targetServiceID, resource string) (uint64, error) {
	var resourcePtr *string
	if resource != "" {
		resourcePtr = &resource
	}

	servicePerm, err := s.repo.FindServicePermission(clientID, targetServiceID, resourcePtr)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, nil
		}
		return 0, err
	}
	return servicePerm.Bitmask, nil
}

// GrantServicePermission grants a permission bit to a service for a target service and resource
func (s *serviceImpl) GrantServicePermission(clientID, targetServiceID, resource string, bit uint8) error {
	if bit > 63 {
		return fmt.Errorf("invalid bit position: %d", bit)
	}

	var resourcePtr *string
	if resource != "" {
		resourcePtr = &resource
	}

	servicePerm, err := s.repo.FindServicePermission(clientID, targetServiceID, resourcePtr)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			targetServiceIDUUID, err := uuid.Parse(targetServiceID)
			if err != nil {
				return fmt.Errorf("invalid target service ID: %w", err)
			}

			servicePerm = &ServicePermission{
				ClientID:        clientID,
				TargetServiceID: targetServiceIDUUID,
				Resource:        resourcePtr,
				Bitmask:         SetBit(0, bit),
			}
			if err := s.repo.CreateServicePermission(servicePerm); err != nil {
				return fmt.Errorf("failed to create service permission: %w", err)
			}
		} else {
			return fmt.Errorf("failed to get service permission: %w", err)
		}
	} else {
		servicePerm.Bitmask = SetBit(servicePerm.Bitmask, bit)
		if err := s.repo.UpdateServicePermission(servicePerm); err != nil {
			return fmt.Errorf("failed to update service permission: %w", err)
		}
	}

	return nil
}

// RevokeServicePermission revokes a permission bit from a service for a target service and resource
func (s *serviceImpl) RevokeServicePermission(clientID, targetServiceID, resource string, bit uint8) error {
	if bit > 63 {
		return fmt.Errorf("invalid bit position: %d", bit)
	}

	var resourcePtr *string
	if resource != "" {
		resourcePtr = &resource
	}

	servicePerm, err := s.repo.FindServicePermission(clientID, targetServiceID, resourcePtr)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return fmt.Errorf("failed to get service permission: %w", err)
	}

	servicePerm.Bitmask = ClearBit(servicePerm.Bitmask, bit)
	if err := s.repo.UpdateServicePermission(servicePerm); err != nil {
		return fmt.Errorf("failed to update service permission: %w", err)
	}

	return nil
}

// HasServicePermission checks if a service has a specific permission bit for a target service and resource
func (s *serviceImpl) HasServicePermission(clientID, targetServiceID, resource string, bit uint8) (bool, error) {
	bitmask, err := s.GetServicePermission(clientID, targetServiceID, resource)
	if err != nil {
		return false, err
	}
	return HasBit(bitmask, bit), nil
}

func (s *serviceImpl) CreatePermission(serviceID, name string, bit uint8, resource *string) (*Permission, error) {
	serviceIDUUID, err := uuid.Parse(serviceID)
	if err != nil {
		return nil, fmt.Errorf("invalid service ID: %w", err)
	}

	perm := &Permission{
		ServiceID: serviceIDUUID,
		Name:      name,
		Bit:       bit,
		Resource:  resource,
		Active:    true,
	}
	if err := s.repo.CreatePermission(perm); err != nil {
		return nil, err
	}
	return perm, nil
}

func (s *serviceImpl) ListPermissions(serviceID string, limit, offset int) ([]*Permission, error) {
	return s.repo.FindPermissionsByServiceID(serviceID, limit, offset)
}

func (s *serviceImpl) UpdatePermission(id, name string, active bool) (*Permission, error) {
	perm, err := s.repo.FindPermissionByID(id)
	if err != nil {
		return nil, err
	}

	perm.Name = name
	perm.Active = active

	if err := s.repo.UpdatePermission(perm); err != nil {
		return nil, err
	}
	return perm, nil
}

func (s *serviceImpl) DeletePermission(id string) error {
	return s.repo.DeletePermission(id)
}
