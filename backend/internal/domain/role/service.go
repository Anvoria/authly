package role

import (
	"github.com/Anvoria/authly/internal/domain/permission"
	"github.com/google/uuid"
)

// Service defines the interface for role business logic
type Service interface {
	CreateRole(role *Role) error
	GetRole(id string) (*Role, error)
	GetRoleByName(serviceID, name string) (*Role, error)
	GetRolesByService(serviceID string) ([]*Role, error)
	GetDefaultRole(serviceID string) (*Role, error)
	UpdateRole(role *Role) error
	DeleteRole(id string) error
	AssignRole(userID, roleID string) error
	AssignDefaultRoles(userID string) error
}

type service struct {
	repo           Repository
	permissionRepo permission.Repository
}

// NewService returns a Service implementation backed by the provided role and permission repositories.
 // The returned Service uses repo for role persistence and permissionRepo for user permission operations.
func NewService(repo Repository, permissionRepo permission.Repository) Service {
	return &service{
		repo:           repo,
		permissionRepo: permissionRepo,
	}
}

func (s *service) CreateRole(role *Role) error {
	if role.IsDefault {
		if err := s.unsetOtherDefaults(role.ServiceID.String(), ""); err != nil {
			return err
		}
	}
	return s.repo.Create(role)
}

func (s *service) GetRole(id string) (*Role, error) {
	return s.repo.FindByID(id)
}

func (s *service) GetRoleByName(serviceID, name string) (*Role, error) {
	return s.repo.FindByName(serviceID, name)
}

func (s *service) GetRolesByService(serviceID string) ([]*Role, error) {
	return s.repo.FindByServiceID(serviceID)
}

func (s *service) GetDefaultRole(serviceID string) (*Role, error) {
	return s.repo.FindDefaultByServiceID(serviceID)
}

// UpdateRole updates the role and syncs changes to all assigned users
func (s *service) UpdateRole(updatedRole *Role) error {
	// Fetch old role to calculate delta
	oldRole, err := s.repo.FindByID(updatedRole.ID.String())
	if err != nil {
		return err
	}

	// If is_default changed to true, unset others
	if updatedRole.IsDefault && !oldRole.IsDefault {
		if err := s.unsetOtherDefaults(updatedRole.ServiceID.String(), updatedRole.ID.String()); err != nil {
			return err
		}
	}

	if err := s.repo.Update(updatedRole); err != nil {
		return err
	}

	addedBits := updatedRole.Bitmask &^ oldRole.Bitmask
	removedBits := oldRole.Bitmask &^ updatedRole.Bitmask

	if addedBits == 0 && removedBits == 0 {
		return nil
	}

	return s.propagateRoleChanges(updatedRole.ID.String(), addedBits, removedBits)
}

func (s *service) DeleteRole(id string) error {
	return s.repo.Delete(id)
}

// AssignRole assigns a role to a user, overwriting their current role for that service
func (s *service) AssignRole(userID, roleID string) error {
	role, err := s.repo.FindByID(roleID)
	if err != nil {
		return err
	}

	userPerms, err := s.permissionRepo.FindUserPermissionsByUserIDAndServiceID(userID, role.ServiceID.String())
	if err != nil {
		return err
	}

	var userPerm *permission.UserPermission
	// Find the global permission (resource is null)
	for _, p := range userPerms {
		if p.Resource == nil {
			userPerm = p
			break
		}
	}

	if userPerm == nil {
		// Create new permission
		uid, _ := uuid.Parse(userID)
		rid, _ := uuid.Parse(roleID)
		userPerm = &permission.UserPermission{
			UserID:    uid,
			ServiceID: role.ServiceID,
			RoleID:    &rid,
			Bitmask:   role.Bitmask,
			Resource:  nil,
		}
		return s.permissionRepo.CreateUserPermission(userPerm)
	}

	// Update existing permission
	if userPerm.RoleID != nil {
		oldRole, err := s.repo.FindByID(userPerm.RoleID.String())
		if err == nil {
			// Remove old role's bits
			userPerm.Bitmask = userPerm.Bitmask &^ oldRole.Bitmask
		}
	}

	userPerm.Bitmask = userPerm.Bitmask | role.Bitmask

	rid, _ := uuid.Parse(roleID)
	userPerm.RoleID = &rid

	return s.permissionRepo.UpdateUserPermission(userPerm)
}

func (s *service) AssignDefaultRoles(userID string) error {
	roles, err := s.repo.FindAllDefaults()
	if err != nil {
		return err
	}

	for _, role := range roles {
		if err := s.AssignRole(userID, role.ID.String()); err != nil {
			return err
		}
	}
	return nil
}

func (s *service) propagateRoleChanges(roleID string, addedBits, removedBits uint64) error {
	userPerms, err := s.permissionRepo.FindUserPermissionsByRoleID(roleID)
	if err != nil {
		return err
	}

	for _, perm := range userPerms {
		// Apply delta
		perm.Bitmask = perm.Bitmask | addedBits
		perm.Bitmask = perm.Bitmask &^ removedBits

		if err := s.permissionRepo.UpdateUserPermission(perm); err != nil {
			return err
		}
		// Increment version to invalidate tokens
		if err := s.permissionRepo.IncrementPermissionVersion(perm.UserID.String()); err != nil {
			return err
		}
	}
	return nil
}

func (s *service) unsetOtherDefaults(serviceID, excludeRoleID string) error {
	roles, err := s.repo.FindByServiceID(serviceID)
	if err != nil {
		return err
	}
	for _, r := range roles {
		if r.IsDefault && r.ID.String() != excludeRoleID {
			r.IsDefault = false
			if err := s.repo.Update(r); err != nil {
				return err
			}
		}
	}
	return nil
}