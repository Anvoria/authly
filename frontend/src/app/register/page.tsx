"use client";

import { useSearchParams, useRouter } from "next/navigation";
import { Suspense, useState, useEffect, useRef } from "react";
import AuthorizeLayout from "@/authly/components/authorize/AuthorizeLayout";
import Input from "@/authly/components/ui/Input";
import Button from "@/authly/components/ui/Button";
import { isApiError } from "@/authly/lib/api";
import { registerFormSchema, registerRequestSchema, type RegisterFormData } from "@/authly/lib/schemas/auth/register";
import { generateCodeVerifier, generateCodeChallenge } from "@/authly/lib/oidc";
import LocalStorageTokenService from "@/authly/lib/globals/client/LocalStorageTokenService";
import { useRegister, useMe } from "@/authly/lib/hooks/useAuth";
import { CheckCircle } from "lucide-react";

/**
 * Renders the registration page UI, manages form state and validation, submits registration requests, and redirects on success or when already authenticated.
 *
 * The component validates input using the registration schema, surfaces per-field and API errors, disables inputs while the registration mutation is pending, and preserves `oidc_params` through redirect flows when present.
 *
 * @returns The registration page content as a React element.
 */
function RegisterPageContent() {
    const searchParams = useSearchParams();
    const router = useRouter();
    const [formData, setFormData] = useState<RegisterFormData>({
        first_name: "",
        last_name: "",
        email: "",
        username: "",
        password: "",
        confirmPassword: "",
    });
    const [errors, setErrors] = useState<Partial<Record<keyof RegisterFormData, string>>>({});
    const [apiError, setApiError] = useState<string | null>(null);
    const [successMessage, setSuccessMessage] = useState<string | null>(null);
    const isRedirectingRef = useRef(false);

    const { data: meResponse, isLoading: isCheckingAuth } = useMe();
    const registerMutation = useRegister();

    useEffect(() => {
        if (meResponse?.success && !successMessage) {
            if (isRedirectingRef.current) return;

            const oidcParams = searchParams.get("oidc_params");
            if (oidcParams) {
                isRedirectingRef.current = true;
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
                    } catch (error) {
                        console.error("Failed to process OIDC parameters:", error);
                        router.push("/authorize?" + oidcParams);
                    }
                };
                handleOidcRedirect().catch((error) => {
                    console.error("OIDC redirect failed:", error);
                    isRedirectingRef.current = false;
                });
            } else {
                router.push("/");
            }
        }
    }, [meResponse, router, searchParams, successMessage]);

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        setErrors({});
        setApiError(null);

        const validation = registerFormSchema.safeParse(formData);
        if (!validation.success) {
            const fieldErrors: Partial<Record<keyof RegisterFormData, string>> = {};
            validation.error.issues.forEach((issue) => {
                const field = issue.path[0] as keyof RegisterFormData;
                if (field) {
                    fieldErrors[field] = issue.message;
                }
            });
            setErrors(fieldErrors);
            return;
        }

        const requestValidation = registerRequestSchema.safeParse({
            first_name: formData.first_name || undefined,
            last_name: formData.last_name || undefined,
            email: formData.email || undefined,
            username: formData.username,
            password: formData.password,
        });

        if (!requestValidation.success) {
            setApiError("Invalid request data");
            return;
        }

        const requestData = requestValidation.data;

        registerMutation.mutate(requestData, {
            onSuccess: (response) => {
                if (!response.success) {
                    setApiError(response.error || "Registration failed");
                } else {
                    setSuccessMessage("Account created successfully! Redirecting to login...");
                    setTimeout(() => {
                        const params = searchParams.toString();
                        router.push("/login" + (params ? `?${params}` : ""));
                    }, 2000);
                }
            },
            onError: (err) => {
                let errorMessage = "An error occurred";
                if (isApiError(err)) {
                    errorMessage = err.error_description || err.error;
                } else if (err instanceof Error) {
                    errorMessage = err.message;
                }
                setApiError(errorMessage);
            },
        });
    };

    const updateField = (field: keyof RegisterFormData, value: string) => {
        setFormData((prev) => ({ ...prev, [field]: value }));
        if (errors[field]) {
            setErrors((prev) => {
                const newErrors = { ...prev };
                delete newErrors[field];
                return newErrors;
            });
        }
    };

    const isLoading = registerMutation.isPending;

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
                    <h2 className="text-xl font-semibold text-white">Create an account</h2>
                    <p className="text-sm text-white/60">Sign up to get started</p>
                </div>

                {successMessage ? (
                    <div className="flex flex-col items-center justify-center py-12 space-y-6 text-center">
                        <div className="w-16 h-16 rounded-full bg-green-500/10 flex items-center justify-center border border-green-500/20">
                            <CheckCircle className="w-8 h-8 text-green-500" />
                        </div>
                        <div className="space-y-2">
                            <p className="text-white font-medium">Account created successfully!</p>
                            <p className="text-sm text-white/40">Redirecting you to the login page...</p>
                        </div>
                    </div>
                ) : (
                    <form onSubmit={handleSubmit} className="space-y-5">
                        <div className="grid grid-cols-2 gap-4">
                            <Input
                                label="First Name"
                                type="text"
                                placeholder="John"
                                value={formData.first_name}
                                onChange={(e) => updateField("first_name", e.target.value)}
                                disabled={isLoading}
                                error={errors.first_name}
                            />

                            <Input
                                label="Last Name"
                                type="text"
                                placeholder="Doe"
                                value={formData.last_name}
                                onChange={(e) => updateField("last_name", e.target.value)}
                                disabled={isLoading}
                                error={errors.last_name}
                            />
                        </div>

                        <Input
                            label="Email"
                            type="email"
                            placeholder="name@example.com"
                            value={formData.email}
                            onChange={(e) => updateField("email", e.target.value)}
                            disabled={isLoading}
                            error={errors.email}
                        />

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

                        <Input
                            label="Confirm Password"
                            type="password"
                            placeholder="••••••••"
                            value={formData.confirmPassword}
                            onChange={(e) => updateField("confirmPassword", e.target.value)}
                            required
                            disabled={isLoading}
                            error={errors.confirmPassword}
                        />

                        {apiError && <p className="text-xs font-medium text-red-500">{apiError}</p>}

                        <div className="pt-1">
                            <Button fullWidth variant="primary" type="submit" disabled={isLoading}>
                                {isLoading ? "Creating account..." : "Sign Up"}
                            </Button>
                        </div>
                    </form>
                )}

                <div className="pt-2 border-t border-white/5">
                    <p className="text-center text-sm text-white/50 mt-2">
                        Already have an account?{" "}
                        <a
                            href={`/login${searchParams.get("oidc_params") ? `?oidc_params=${encodeURIComponent(searchParams.get("oidc_params")!)}` : ""}`}
                            className="text-white/80 hover:text-white font-medium underline underline-offset-4 transition-colors duration-200"
                        >
                            Sign In
                        </a>
                    </p>
                </div>
            </div>
        </AuthorizeLayout>
    );
}

/**
 * Renders the registration page wrapped in a Suspense boundary that shows a fullscreen loading fallback while content loads.
 *
 * @returns The React element for the registration page.
 */
export default function RegisterPage() {
    return (
        <Suspense
            fallback={
                <div className="min-h-screen w-full flex items-center justify-center bg-black">
                    <div className="text-white/60">Loading...</div>
                </div>
            }
        >
            <RegisterPageContent />
        </Suspense>
    );
}
