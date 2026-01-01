import AuthlyAdminGuard from "@/authly/components/auth/AuthlyAdminGuard";

/**
 * Layout for the Authly Admin Panel.
 * Protects all sub-routes with AuthlyAdminGuard.
 */
export default function AdminLayout({ children }: { children: React.ReactNode }) {
    return <AuthlyAdminGuard>{children}</AuthlyAdminGuard>;
}
