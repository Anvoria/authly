import GeneralClient from "./globals/api/GeneralClient";
import {
    registerRequestSchema,
    registerResponseSchema,
    type RegisterRequest,
    type RegisterResponse,
} from "./schemas/auth/register";
import { loginRequestSchema, loginResponseSchema, type LoginRequest, type LoginResponse } from "./schemas/auth/login";
import { meResponseSchema, type MeResponse } from "./schemas/auth/me";
import {
    validateAuthorizationRequestResponseSchema,
    confirmAuthorizationRequestSchema,
    confirmAuthorizationResponseSchema,
    type ValidateAuthorizationRequestResponse,
    type ConfirmAuthorizationRequest,
    type ConfirmAuthorizationResponse,
} from "./schemas/oidc";
import type { ApiError } from "./schemas/auth/login";
import type { ReadonlyURLSearchParams } from "next/navigation";

export type { ApiError };

export async function login(data: LoginRequest): Promise<LoginResponse> {
    const validatedData = loginRequestSchema.parse(data);

    const response = await GeneralClient.post<{
        user: {
            id: string;
            username: string;
            first_name: string;
            last_name: string;
            email: string | null;
            is_active: boolean;
            created_at: string;
            updated_at: string;
        };
    }>("/auth/login", validatedData);

    if (!response.success) {
        if ("isRedirect" in response && response.isRedirect) {
            const errorResponse: LoginResponse = {
                success: false,
                error: "redirect_occurred",
            };
            return loginResponseSchema.parse(errorResponse);
        }
        if ("error" in response) {
            const errorResponse: LoginResponse = {
                success: false,
                error: response.error,
            };
            return loginResponseSchema.parse(errorResponse);
        }
        const errorResponse: LoginResponse = {
            success: false,
            error: "unknown_error",
        };
        return loginResponseSchema.parse(errorResponse);
    }

    const successResponse: LoginResponse = {
        success: true,
        data: response.data,
        message: response.message ?? "",
    };

    return loginResponseSchema.parse(successResponse);
}

export async function register(data: RegisterRequest): Promise<RegisterResponse> {
    const validatedData = registerRequestSchema.parse(data);

    const response = await GeneralClient.post<{ user: { id: string } }>("/auth/register", validatedData);

    if (!response.success) {
        if ("isRedirect" in response && response.isRedirect) {
            const errorResponse: RegisterResponse = {
                success: false,
                error: "redirect_occurred",
            };
            return registerResponseSchema.parse(errorResponse);
        }
        if ("error" in response) {
            const errorResponse: RegisterResponse = {
                success: false,
                error: response.error,
            };
            return registerResponseSchema.parse(errorResponse);
        }
        const errorResponse: RegisterResponse = {
            success: false,
            error: "unknown_error",
        };
        return registerResponseSchema.parse(errorResponse);
    }

    const successResponse: RegisterResponse = {
        success: true,
        data: response.data,
        message: response.message ?? "",
    };

    return registerResponseSchema.parse(successResponse);
}

export async function getMe(): Promise<MeResponse> {
    const response = await GeneralClient.get<{
        user: {
            id: string;
            username: string;
            first_name: string;
            last_name: string;
            email: string | null;
            is_active: boolean;
            created_at: string;
            updated_at: string;
        };
    }>("/auth/me");

    if (!response.success) {
        if ("isRedirect" in response && response.isRedirect) {
            const errorResponse: MeResponse = {
                success: false,
                error: "redirect_occurred",
            };
            return meResponseSchema.parse(errorResponse);
        }
        if ("error" in response) {
            const errorResponse: MeResponse = {
                success: false,
                error: response.error,
            };
            return meResponseSchema.parse(errorResponse);
        }
        const errorResponse: MeResponse = {
            success: false,
            error: "unknown_error",
        };
        return meResponseSchema.parse(errorResponse);
    }

    const successResponse: MeResponse = {
        success: true,
        data: response.data,
        message: response.message ?? "",
    };

    return meResponseSchema.parse(successResponse);
}

export async function validateAuthorizationRequest(
    searchParams: ReadonlyURLSearchParams,
): Promise<ValidateAuthorizationRequestResponse> {
    const queryString = searchParams.toString();
    const response = await GeneralClient.get<ValidateAuthorizationRequestResponse>(
        `/oauth/authorize/validate?${queryString}`,
    );

    if (!response.success) {
        return {
            valid: false,
            error: "server_error",
            error_description: response.error || "Failed to validate authorization request",
        };
    }

    return validateAuthorizationRequestResponseSchema.parse(response.data);
}

export async function checkAuthStatus(): Promise<{ authenticated: boolean; user_id?: string }> {
    try {
        const response = await getMe();
        if (response.success) {
            return {
                authenticated: true,
                user_id: response.data.user.id,
            };
        }
        return { authenticated: false };
    } catch {
        return { authenticated: false };
    }
}

export async function confirmAuthorization(
    request: ConfirmAuthorizationRequest,
): Promise<ConfirmAuthorizationResponse> {
    const validatedData = confirmAuthorizationRequestSchema.parse(request);

    const response = await GeneralClient.post<ConfirmAuthorizationResponse>("/oauth/authorize/confirm", validatedData);

    if (!response.success) {
        if ("isRedirect" in response && response.isRedirect) {
            return {
                success: false,
                error: "redirect_occurred",
                error_description: `Redirect to ${response.redirectUrl}`,
            };
        }
        if ("error" in response) {
            return {
                success: false,
                error: response.error,
                error_description: response.errorDescription,
            };
        }
        return {
            success: false,
            error: "unknown_error",
            error_description: "An unexpected error occurred",
        };
    }

    const backendResponse = (response.data || response.rawResponse.data) as ConfirmAuthorizationResponse;

    const validated = confirmAuthorizationResponseSchema.parse(backendResponse);

    if (validated.success && !validated.redirect_uri) {
        throw new Error("Backend did not return redirect_uri in response");
    }

    return validated;
}
