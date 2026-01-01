"use client";

import { useAuth } from "@/authly/components/providers/AuthProvider";
import { Bit } from "@/authly/lib/constants/permissions";
import { useRouter } from "next/navigation";
import { useEffect } from "react";

interface AuthlyAdminGuardProps {
    children: React.ReactNode;
    /**
     * Optional: Specific bit required (e.g. Bit.ManageUsers).
     * If not provided, defaults to Bit.SystemAdmin.
     */
    requiredBit?: number;
    redirectTo?: string;
}

/**
 * Special AuthGuard for Authly Management Panel.
 * It checks permissions for the "authly" service.
 */
export default function AuthlyAdminGuard({
    children,
    requiredBit = Bit.SystemAdmin,
    redirectTo = "/dashboard/profile",
}: AuthlyAdminGuardProps) {
    const { isAuthenticated, isLoading, permissions } = useAuth();
    const router = useRouter();

    const AUTHLY_CLIENT_ID = "authly_authly_00000000";

    useEffect(() => {
        if (!isLoading) {
            if (!isAuthenticated) {
                router.push("/auth/login");
                return;
            }

            const userBitmask = permissions[AUTHLY_CLIENT_ID] || 0;
            const hasPermission = (userBitmask & (1 << requiredBit)) !== 0;

            if (!hasPermission) {
                router.push(redirectTo);
            }
        }
    }, [isLoading, isAuthenticated, permissions, requiredBit, router, redirectTo]);

    if (isLoading) {
        return (
            <div className="flex min-h-screen items-center justify-center bg-black text-white font-mono">
                LOADING...
            </div>
        );
    }

    if (!isAuthenticated) return null;

    const userBitmask = permissions[AUTHLY_CLIENT_ID] || 0;
    const hasPermission = (userBitmask & (1 << requiredBit)) !== 0;

    if (!hasPermission) {
        return null;
    }

    return <>{children}</>;
}
