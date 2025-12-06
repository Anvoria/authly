package permission

import (
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Repository interface for permission operations
type Repository interface {
	// Permissions
	CreatePermission(permission *Permission) error
	FindPermissionByID(id string) (*Permission, error)
	FindPermissionsByServiceID(serviceID string) ([]*Permission, error)
	FindPermissionsByServiceIDAndResource(serviceID string, resource *string) ([]*Permission, error)
	FindActivePermissionsByServiceID(serviceID string) ([]*Permission, error)
	FindActivePermissionsByServiceIDAndResource(serviceID string, resource *string) ([]*Permission, error)
	UpdatePermission(permission *Permission) error
	DeletePermission(id string) error

	// User Permissions
	CreateUserPermission(userPerm *UserPermission) error
	FindUserPermission(userID, serviceID string, resource *string) (*UserPermission, error)
	FindUserPermissionsByUserID(userID string) ([]*UserPermission, error)
	UpdateUserPermission(userPerm *UserPermission) error
	DeleteUserPermission(userID, serviceID string, resource *string) error
	IncrementPermissionVersion(userID string) error
}

// repository struct for permission operations
type repository struct {
	db *gorm.DB
}

// NewRepository creates a Repository backed by the provided *gorm.DB.
func NewRepository(db *gorm.DB) Repository {
	return &repository{db}
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

// FindPermissionsByServiceID gets all permissions for a service
func (r *repository) FindPermissionsByServiceID(serviceID string) ([]*Permission, error) {
	var permissions []*Permission
	if err := r.db.Where("service_id = ?", serviceID).Find(&permissions).Error; err != nil {
		return nil, err
	}
	return permissions, nil
}

// FindPermissionsByServiceIDAndResource gets all permissions for a service and resource
func (r *repository) FindPermissionsByServiceIDAndResource(serviceID string, resource *string) ([]*Permission, error) {
	var permissions []*Permission
	query := r.db.Where("service_id = ?", serviceID)

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
// On conflict with the unique constraint (user_id, service_id, resource),
// it updates the bitmask and permission_v columns.
func (r *repository) CreateUserPermission(userPerm *UserPermission) error {
	return r.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "user_id"},
			{Name: "service_id"},
			{Name: "resource"},
		},
		DoUpdates: clause.AssignmentColumns([]string{"bitmask", "permission_v", "updated_at"}),
	}).Create(userPerm).Error
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
