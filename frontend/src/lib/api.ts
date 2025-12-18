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
    type TokenRequest,
    type TokenResponse,
    tokenRequestSchema,
    tokenResponseSchema,
} from "./schemas/oidc";
import type { ApiError } from "./schemas/auth/login";
import type { ReadonlyURLSearchParams } from "next/navigation";
import { z } from "zod";
import { IRequestResponsePayload } from "./globals/api/interfaces/IRequestResponsePayload";

export type { ApiError };

/**
 * Type guard to check if an error is an ApiError.
 *
 * @param err - The error to check
 * @returns `true` if `err` matches the `ApiError` structure, `false` otherwise
 */
export function isApiError(err: unknown): err is ApiError {
    return typeof err === "object" && err !== null && ("error" in err || "error_description" in err);
}

async function handleAuthRequest<T, S>(
    request: () => Promise<IRequestResponsePayload<T, unknown>>,
    schema: z.ZodType<S>,
): Promise<S> {
    const response = await request();

    if (!response.success) {
        if ("isRedirect" in response && response.isRedirect) {
            return schema.parse({
                success: false,
                error: "redirect_occurred",
            });
        }
        if ("error" in response) {
            return schema.parse({
                success: false,
                error: response.error,
            });
        }
        return schema.parse({
            success: false,
            error: "unknown_error",
        });
    }

    return schema.parse({
        success: true,
        data: response.data,
        message: response.message ?? "",
    });
}

/**
 * Authenticate a user with the provided login credentials and return a validated login response.
 *
 * @param data - The login credentials to submit (e.g., username/email and password)
 * @returns A `LoginResponse` describing the operation result. On success it contains `data` with the authenticated user's information and an optional `message`. On failure it contains an `error` code such as `"redirect_occurred"`, a server-provided error, or `"unknown_error"`.
 */
export async function login(data: LoginRequest): Promise<LoginResponse> {
    const validatedData = loginRequestSchema.parse(data);
    return handleAuthRequest(
        () =>
            GeneralClient.post<{
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
            }>("/auth/login", validatedData),
        loginResponseSchema,
    );
}

/**
 * Create a new user account with the provided registration data.
 *
 * @param data - Registration details to validate and send to the backend
 * @returns A `RegisterResponse`: `success: true` includes `data` and optional `message`; `success: false` includes an `error` code (`redirect_occurred`, a server-provided error, or `unknown_error`)
 */
export async function register(data: RegisterRequest): Promise<RegisterResponse> {
    const validatedData = registerRequestSchema.parse(data);
    return handleAuthRequest(
        () => GeneralClient.post<{ user: { id: string } }>("/auth/register", validatedData),
        registerResponseSchema,
    );
}

/**
 * Fetches the current authenticated user's profile from the backend and returns a validated response.
 *
 * @returns A `MeResponse` containing the user's profile under `data` when successful; otherwise a `MeResponse` with `success: false` and an `error` string describing the failure—either `redirect_occurred`, the backend-provided error, or `unknown_error`.
 */
export async function getMe(): Promise<MeResponse> {
    return handleAuthRequest(
        () =>
            GeneralClient.get<{
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
            }>("/auth/me"),
        meResponseSchema,
    );
}

/**
 * Validates an OAuth2/OIDC authorization request represented by URL search parameters.
 *
 * @param searchParams - The query parameters from the authorization request (e.g., from window.location.search)
 * @returns The parsed validation result indicating whether the request is valid and, on failure, an error and description
 */
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

/**
 * Determine whether a user is currently authenticated and, if so, provide their user ID.
 *
 * @returns `{ authenticated: true, user_id: string }` when authenticated; `{ authenticated: false }` otherwise.
 */
export async function checkAuthStatus(): Promise<{
    authenticated: boolean;
    user_id?: string;
}> {
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

/**
 * Confirms an OAuth authorization decision with the backend and returns the validated authorization response.
 *
 * @param request - The authorization confirmation payload (validated against `confirmAuthorizationRequestSchema`)
 * @returns A `ConfirmAuthorizationResponse` parsed and validated by `confirmAuthorizationResponseSchema`. On failure the response contains `success: false` and `error`/`error_description`; on success it contains the backend-provided authorization data (including `redirect_uri` when applicable).
 * @throws Error if the backend indicates success but does not provide a `redirect_uri`
 */
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

export async function exchangeToken(request: TokenRequest): Promise<TokenResponse> {
    const validatedData = tokenRequestSchema.parse(request);

    const formData = new URLSearchParams();
    for (const key in validatedData) {
        const value = validatedData[key as keyof typeof validatedData];
        if (value !== undefined && value !== null) {
            formData.append(key, String(value));
        }
    }

    const response = await GeneralClient.post<TokenResponse>("/oauth/token", formData.toString(), {
        headers: {
            "Content-Type": "application/x-www-form-urlencoded",
        },
    });

    if (!response.success) {
        if ("isRedirect" in response && response.isRedirect) {
            return {
                error: "server_error",
                error_description: "Unexpected redirect from token endpoint",
            };
        }

        if ("error" in response) {
            return {
                error: response.error,
                error_description: response.errorDescription,
            };
        }

        return {
            error: "unknown_error",
            error_description: "An unexpected error occurred",
        };
    }

    const result = tokenResponseSchema.parse(response.data);
    return result;
}
