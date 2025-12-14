"use client";

import { useSearchParams, useRouter } from "next/navigation";
import { Suspense, useEffect, useState, useCallback } from "react";
import AuthorizeLayout from "@/authly/components/authorize/AuthorizeLayout";
import { getMe } from "@/authly/lib/api";

function AuthorizePageContent() {
    const searchParams = useSearchParams();
    const router = useRouter();
    const [isLoading, setIsLoading] = useState(true);
    const [isAuthenticated, setIsAuthenticated] = useState(false);

    const redirectToLogin = useCallback(() => {
        const oidcParams = new URLSearchParams();
        searchParams.forEach((value, key) => {
            oidcParams.set(key, value);
        });

        const encodedParams = encodeURIComponent(oidcParams.toString());
        router.push(`/login?oidc_params=${encodedParams}`);
    }, [searchParams, router]);

    const checkAuthentication = useCallback(async () => {
        try {
            setIsLoading(true);
            const response = await getMe();

            if (response.success) {
                setIsAuthenticated(true);
            } else {
                setIsAuthenticated(false);
                redirectToLogin();
            }
        } catch {
            setIsAuthenticated(false);
            redirectToLogin();
        } finally {
            setIsLoading(false);
        }
    }, [redirectToLogin]);

    useEffect(() => {
        checkAuthentication();
    }, [checkAuthentication]);

    if (isLoading) {
        return (
            <AuthorizeLayout>
                <div className="flex items-center justify-center py-12">
                    <div className="text-white/60">Checking authentication...</div>
                </div>
            </AuthorizeLayout>
        );
    }

    if (!isAuthenticated) {
        return null;
    }

    return (
        <AuthorizeLayout>
            <div className="space-y-6">
                <div className="space-y-1">
                    <h2 className="text-xl font-semibold text-white">Authorization</h2>
                    <p className="text-sm text-white/60">Authorization flow will be implemented here</p>
                </div>
            </div>
        </AuthorizeLayout>
    );
}

export default function AuthorizePage() {
    return (
        <Suspense
            fallback={
                <div className="min-h-screen w-full flex items-center justify-center bg-black">
                    <div className="text-white/60">Loading...</div>
                </div>
            }
        >
            <AuthorizePageContent />
        </Suspense>
    );
}
