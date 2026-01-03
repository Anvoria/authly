package permission

import (
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Repository interface for permission operations
type Repository interface {
	WithTx(tx *gorm.DB) Repository
	// Permissions
	CreatePermission(permission *Permission) error
	FindPermissionByID(id string) (*Permission, error)
	FindPermissionsByServiceID(serviceID string, limit, offset int) ([]*Permission, int64, error)
	FindPermissionsByServiceIDAndResource(serviceID string, resource *string, limit, offset int) ([]*Permission, int64, error)
	FindActivePermissionsByServiceID(serviceID string) ([]*Permission, error)
	FindActivePermissionsByServiceIDAndResource(serviceID string, resource *string) ([]*Permission, error)
	UpdatePermission(permission *Permission) error
	DeletePermission(id string) error

	// User Permissions
	CreateUserPermission(userPerm *UserPermission) error
	FindUserPermission(userID, serviceID string, resource *string) (*UserPermission, error)
	FindUserPermissionsByUserID(userID string) ([]*UserPermission, error)
	FindUserPermissionsByUserIDAndServiceID(userID, serviceID string) ([]*UserPermission, error)
	FindUserPermissionsByRoleID(roleID string) ([]*UserPermission, error)
	UpdateUserPermission(userPerm *UserPermission) error
	DeleteUserPermission(userID, serviceID string, resource *string) error
	IncrementPermissionVersion(userID string) error

	// Service Permissions
	CreateServicePermission(servicePerm *ServicePermission) error
	FindServicePermission(clientID, targetServiceID string, resource *string) (*ServicePermission, error)
	FindServicePermissionsByClientID(clientID string) ([]*ServicePermission, error)
	UpdateServicePermission(servicePerm *ServicePermission) error
	DeleteServicePermission(clientID, targetServiceID string, resource *string) error
}

// repository struct for permission operations
type repository struct {
	db *gorm.DB
}

// NewRepository creates a Repository backed by the provided *gorm.DB.
func NewRepository(db *gorm.DB) Repository {
	return &repository{db}
}

// WithTx returns a new repository instance with the provided transaction
func (r *repository) WithTx(tx *gorm.DB) Repository {
	return &repository{db: tx}
}

// CreatePermission creates a new permission
func (r *repository) CreatePermission(permission *Permission) error {
	return r.db.Create(permission).Error
}

// FindPermissionByID gets a permission by ID
func (r *repository) FindPermissionByID(id string) (*Permission, error) {
	var permission Permission
	if err := r.db.Where("id = ?", id).First(&permission).Error; err != nil {
		return nil, err
	}
	return &permission, nil
}

// FindPermissionsByServiceID gets all permissions for a service with pagination
func (r *repository) FindPermissionsByServiceID(serviceID string, limit, offset int) ([]*Permission, int64, error) {
	var permissions []*Permission
	var count int64

	if err := r.db.Model(&Permission{}).Where("service_id = ?", serviceID).Count(&count).Error; err != nil {
		return nil, 0, err
	}

	if err := r.db.Where("service_id = ?", serviceID).Limit(limit).Offset(offset).Find(&permissions).Error; err != nil {
		return nil, 0, err
	}
	return permissions, count, nil
}

// FindPermissionsByServiceIDAndResource gets all permissions for a service and resource with pagination
func (r *repository) FindPermissionsByServiceIDAndResource(serviceID string, resource *string, limit, offset int) ([]*Permission, int64, error) {
	var permissions []*Permission
	var count int64
	query := r.db.Model(&Permission{}).Where("service_id = ?", serviceID)

	if resource == nil {
		query = query.Where("resource IS NULL")
	} else {
		query = query.Where("resource = ?", *resource)
	}

	if err := query.Count(&count).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Limit(limit).Offset(offset).Find(&permissions).Error; err != nil {
		return nil, 0, err
	}
	return permissions, count, nil
}

// FindActivePermissionsByServiceID gets all active permissions for a service
func (r *repository) FindActivePermissionsByServiceID(serviceID string) ([]*Permission, error) {
	var permissions []*Permission
	if err := r.db.Where("service_id = ? AND active = ?", serviceID, true).Find(&permissions).Error; err != nil {
		return nil, err
	}
	return permissions, nil
}

// FindActivePermissionsByServiceIDAndResource gets all active permissions for a service and resource
func (r *repository) FindActivePermissionsByServiceIDAndResource(serviceID string, resource *string) ([]*Permission, error) {
	var permissions []*Permission
	query := r.db.Where("service_id = ? AND active = ?", serviceID, true)

	if resource == nil {
		query = query.Where("resource IS NULL")
	} else {
		query = query.Where("resource = ?", *resource)
	}

	if err := query.Find(&permissions).Error; err != nil {
		return nil, err
	}
	return permissions, nil
}

// UpdatePermission updates a permission
func (r *repository) UpdatePermission(permission *Permission) error {
	return r.db.Save(permission).Error
}

// DeletePermission deletes a permission (soft delete)
func (r *repository) DeletePermission(id string) error {
	return r.db.Delete(&Permission{}, id).Error
}

// CreateUserPermission creates or updates a user permission
func (r *repository) CreateUserPermission(userPerm *UserPermission) error {
	// Using raw SQL to handle the functional unique index on COALESCE(resource, '')
	// GORM's Clauses/OnConflict doesn't easily support functional indexes without raw SQL.
	query := `
		INSERT INTO user_permissions (user_id, service_id, resource, bitmask, permission_v, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, NOW(), NOW())
		ON CONFLICT (user_id, service_id, COALESCE(resource, ''))
		DO UPDATE SET 
			bitmask = EXCLUDED.bitmask, 
			permission_v = EXCLUDED.permission_v, 
			updated_at = NOW()
		RETURNING id, created_at, updated_at
	`

	return r.db.Raw(query,
		userPerm.UserID,
		userPerm.ServiceID,
		userPerm.Resource,
		userPerm.Bitmask,
		userPerm.PermissionV,
	).Scan(userPerm).Error
}

// FindUserPermission gets a user's permission for a specific service and resource
func (r *repository) FindUserPermission(userID, serviceID string, resource *string) (*UserPermission, error) {
	var userPerm UserPermission
	query := r.db.Where("user_id = ? AND service_id = ?", userID, serviceID)

	if resource == nil {
		query = query.Where("resource IS NULL")
	} else {
		query = query.Where("resource = ?", *resource)
	}

	if err := query.First(&userPerm).Error; err != nil {
		return nil, err
	}
	return &userPerm, nil
}

// FindUserPermissionsByUserID gets all permissions for a user
func (r *repository) FindUserPermissionsByUserID(userID string) ([]*UserPermission, error) {
	var userPerms []*UserPermission
	if err := r.db.Where("user_id = ?", userID).Find(&userPerms).Error; err != nil {
		return nil, err
	}
	return userPerms, nil
}

// FindUserPermissionsByUserIDAndServiceID gets all permissions for a user for a specific service
func (r *repository) FindUserPermissionsByUserIDAndServiceID(userID, serviceID string) ([]*UserPermission, error) {
	var userPerms []*UserPermission
	if err := r.db.Where("user_id = ? AND service_id = ?", userID, serviceID).Find(&userPerms).Error; err != nil {
		return nil, err
	}
	return userPerms, nil
}

// FindUserPermissionsByRoleID gets all user permissions associated with a specific role
func (r *repository) FindUserPermissionsByRoleID(roleID string) ([]*UserPermission, error) {
	var userPerms []*UserPermission
	if err := r.db.Where("role_id = ?", roleID).Find(&userPerms).Error; err != nil {
		return nil, err
	}
	return userPerms, nil
}

// UpdateUserPermission updates a user permission
func (r *repository) UpdateUserPermission(userPerm *UserPermission) error {
	return r.db.Save(userPerm).Error
}

// DeleteUserPermission deletes a user permission (soft delete)
func (r *repository) DeleteUserPermission(userID, serviceID string, resource *string) error {
	query := r.db.Where("user_id = ? AND service_id = ?", userID, serviceID)

	if resource == nil {
		query = query.Where("resource IS NULL")
	} else {
		query = query.Where("resource = ?", *resource)
	}

	return query.Delete(&UserPermission{}).Error
}

// IncrementPermissionVersion increments the permission version for all user's permissions
// This invalidates cached tokens
func (r *repository) IncrementPermissionVersion(userID string) error {
	return r.db.Model(&UserPermission{}).
		Where("user_id = ?", userID).
		Update("permission_v", gorm.Expr("permission_v + 1")).Error
}

// CreateServicePermission creates or updates a service permission
// On conflict with the unique constraint (client_id, target_service_id, resource),
// it updates the bitmask.
func (r *repository) CreateServicePermission(servicePerm *ServicePermission) error {
	return r.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "client_id"},
			{Name: "target_service_id"},
			{Name: "resource"},
		},
		DoUpdates: clause.AssignmentColumns([]string{"bitmask", "updated_at"}),
	}).Create(servicePerm).Error
}

// FindServicePermission gets a service's permission for a specific target service and resource
func (r *repository) FindServicePermission(clientID, targetServiceID string, resource *string) (*ServicePermission, error) {
	var servicePerm ServicePermission
	query := r.db.Where("client_id = ? AND target_service_id = ?", clientID, targetServiceID)

	if resource == nil {
		query = query.Where("resource IS NULL")
	} else {
		query = query.Where("resource = ?", *resource)
	}

	if err := query.First(&servicePerm).Error; err != nil {
		return nil, err
	}
	return &servicePerm, nil
}

// FindServicePermissionsByClientID gets all permissions for a client service
func (r *repository) FindServicePermissionsByClientID(clientID string) ([]*ServicePermission, error) {
	var servicePerms []*ServicePermission
	if err := r.db.Where("client_id = ?", clientID).Find(&servicePerms).Error; err != nil {
		return nil, err
	}
	return servicePerms, nil
}

// UpdateServicePermission updates a service permission
func (r *repository) UpdateServicePermission(servicePerm *ServicePermission) error {
	return r.db.Save(servicePerm).Error
}

// DeleteServicePermission deletes a service permission (soft delete)
func (r *repository) DeleteServicePermission(clientID, targetServiceID string, resource *string) error {
	query := r.db.Where("client_id = ? AND target_service_id = ?", clientID, targetServiceID)

	if resource == nil {
		query = query.Where("resource IS NULL")
	} else {
		query = query.Where("resource = ?", *resource)
	}

	return query.Delete(&ServicePermission{}).Error
}
