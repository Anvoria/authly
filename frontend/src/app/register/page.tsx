"use client";

import { useSearchParams, useRouter } from "next/navigation";
import { Suspense, useState, useEffect, useCallback } from "react";
import AuthorizeLayout from "@/authly/components/authorize/AuthorizeLayout";
import Input from "@/authly/components/ui/Input";
import Button from "@/authly/components/ui/Button";
import { register, getMe, isApiError } from "@/authly/lib/api";
import { registerFormSchema, registerRequestSchema, type RegisterFormData } from "@/authly/lib/schemas/auth/register";
import { redirectToAuthorize } from "@/authly/lib/oidc";

/**
 * Renders the registration page UI and manages the full registration flow: checks current authentication and redirects if already authenticated, validates form input, submits registration requests, preserves and forwards `oidc_params` when present, and surfaces field and API errors.
 *
 * @returns The JSX element for the register page content.
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
    const [isLoading, setIsLoading] = useState(false);
    const [isCheckingAuth, setIsCheckingAuth] = useState(true);
    const [errors, setErrors] = useState<Partial<Record<keyof RegisterFormData, string>>>({});
    const [apiError, setApiError] = useState<string | null>(null);

    const checkAuthentication = useCallback(async () => {
        try {
            const response = await getMe();
            if (response.success) {
                router.push("/");
            }
        } catch {
        } finally {
            setIsCheckingAuth(false);
        }
    }, [router]);

    useEffect(() => {
        checkAuthentication();
    }, [checkAuthentication]);

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

        setIsLoading(true);

        try {
            const requestData = registerRequestSchema.parse({
                first_name: formData.first_name || undefined,
                last_name: formData.last_name || undefined,
                email: formData.email || undefined,
                username: formData.username,
                password: formData.password,
            });

            const response = await register(requestData);

            if (response.success) {
                const oidcParams = searchParams.get("oidc_params");

                if (oidcParams) {
                    redirectToAuthorize(oidcParams);
                    return;
                }

                router.push("/");
            } else {
                setApiError(response.error || "Registration failed");
            }
        } catch (err) {
            let errorMessage = "An error occurred";
            if (isApiError(err)) {
                errorMessage = err.error_description || err.error;
            } else if (err instanceof Error) {
                errorMessage = err.message;
            } else if (typeof err === "string") {
                errorMessage = err;
            }
            setApiError(errorMessage);
        } finally {
            setIsLoading(false);
        }
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
 * Renders the registration page content inside a Suspense boundary with a centered loading fallback.
 *
 * @returns A JSX element containing <RegisterPageContent /> wrapped in React.Suspense and a full-screen loading indicator.
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
