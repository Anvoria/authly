"use client";

import { useSearchParams, useRouter } from "next/navigation";
import { Suspense, useState, useEffect } from "react";
import AuthorizeLayout from "@/authly/components/authorize/AuthorizeLayout";
import Input from "@/authly/components/ui/Input";
import Button from "@/authly/components/ui/Button";
import { isApiError } from "@/authly/lib/api";
import { loginRequestSchema, type LoginRequest } from "@/authly/lib/schemas/auth/login";
import { generateCodeVerifier, generateCodeChallenge, redirectToAuthorize } from "@/authly/lib/oidc";
import LocalStorageTokenService from "@/authly/lib/globals/client/LocalStorageTokenService";
import { useLogin, useMe } from "@/authly/lib/hooks/useAuth";

type LoginFormData = {
    username: string;
    password: string;
};

/**
 * Render the login form and manage local form state, validation, authentication, and OIDC redirect handling.
 *
 * Displays username/password inputs, validates input against the login schema, performs sign-in via the login mutation,
 * shows field-specific and API errors, and reflects loading states. If the user is already authenticated, or signs in
 * successfully and an `oidc_params` query parameter is present, the component prepares PKCE parameters when missing,
 * persists the PKCE verifier, and redirects to the authorize endpoint; otherwise it redirects to the app root.
 *
 * @returns The rendered login page content as a React element
 */
function LoginPageContent() {
    const searchParams = useSearchParams();
    const router = useRouter();
    const [formData, setFormData] = useState<LoginFormData>({
        username: "",
        password: "",
    });
    const [errors, setErrors] = useState<Partial<Record<keyof LoginFormData, string>>>({});
    const [apiError, setApiError] = useState<string | null>(null);

    const { data: meResponse, isLoading: isCheckingAuth } = useMe();
    const loginMutation = useLogin();

    useEffect(() => {
        if (meResponse?.success) {
            const oidcParams = searchParams.get("oidc_params");
            if (oidcParams) {
                const handleOidcRedirect = async () => {
                    try {
                        const decoded = decodeURIComponent(oidcParams);
                        const params = new URLSearchParams(decoded);

                        if (!params.has("code_challenge")) {
                            const verifier = generateCodeVerifier();
                            const challenge = await generateCodeChallenge(verifier);

                            params.set("code_challenge", challenge);
                            params.set("code_challenge_method", "S256");

                            LocalStorageTokenService.setOidcCodeVerifier(verifier);
                        }

                        const authorizeUrl = new URL("/authorize", window.location.origin);
                        params.forEach((value, key) => {
                            authorizeUrl.searchParams.set(key, value);
                        });
                        router.push(authorizeUrl.toString());
                    } catch {
                        router.push("/authorize?" + oidcParams);
                    }
                };
                handleOidcRedirect();
            } else {
                router.push("/");
            }
        }
    }, [meResponse, router, searchParams]);

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        setErrors({});
        setApiError(null);

        const validation = loginRequestSchema.safeParse(formData);
        if (!validation.success) {
            const fieldErrors: Partial<Record<keyof LoginFormData, string>> = {};
            validation.error.issues.forEach((issue) => {
                const field = issue.path[0] as keyof LoginFormData;
                if (field) {
                    fieldErrors[field] = issue.message;
                }
            });
            setErrors(fieldErrors);
            return;
        }

        const requestData: LoginRequest = {
            username: formData.username,
            password: formData.password,
        };

        loginMutation.mutate(requestData, {
            onSuccess: (response) => {
                if (response.success) {
                    const oidcParams = searchParams.get("oidc_params");
                    if (oidcParams) {
                        redirectToAuthorize(oidcParams);
                    } else {
                        router.push("/");
                    }
                } else {
                    setApiError(response.error || "Login failed");
                }
            },
            onError: (err) => {
                if (isApiError(err)) {
                    setApiError(err.error_description || err.error);
                } else if (err instanceof Error) {
                    setApiError(err.message);
                } else {
                    setApiError("An error occurred");
                }
            },
        });
    };

    const updateField = (field: keyof LoginFormData, value: string) => {
        setFormData((prev) => ({ ...prev, [field]: value }));
        if (errors[field]) {
            setErrors((prev) => {
                const newErrors = { ...prev };
                delete newErrors[field];
                return newErrors;
            });
        }
    };

    const isLoading = loginMutation.isPending;

    if (isCheckingAuth) {
        return (
            <AuthorizeLayout>
                <div className="flex items-center justify-center py-12">
                    <div className="text-white/60">Loading...</div>
                </div>
            </AuthorizeLayout>
        );
    }

    return (
        <AuthorizeLayout>
            <div className="space-y-6">
                <div className="space-y-1">
                    <h2 className="text-xl font-semibold text-white">Sign in</h2>
                    <p className="text-sm text-white/60">Enter your credentials to continue</p>
                </div>

                <form onSubmit={handleSubmit} className="space-y-5">
                    <Input
                        label="Username"
                        type="text"
                        placeholder="username"
                        value={formData.username}
                        onChange={(e) => updateField("username", e.target.value)}
                        required
                        disabled={isLoading}
                        error={errors.username}
                    />

                    <Input
                        label="Password"
                        type="password"
                        placeholder="••••••••"
                        value={formData.password}
                        onChange={(e) => updateField("password", e.target.value)}
                        required
                        disabled={isLoading}
                        error={errors.password}
                    />

                    {apiError && <p className="text-xs font-medium text-red-500">{apiError}</p>}

                    <div className="pt-1">
                        <Button fullWidth variant="primary" type="submit" disabled={isLoading}>
                            {isLoading ? "Signing in..." : "Sign In"}
                        </Button>
                    </div>
                </form>

                <div className="pt-2 border-t border-white/5">
                    <p className="text-center text-sm text-white/50 mt-2">
                        Don&apos;t have an account?{" "}
                        <a
                            href={`/register${searchParams.get("oidc_params") ? `?oidc_params=${encodeURIComponent(searchParams.get("oidc_params")!)}` : ""}`}
                            className="text-white/80 hover:text-white font-medium underline underline-offset-4 transition-colors duration-200"
                        >
                            Sign Up
                        </a>
                    </p>
                </div>
            </div>
        </AuthorizeLayout>
    );
}

/**
 * Page wrapper that renders the login UI inside a Suspense boundary.
 *
 * @returns The login page UI wrapped in a Suspense boundary. While the login content is resolving, displays a full-screen loading fallback.
 */
export default function LoginPage() {
    return (
        <Suspense
            fallback={
                <div className="min-h-screen w-full flex items-center justify-center bg-black">
                    <div className="text-white/60">Loading...</div>
                </div>
            }
        >
            <LoginPageContent />
        </Suspense>
    );
}