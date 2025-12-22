package role

import (
	"fmt"

	"github.com/Anvoria/authly/internal/domain/permission"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Service defines the interface for role business logic
type Service interface {
	WithTx(tx *gorm.DB) Service
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
	db             *gorm.DB
	repo           Repository
	permissionRepo permission.Repository
}

// NewService returns a Service implementation configured with the provided database handle,
// role repository, and permission repository.
func NewService(db *gorm.DB, repo Repository, permissionRepo permission.Repository) Service {
	return &service{
		db:             db,
		repo:           repo,
		permissionRepo: permissionRepo,
	}
}

func (s *service) WithTx(tx *gorm.DB) Service {
	return &service{
		db:             tx,
		repo:           s.repo.WithTx(tx),
		permissionRepo: s.permissionRepo.WithTx(tx),
	}
}

func (s *service) CreateRole(role *Role) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		txService := s.WithTx(tx)

		txSvc, ok := txService.(*service)
		if !ok {
			return fmt.Errorf("internal error: failed to cast service")
		}

		if role.IsDefault {
			if err := txSvc.unsetOtherDefaults(role.ServiceID.String(), ""); err != nil {
				return err
			}
		}
		return txSvc.repo.Create(role)
	})
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
	return s.db.Transaction(func(tx *gorm.DB) error {
		txService := s.WithTx(tx)
		txSvc, ok := txService.(*service)
		if !ok {
			return fmt.Errorf("internal error: failed to cast service")
		}

		oldRole, err := txSvc.repo.FindByID(updatedRole.ID.String())
		if err != nil {
			return err
		}

		if updatedRole.IsDefault && !oldRole.IsDefault {
			if err := txSvc.unsetOtherDefaults(updatedRole.ServiceID.String(), updatedRole.ID.String()); err != nil {
				return err
			}
		}

		if err := txSvc.repo.Update(updatedRole); err != nil {
			return err
		}

		addedBits := updatedRole.Bitmask &^ oldRole.Bitmask
		removedBits := oldRole.Bitmask &^ updatedRole.Bitmask

		if addedBits == 0 && removedBits == 0 {
			return nil
		}

		return txSvc.propagateRoleChanges(updatedRole.ID.String(), addedBits, removedBits)
	})
}

func (s *service) DeleteRole(id string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		txSvc := s.WithTx(tx)
		svc, ok := txSvc.(*service)
		if !ok {
			return fmt.Errorf("internal error: failed to cast service")
		}

		role, err := svc.repo.FindByID(id)
		if err != nil {
			return err
		}

		userPerms, err := svc.permissionRepo.FindUserPermissionsByRoleID(id)
		if err != nil {
			return err
		}

		for _, perm := range userPerms {
			perm.Bitmask = perm.Bitmask &^ role.Bitmask

			perm.RoleID = nil

			if err := svc.permissionRepo.UpdateUserPermission(perm); err != nil {
				return fmt.Errorf("failed to update user permission cleanup: %w", err)
			}

			if err := svc.permissionRepo.IncrementPermissionVersion(perm.UserID.String()); err != nil {
				return err
			}
		}

		return svc.repo.Delete(id)
	})
}

// AssignRole assigns a role to a user, overwriting their current role for that service
func (s *service) AssignRole(userID, roleID string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		txService := s.WithTx(tx)
		txSvc, ok := txService.(*service)
		if !ok {
			return fmt.Errorf("internal error: failed to cast service")
		}

		role, err := txSvc.repo.FindByID(roleID)
		if err != nil {
			return err
		}

		userPerms, err := txSvc.permissionRepo.FindUserPermissionsByUserIDAndServiceID(userID, role.ServiceID.String())
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
			uid, err := uuid.Parse(userID)
			if err != nil {
				return err
			}

			rid, err := uuid.Parse(roleID)
			if err != nil {
				return err
			}

			userPerm = &permission.UserPermission{
				UserID:    uid,
				ServiceID: role.ServiceID,
				RoleID:    &rid,
				Bitmask:   role.Bitmask,
				Resource:  nil,
			}
			return txSvc.permissionRepo.CreateUserPermission(userPerm)
		}

		// Update existing permission
		if userPerm.RoleID != nil {
			oldRole, err := txSvc.repo.FindByID(userPerm.RoleID.String())
			if err != nil {
				return fmt.Errorf("failed to fetch old role %s for user %s: %w", userPerm.RoleID, userID, err)
			}
			// Remove old role's bits
			userPerm.Bitmask = userPerm.Bitmask &^ oldRole.Bitmask
		}

		userPerm.Bitmask = userPerm.Bitmask | role.Bitmask

		rid, err := uuid.Parse(roleID)
		if err != nil {
			return err
		}
		userPerm.RoleID = &rid

		if err := txSvc.permissionRepo.UpdateUserPermission(userPerm); err != nil {
			return err
		}

		// Increment permission version to invalidate tokens
		return txSvc.permissionRepo.IncrementPermissionVersion(userID)
	})
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
