package permission

// HasSystemAdmin checks if a bitmask has system admin permission
func HasSystemAdmin(bitmask uint64) bool {
	return HasBit(bitmask, BitSystemAdmin)
}

// HasManageServices checks if a bitmask has manage services permission
func HasManageServices(bitmask uint64) bool {
	return HasBit(bitmask, BitManageServices)
}

// HasManagePermissions checks if a bitmask has manage permissions permission
func HasManagePermissions(bitmask uint64) bool {
	return HasBit(bitmask, BitManagePermissions)
}

// HasManageUsers checks if a bitmask has manage users permission
func HasManageUsers(bitmask uint64) bool {
	return HasBit(bitmask, BitManageUsers)
}

// HasManageRoles checks if a bitmask has manage roles permission
func HasManageRoles(bitmask uint64) bool {
	return HasBit(bitmask, BitManageRoles)
}

// HasAnyManagementPermission checks if a bitmask has any management permission
func HasAnyManagementPermission(bitmask uint64) bool {
	return HasAny(bitmask, BitManageServices, BitManagePermissions, BitManageUsers, BitManageRoles, BitSystemAdmin)
}

// HasAllManagementPermissions checks if a bitmask has all management permissions
func HasAllManagementPermissions(bitmask uint64) bool {
	return HasAll(bitmask, BitManageServices, BitManagePermissions, BitManageUsers, BitManageRoles)
}

// GetAuthlyScopeKey returns the scope key for authly service
// resource can be empty string for global permissions
func GetAuthlyScopeKey(resource string) string {
	if resource == "" {
		return "authly"
	}
	return "authly:" + resource
}
