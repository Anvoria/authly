"use client";

import { useSearchParams, useRouter } from "next/navigation";
import { Suspense, useEffect, useState, useCallback } from "react";
import AuthorizeLayout from "@/authly/components/authorize/AuthorizeLayout";
import ConsentScreen from "@/authly/components/authorize/ConsentScreen";
import { validateAuthorizationParams, buildErrorRedirect } from "@/authly/lib/oidc";
import { validateAuthorizationRequest, checkAuthStatus, confirmAuthorization, type ApiError } from "@/authly/lib/api";

type AuthStep = "validating" | "consent" | "error";

interface ErrorState {
    title: string;
    message: string;
    redirect?: string;
}

/**
 * Render the authorization consent flow UI for an incoming OIDC authorization request.
 *
 * Validates the incoming request and client, checks requested scopes and authentication status,
 * and presents one of three UI states: validating, an error page with optional redirect, or a consent
 * screen. If the user is not authenticated, navigates to the login page with encoded OIDC parameters.
 * On approval or denial, builds the appropriate redirect (authorization code or error) and navigates there.
 *
 * @returns A JSX element representing the current authorization UI state (validating, error, or consent), or `null` when no UI should be rendered.
 */
function AuthorizePageContent() {
    const searchParams = useSearchParams();
    const router = useRouter();
    const [step, setStep] = useState<AuthStep>("validating");
    const [isLoading, setIsLoading] = useState(false);
    const [error, setError] = useState<ErrorState | null>(null);
    const [clientInfo, setClientInfo] = useState<{
        name: string;
        logo_url?: string;
    } | null>(null);
    const [authParams, setAuthParams] = useState<{
        client_id: string;
        redirect_uri: string;
        scope: string;
        state: string;
        code_challenge: string;
        code_challenge_method: string;
    } | null>(null);
    const [scopes, setScopes] = useState<string[]>([]);

    const initializeAuthorization = useCallback(async () => {
        try {
            setStep("validating");
            setError(null);

            const validation = validateAuthorizationParams(searchParams);

            if (!validation.valid) {
                const redirectUri = searchParams.get("redirect_uri");
                let errorRedirect: string | undefined;

                if (redirectUri) {
                    try {
                        errorRedirect = buildErrorRedirect(redirectUri, validation.error!);
                    } catch {
                        errorRedirect = undefined;
                    }
                }

                setError({
                    title: "Invalid Request",
                    message: validation.error!.error_description,
                    redirect: errorRedirect,
                });
                setStep("error");
                return;
            }

            const params = validation.params!;

            const clientValidation = await validateAuthorizationRequest(searchParams);

            if (!clientValidation.valid || !clientValidation.client) {
                const errorRedirect = buildErrorRedirect(params.redirect_uri, {
                    error: clientValidation.error || "invalid_client",
                    error_description: clientValidation.error_description || "Invalid or inactive client",
                    state: params.state,
                });
                setError({
                    title: "Invalid Client",
                    message: clientValidation.error_description || "The client application is invalid or inactive",
                    redirect: errorRedirect,
                });
                setStep("error");
                return;
            }

            const client = clientValidation.client;

            if (!client.redirect_uris.includes(params.redirect_uri)) {
                setError({
                    title: "Invalid Redirect URI",
                    message: "The redirect URI is not registered for this application",
                    redirect: undefined,
                });
                setStep("error");
                return;
            }

            const requestedScopes = params.scope.split(" ").filter(Boolean);
            const invalidScopes = requestedScopes.filter((scope) => !client.allowed_scopes.includes(scope));

            if (invalidScopes.length > 0) {
                const errorRedirect = buildErrorRedirect(params.redirect_uri, {
                    error: "invalid_scope",
                    error_description: `Invalid scopes: ${invalidScopes.join(", ")}`,
                    state: params.state,
                });
                setError({
                    title: "Invalid Scopes",
                    message: `The following scopes are not allowed: ${invalidScopes.join(", ")}`,
                    redirect: errorRedirect,
                });
                setStep("error");
                return;
            }

            setClientInfo({
                name: client.name,
                logo_url: client.logo_url,
            });
            setAuthParams({
                client_id: params.client_id,
                redirect_uri: params.redirect_uri,
                scope: params.scope,
                state: params.state,
                code_challenge: params.code_challenge,
                code_challenge_method: params.code_challenge_method,
            });
            setScopes(requestedScopes);

            const authStatus = await checkAuthStatus();

            if (authStatus.authenticated) {
                setStep("consent");
            } else {
                const oidcParams = new URLSearchParams();
                oidcParams.set("client_id", params.client_id);
                oidcParams.set("redirect_uri", params.redirect_uri);
                oidcParams.set("response_type", params.response_type);
                oidcParams.set("scope", params.scope);
                oidcParams.set("state", params.state);
                oidcParams.set("code_challenge", params.code_challenge);
                oidcParams.set("code_challenge_method", params.code_challenge_method);

                const encodedParams = encodeURIComponent(oidcParams.toString());
                router.push(`/login?oidc_params=${encodedParams}`);
            }
        } catch (err) {
            const apiError = err as ApiError;
            setError({
                title: "Error",
                message: apiError.error_description || apiError.error || "An unexpected error occurred",
            });
            setStep("error");
        }
    }, [searchParams, router]);

    useEffect(() => {
        initializeAuthorization();
    }, [initializeAuthorization]);

    const handleApprove = async () => {
        if (!authParams) return;

        setIsLoading(true);

        try {
            const response = await confirmAuthorization({
                client_id: authParams.client_id,
                redirect_uri: authParams.redirect_uri,
                response_type: "code",
                scope: authParams.scope,
                state: authParams.state,
                code_challenge: authParams.code_challenge,
                code_challenge_method: authParams.code_challenge_method,
            });

            if (response.success && response.redirect_uri) {
                window.location.href = response.redirect_uri;
            } else {
                const errorRedirect = buildErrorRedirect(authParams.redirect_uri, {
                    error: !response.success ? response.error : "server_error",
                    error_description:
                        (!response.success && response.error_description) || "Failed to generate authorization code",
                    state: authParams.state,
                });
                window.location.href = errorRedirect;
            }
        } catch (err) {
            const apiError = err as ApiError;
            const errorRedirect = buildErrorRedirect(authParams.redirect_uri, {
                error: apiError.error || "server_error",
                error_description: apiError.error_description || "An error occurred during authorization",
                state: authParams.state,
            });
            window.location.href = errorRedirect;
        } finally {
            setIsLoading(false);
        }
    };

    const handleDeny = () => {
        if (!authParams) return;

        const errorRedirect = buildErrorRedirect(authParams.redirect_uri, {
            error: "access_denied",
            error_description: "The user denied the authorization request",
            state: authParams.state,
        });
        window.location.href = errorRedirect;
    };

    if (step === "validating") {
        return (
            <AuthorizeLayout>
                <div className="flex items-center justify-center py-12">
                    <div className="text-white/60">Validating request...</div>
                </div>
            </AuthorizeLayout>
        );
    }

    if (step === "error") {
        return (
            <AuthorizeLayout>
                <div className="space-y-6">
                    <div className="space-y-2">
                        <h2 className="text-xl font-semibold text-white">{error?.title}</h2>
                        <p className="text-sm text-white/60">{error?.message}</p>
                    </div>
                    {error?.redirect && (
                        <button
                            onClick={() => {
                                window.location.href = error.redirect!;
                            }}
                            className="w-full text-center text-xs text-white/50 uppercase tracking-widest hover:text-white/80 transition-colors duration-200 hover:cursor-pointer"
                        >
                            Return to application
                        </button>
                    )}
                </div>
            </AuthorizeLayout>
        );
    }

    if (step === "consent" && clientInfo && authParams) {
        return (
            <AuthorizeLayout>
                <ConsentScreen
                    clientName={clientInfo.name}
                    clientLogoUrl={clientInfo.logo_url}
                    scopes={scopes}
                    onApprove={handleApprove}
                    onDeny={handleDeny}
                    isLoading={isLoading}
                />
            </AuthorizeLayout>
        );
    }

    return null;
}

/**
 * Render the authorization page wrapped in a Suspense boundary with a loading fallback.
 *
 * @returns The authorization page element, showing a centered loading fallback until content is ready.
 */
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
