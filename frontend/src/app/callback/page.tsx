"use client";

import { useEffect, useRef, useState, Suspense } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { exchangeToken } from "@/authly/lib/api";
import LocalStorageTokenService from "@/authly/lib/globals/client/LocalStorageTokenService";
import { OIDC_CONFIG } from "@/authly/lib/config";
import AuthorizeLayout from "@/authly/components/authorize/AuthorizeLayout";
import Button from "@/authly/components/ui/Button";

function CallbackPageContent() {
    const router = useRouter();
    const searchParams = useSearchParams();
    const [error, setError] = useState<string | null>(null);
    const processedRef = useRef(false);

    useEffect(() => {
        if (processedRef.current) return;
        processedRef.current = true;

        const handleCallback = async () => {
            const code = searchParams.get("code");
            const errorParam = searchParams.get("error");
            const errorDescription = searchParams.get("error_description");

            if (errorParam) {
                setError(`${errorParam}: ${errorDescription || "Unknown error"}`);
                return;
            }

            if (!code) {
                setError("No authorization code found in URL");
                return;
            }

            const codeVerifier = LocalStorageTokenService.oidcCodeVerifier;
            if (!codeVerifier) {
                setError("No code verifier found. Please try logging in again.");
                return;
            }

            try {
                const response = await exchangeToken({
                    grant_type: "authorization_code",
                    code: code,
                    redirect_uri: OIDC_CONFIG.redirect_uri,
                    client_id: OIDC_CONFIG.client_id,
                    code_verifier: codeVerifier,
                });

                if ("error" in response) {
                    setError(`${response.error}: ${response.error_description}`);
                } else {
                    LocalStorageTokenService.setAccessToken(response.access_token);
                    LocalStorageTokenService.setOidcCodeVerifier("");

                    router.push("/dashboard");
                }
            } catch (err) {
                console.error("Token exchange failed", err);
                setError("An unexpected error occurred during token exchange.");
            }
        };

        handleCallback();
    }, [router, searchParams]);

    if (error) {
        return (
            <AuthorizeLayout>
                <div className="space-y-6">
                    <div className="space-y-1">
                        <h2 className="text-xl font-semibold text-red-500">Authentication Failed</h2>
                        <p className="text-sm text-white/60">{error}</p>
                    </div>

                    <div className="pt-1">
                        <Button fullWidth variant="primary" onClick={() => router.push("/login")}>
                            Back to Login
                        </Button>
                    </div>
                </div>
            </AuthorizeLayout>
        );
    }

    return (
        <AuthorizeLayout>
            <div className="flex flex-col items-center justify-center py-12 space-y-4">
                <div className="h-8 w-8 animate-spin rounded-full border-2 border-white/80 border-t-transparent"></div>
                <p className="text-sm text-white/60">Authenticating...</p>
            </div>
        </AuthorizeLayout>
    );
}

export default function CallbackPage() {
    return (
        <Suspense
            fallback={
                <div className="min-h-screen w-full flex items-center justify-center bg-black">
                    <div className="text-white/60">Loading...</div>
                </div>
            }
        >
            <CallbackPageContent />
        </Suspense>
    );
}
