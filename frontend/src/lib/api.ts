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
import { OIDC_CONFIG } from "./config";

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
 * Fetch the current user's profile and permissions from the internal /auth/me endpoint.
 *
 * @returns A `MeResponse` containing user details and permissions.
 */
export async function getMe(): Promise<MeResponse> {
    const response = await GeneralClient.get<unknown>("/auth/me");

    if (!response.success) {
        return {
            success: false,
            error: response.error || "unknown_error",
        };
    }

    return meResponseSchema.parse({
        success: true,
        data: response.data,
        message: response.message || "User info fetched successfully",
    });
}

/**
 * Retrieve the current authenticated user's profile from the OIDC UserInfo endpoint.
 *
 * Maps standard OIDC userinfo claims (e.g., `sub`, `preferred_username`, `given_name`, `family_name`, `email`, `active`, `created_at`, `updated_at`) to the internal `MeResponse` shape.
 *
 * @returns On success, a `MeResponse` with `success: true` and `data.user` containing `id`, `username`, `first_name`, `last_name`, `email`, `is_active`, `created_at`, and `updated_at`. On failure, a `MeResponse` with `success: false` and an `error` code describing the failure.
 */
export async function getUserInfo(): Promise<MeResponse> {
    const response = await GeneralClient.get<{
        sub: string;
        name?: string;
        preferred_username?: string;
        email?: string;
        given_name?: string;
        family_name?: string;
        active?: boolean;
        created_at?: string;
        updated_at?: string;
    }>("/oauth/userinfo");

    if (!response.success) {
        return {
            success: false,
            error: response.error || "unknown_error",
        };
    }

    // Map OIDC claims to MeResponse format
    return {
        success: true,
        data: {
            user: {
                id: response.data.sub,
                username: response.data.preferred_username || "",
                first_name: response.data.given_name || "",
                last_name: response.data.family_name || "",
                email: response.data.email || null,
                is_active: response.data.active ?? true,
                created_at: response.data.created_at || "",
                updated_at: response.data.updated_at || "",
            },
        },
        message: "User info fetched successfully",
    };
}

/**
 * Determines whether the Identity Provider has an active session for the current client.
 *
 * Relies on HTTP-only cookies managed by the backend; no explicit tokens need to be provided.
 *
 * @returns `true` if the `/auth/me` endpoint indicates an active session, `false` otherwise.
 */
export async function checkIdPSession(): Promise<boolean> {
    try {
        const response = await GeneralClient.get("/auth/me");
        return response.success;
    } catch {
        return false;
    }
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
            error: response.error || "server_error",
            error_description: response.errorDescription ?? "Failed to validate authorization request",
        };
    }

    return validateAuthorizationRequestResponseSchema.parse(response.data);
}

/**
 * Check whether the current session is authenticated and include the user's id when authenticated.
 *
 * @returns `{ authenticated: true, user_id: string }` when the current session is authenticated, `{ authenticated: false }` otherwise.
 */
export async function checkAuthStatus(): Promise<{
    authenticated: boolean;
    user_id?: string;
}> {
    try {
        const response = await getUserInfo();
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
 * Confirm an OAuth authorization decision with the backend and return the validated authorization response.
 *
 * @param request - The authorization confirmation payload
 * @returns A `ConfirmAuthorizationResponse` object. If `success` is `false`, the object contains `error` and `error_description`. If `success` is `true`, the object contains the backend-provided authorization data, including `redirect_uri` when applicable.
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

/**
 * Exchanges an authorization code for OAuth 2.0 tokens (access token, refresh token, ID token).
 *
 * @param request - The token exchange request containing the authorization code, redirect URI,
 *                  code verifier (for PKCE), and client credentials
 * @returns A `TokenResponse` with the exchanged tokens on success, or an error response
 *          containing `error` and `error_description` on failure
 *
 * @note This function sends the request as `application/x-www-form-urlencoded` data as required
 *       by the OAuth 2.0 specification for token endpoint requests.
 */
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

/**
 * Refreshes the OAuth access token using the backend session cookie.
 *
 * Sends a form-encoded refresh request (grant_type=refresh_token) to the token endpoint and
 * relies on the backend's HTTP-only session cookie for authentication when a refresh token value
 * is not provided by the client.
 *
 * @returns A `TokenResponse` containing `access_token`, `refresh_token`, `token_type`, and `expires_in` on success; on failure an object with `error` and `error_description`.
 */
export async function refreshAccessToken(): Promise<TokenResponse> {
    const formData = new URLSearchParams();
    formData.append("grant_type", "refresh_token");
    formData.append("client_id", OIDC_CONFIG.client_id);

    const response = await GeneralClient.post<TokenResponse>("/oauth/token", formData.toString(), {
        headers: {
            "Content-Type": "application/x-www-form-urlencoded",
        },
    });

    if (!response.success) {
        return {
            error: response.error || "refresh_failed",
            error_description: response.errorDescription || "Failed to refresh token via cookie",
        };
    }

    return tokenResponseSchema.parse(response.data);
}
