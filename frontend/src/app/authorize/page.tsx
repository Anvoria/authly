"use client";

import { useSearchParams, useRouter } from "next/navigation";
import { Suspense, useEffect, useMemo } from "react";
import AuthorizeLayout from "@/authly/components/authorize/AuthorizeLayout";
import ConsentScreen from "@/authly/components/authorize/ConsentScreen";
import { validateAuthorizationParams, buildErrorRedirect } from "@/authly/lib/oidc";
import { isApiError } from "@/authly/lib/api";
import { useAuthStatus } from "@/authly/lib/hooks/useAuth";
import { useValidateAuthorization, useConfirmAuthorization } from "@/authly/lib/hooks/useOidc";

interface ErrorState {
    title: string;
    message: string;
    redirect?: string;
}

type AuthPageState =
    | { type: "validating" }
    | { type: "error"; error: ErrorState }
    | { type: "consent"; client: { name: string; logo_url?: string }; scopes: string[] }
    | { type: "redirecting" };

/**
 * Render the authorize page content and drive validation, consent, and redirect flows for an OIDC authorization request.
 *
 * Reads OIDC query parameters, validates the client and requested scopes, checks authentication status, and renders
 * one of: a validating/redirecting message, an error view (optionally with a return-to-app redirect), or a consent
 * screen. Also handles approve and deny actions producing the appropriate redirects.
 *
 * @returns The JSX for the authorization page or `null` when nothing should be rendered
 */
function AuthorizePageContent() {
    const searchParams = useSearchParams();
    const router = useRouter();

    const { data: authStatus, isLoading: isCheckingAuth } = useAuthStatus();
    const {
        data: clientValidation,
        isLoading: isValidatingClient,
        error: validationError,
    } = useValidateAuthorization(searchParams);
    const confirmMutation = useConfirmAuthorization();

    const validationParams = useMemo(() => validateAuthorizationParams(searchParams), [searchParams]);

    const state: AuthPageState = useMemo(() => {
        if (!validationParams.valid) {
            return {
                type: "error",
                error: {
                    title: "Invalid Request",
                    message: validationParams.error!.error_description,
                },
            };
        }

        if (validationError) {
            return {
                type: "error",
                error: {
                    title: "Error",
                    message: validationError instanceof Error ? validationError.message : "Validation failed",
                },
            };
        }

        if (isValidatingClient || isCheckingAuth) {
            return { type: "validating" };
        }

        if (clientValidation) {
            if (!clientValidation.valid || !clientValidation.client) {
                return {
                    type: "error",
                    error: {
                        title: "Invalid Client",
                        message: clientValidation.error_description || "The client application is invalid or inactive",
                    },
                };
            }

            const params = validationParams.params!;
            const client = clientValidation.client;

            if (!client.redirect_uris.includes(params.redirect_uri)) {
                return {
                    type: "error",
                    error: {
                        title: "Invalid Redirect URI",
                        message: "The redirect URI is not registered for this application",
                    },
                };
            }

            const requestedScopes = params.scope.split(" ").filter(Boolean);
            const invalidScopes = requestedScopes.filter((scope) => !client.allowed_scopes.includes(scope));

            if (invalidScopes.length > 0) {
                const errorRedirect = buildErrorRedirect(params.redirect_uri, {
                    error: "invalid_scope",
                    error_description: `Invalid scopes: ${invalidScopes.join(", ")}`,
                    state: params.state,
                });
                return {
                    type: "error",
                    error: {
                        title: "Invalid Scopes",
                        message: `The following scopes are not allowed: ${invalidScopes.join(", ")}`,
                        redirect: errorRedirect,
                    },
                };
            }

            if (authStatus?.authenticated) {
                return {
                    type: "consent",
                    client: {
                        name: client.name,
                        logo_url: client.logo_url,
                    },
                    scopes: requestedScopes,
                };
            }

            return { type: "redirecting" };
        }

        return { type: "validating" };
    }, [validationParams, clientValidation, validationError, authStatus, isValidatingClient, isCheckingAuth]);

    // Handle redirect to login in an effect
    useEffect(() => {
        if (state.type === "redirecting" && validationParams.params) {
            const params = validationParams.params;
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
    }, [state.type, validationParams.params, router]);

    const handleApprove = async () => {
        if (!validationParams.params) return;

        const params = validationParams.params;

        confirmMutation.mutate(
            {
                client_id: params.client_id,
                redirect_uri: params.redirect_uri,
                response_type: "code",
                scope: params.scope,
                state: params.state,
                code_challenge: params.code_challenge,
                code_challenge_method: params.code_challenge_method,
            },
            {
                onSuccess: (response) => {
                    if (response.success && response.redirect_uri) {
                        window.location.href = response.redirect_uri;
                    } else {
                        const errorRedirect = buildErrorRedirect(params.redirect_uri, {
                            error: !response.success ? response.error : "server_error",
                            error_description:
                                (!response.success && response.error_description) ||
                                "Failed to generate authorization code",
                            state: params.state,
                        });
                        window.location.href = errorRedirect;
                    }
                },
                onError: (err) => {
                    let error = "server_error";
                    let errorDescription = "An error occurred during authorization";

                    if (isApiError(err)) {
                        error = err.error;
                        errorDescription = err.error_description || errorDescription;
                    } else if (err instanceof Error) {
                        errorDescription = err.message;
                    }

                    const errorRedirect = buildErrorRedirect(params.redirect_uri, {
                        error,
                        error_description: errorDescription,
                        state: params.state,
                    });
                    window.location.href = errorRedirect;
                },
            },
        );
    };

    const handleDeny = () => {
        if (!validationParams.params) return;
        const params = validationParams.params;

        const errorRedirect = buildErrorRedirect(params.redirect_uri, {
            error: "access_denied",
            error_description: "The user denied the authorization request",
            state: params.state,
        });
        window.location.href = errorRedirect;
    };

    if (state.type === "validating" || state.type === "redirecting") {
        return (
            <AuthorizeLayout>
                <div className="flex items-center justify-center py-12">
                    <div className="text-white/60">
                        {state.type === "validating" ? "Validating request..." : "Redirecting to login..."}
                    </div>
                </div>
            </AuthorizeLayout>
        );
    }

    if (state.type === "error") {
        return (
            <AuthorizeLayout>
                <div className="space-y-6">
                    <div className="space-y-2">
                        <h2 className="text-xl font-semibold text-white">{state.error.title}</h2>
                        <p className="text-sm text-white/60">{state.error.message}</p>
                    </div>
                    {state.error.redirect && (
                        <button
                            onClick={() => {
                                window.location.href = state.error.redirect!;
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

    if (state.type === "consent") {
        return (
            <AuthorizeLayout>
                <ConsentScreen
                    clientName={state.client.name}
                    clientLogoUrl={state.client.logo_url}
                    scopes={state.scopes}
                    onApprove={handleApprove}
                    onDeny={handleDeny}
                    isLoading={confirmMutation.isPending}
                />
            </AuthorizeLayout>
        );
    }

    return null;
}

/**
 * Page component that renders the authorization UI and provides a loading fallback.
 *
 * @returns The React element for the authorize page wrapped in a Suspense boundary with a centered loading fallback.
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