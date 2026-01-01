"use client";

import { useAuth } from "@/authly/components/providers/AuthProvider";
import { useRouter } from "next/navigation";
import { useEffect } from "react";

interface AuthGuardProps {
    children: React.ReactNode;
    requiredPermissions?: Record<string, number>;
    requireAll?: boolean;
    redirectTo?: string;
}

export default function AuthGuard({
    children,
    requiredPermissions,
    requireAll = false,
    redirectTo = "/auth/login",
}: AuthGuardProps) {
    const { isAuthenticated, isLoading, permissions } = useAuth();
    const router = useRouter();

    useEffect(() => {
        if (!isLoading && !isAuthenticated) {
            router.push(redirectTo);
        }
    }, [isLoading, isAuthenticated, router, redirectTo]);

    if (isLoading) {
        return (
            <div className="flex min-h-screen items-center justify-center bg-black text-white font-mono">
                LOADING...
            </div>
        );
    }

    if (!isAuthenticated) {
        return null;
    }

    if (requiredPermissions) {
        const checkPermission = ([serviceId, requiredBitmask]: [string, number]) => {
            const userBitmask = permissions[serviceId] || 0;
            return (userBitmask & requiredBitmask) === requiredBitmask;
        };

        const isAuthorized = requireAll
            ? Object.entries(requiredPermissions).every(checkPermission)
            : Object.entries(requiredPermissions).some(checkPermission);

        if (!isAuthorized) {
            return null;
        }
    }

    return <>{children}</>;
}
