package permission

// HasSystemAdmin reports whether the BitSystemAdmin bit is set in the given bitmask.
func HasSystemAdmin(bitmask uint64) bool {
	return HasBit(bitmask, BitSystemAdmin)
}

// HasManageServices reports whether the manage services permission is set in the bitmask.
// It returns `true` if the manage services permission bit is set, `false` otherwise.
func HasManageServices(bitmask uint64) bool {
	return HasBit(bitmask, BitManageServices)
}

// HasManagePermissions reports whether the Manage Permissions bit is set in the provided permission bitmask.
// It returns true if the bit corresponding to manage permissions is set, false otherwise.
func HasManagePermissions(bitmask uint64) bool {
	return HasBit(bitmask, BitManagePermissions)
}

// HasManageUsers reports whether the Manage Users permission bit is set in the provided bitmask.
func HasManageUsers(bitmask uint64) bool {
	return HasBit(bitmask, BitManageUsers)
}

// HasManageRoles reports whether the manage roles permission bit is set in the provided bitmask.
// It returns `true` if the BitManageRoles bit is set, `false` otherwise.
func HasManageRoles(bitmask uint64) bool {
	return HasBit(bitmask, BitManageRoles)
}

// HasAnyManagementPermission reports whether bitmask includes any management permission bit.
// It is true when any of BitManageServices, BitManagePermissions, BitManageUsers,
// BitManageRoles, or BitSystemAdmin is set.
func HasAnyManagementPermission(bitmask uint64) bool {
	return HasAny(bitmask, BitManageServices, BitManagePermissions, BitManageUsers, BitManageRoles, BitSystemAdmin)
}

// HasAllManagementPermissions reports whether bitmask has all management permission bits set:
// Manage Services, Manage Permissions, Manage Users, and Manage Roles.
func HasAllManagementPermissions(bitmask uint64) bool {
	return HasAll(bitmask, BitManageServices, BitManagePermissions, BitManageUsers, BitManageRoles)
}

// GetAuthlyScopeKey returns the scope key for authly service
// GetAuthlyScopeKey constructs the scope key for the authly service for the given resource.
// GetAuthlyScopeKey returns the scope key for the authly service for the given resource.
// If resource is empty the key is "authly"; otherwise the key is "authly:<resource>".
func GetAuthlyScopeKey(resource string) string {
	if resource == "" {
		return "authly"
	}
	return "authly:" + resource
}