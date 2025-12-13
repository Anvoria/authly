import GeneralClient from "./globals/api/GeneralClient";
import {
    registerRequestSchema,
    registerResponseSchema,
    type RegisterRequest,
    type RegisterResponse,
} from "./schemas/auth/register";
import { loginRequestSchema, loginResponseSchema, type LoginRequest, type LoginResponse } from "./schemas/auth/login";
import type { ApiError } from "./schemas/auth/login";

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
