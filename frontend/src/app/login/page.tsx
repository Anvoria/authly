"use client";

import { useSearchParams, useRouter } from "next/navigation";
import { Suspense, useState } from "react";
import AuthorizeLayout from "@/authly/components/authorize/AuthorizeLayout";
import Input from "@/authly/components/ui/Input";
import Button from "@/authly/components/ui/Button";
import { login, type ApiError } from "@/authly/lib/api";
import { loginRequestSchema, type LoginRequest } from "@/authly/lib/schemas/auth/login";

type LoginFormData = {
    username: string;
    password: string;
};

function LoginPageContent() {
    const searchParams = useSearchParams();
    const router = useRouter();
    const [formData, setFormData] = useState<LoginFormData>({
        username: "",
        password: "",
    });
    const [isLoading, setIsLoading] = useState(false);
    const [errors, setErrors] = useState<Partial<Record<keyof LoginFormData, string>>>({});
    const [apiError, setApiError] = useState<string | null>(null);

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

        setIsLoading(true);

        try {
            const requestData: LoginRequest = {
                username: formData.username,
                password: formData.password,
            };

            const response = await login(requestData);

            if (response.success) {
                const oidcParams = searchParams.get("oidc_params");

                if (oidcParams) {
                    try {
                        const decoded = decodeURIComponent(oidcParams);
                        const params = new URLSearchParams(decoded);
                        const authorizeUrl = new URL("/authorize", window.location.origin);
                        params.forEach((value, key) => {
                            authorizeUrl.searchParams.set(key, value);
                        });
                        window.location.href = authorizeUrl.toString();
                        return;
                    } catch {
                        router.push("/authorize?" + oidcParams);
                        return;
                    }
                }

                router.push("/");
            } else {
                setApiError(response.error || "Login failed");
            }
        } catch (err) {
            const apiError = err as ApiError;
            setApiError(apiError.error_description || apiError.error || "An error occurred");
        } finally {
            setIsLoading(false);
        }
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
