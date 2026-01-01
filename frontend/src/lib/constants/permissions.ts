/**
 * Permission bits matching backend/internal/domain/permission/model.go
 */
export const Bit = {
    Read: 0,
    Write: 1,
    Delete: 2,
    Admin: 3,

    // Authly system management
    ManageServices: 4,
    ManagePermissions: 5,
    ManageUsers: 6,
    ManageRoles: 7,
    SystemAdmin: 8,
} as const;

/**
 * Helper to calculate bitmask from bits
 */
export const toBitmask = (...bits: number[]): number => {
    return bits.reduce((mask, bit) => mask | (1 << bit), 0);
};

// Common masks
export const AUTHLY_ADMIN_MASK = toBitmask(Bit.SystemAdmin);
export const AUTHLY_MANAGER_MASK = toBitmask(
    Bit.ManageServices,
    Bit.ManagePermissions,
    Bit.ManageUsers,
    Bit.ManageRoles,
);
