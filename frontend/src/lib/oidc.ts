import type { ReadonlyURLSearchParams } from "next/navigation";

export interface ValidationError {
    error: string;
    error_description: string;
}

export interface ValidatedParams {
    client_id: string;
    redirect_uri: string;
    response_type: string;
    scope: string;
    state: string;
    code_challenge: string;
    code_challenge_method: string;
}

export interface ValidationResult {
    valid: boolean;
    params?: ValidatedParams;
    error?: ValidationError;
}

/**
 * Validates OIDC authorization request parameters
 */
export function validateAuthorizationParams(searchParams: ReadonlyURLSearchParams): ValidationResult {
    const clientId = searchParams.get("client_id");
    const redirectUri = searchParams.get("redirect_uri");
    const responseType = searchParams.get("response_type");
    const scope = searchParams.get("scope");
    const state = searchParams.get("state");
    const codeChallenge = searchParams.get("code_challenge");
    const codeChallengeMethod = searchParams.get("code_challenge_method");

    if (!clientId) {
        return {
            valid: false,
            error: {
                error: "invalid_request",
                error_description: "Missing required parameter: client_id",
            },
        };
    }

    if (!redirectUri) {
        return {
            valid: false,
            error: {
                error: "invalid_request",
                error_description: "Missing required parameter: redirect_uri",
            },
        };
    }

    if (!responseType) {
        return {
            valid: false,
            error: {
                error: "invalid_request",
                error_description: "Missing required parameter: response_type",
            },
        };
    }

    if (responseType !== "code") {
        return {
            valid: false,
            error: {
                error: "unsupported_response_type",
                error_description: `Unsupported response_type: ${responseType}. Only 'code' is supported.`,
            },
        };
    }

    if (!scope) {
        return {
            valid: false,
            error: {
                error: "invalid_request",
                error_description: "Missing required parameter: scope",
            },
        };
    }

    if (!state) {
        return {
            valid: false,
            error: {
                error: "invalid_request",
                error_description: "Missing required parameter: state",
            },
        };
    }

    if (!codeChallenge) {
        return {
            valid: false,
            error: {
                error: "invalid_request",
                error_description: "Missing required parameter: code_challenge",
            },
        };
    }

    if (!codeChallengeMethod) {
        return {
            valid: false,
            error: {
                error: "invalid_request",
                error_description: "Missing required parameter: code_challenge_method",
            },
        };
    }

    if (codeChallengeMethod !== "s256" && codeChallengeMethod !== "plain") {
        return {
            valid: false,
            error: {
                error: "invalid_request",
                error_description: `Unsupported code_challenge_method: ${codeChallengeMethod}. Only 'S256' and 'plain' are supported.`,
            },
        };
    }

    try {
        new URL(redirectUri);
    } catch {
        return {
            valid: false,
            error: {
                error: "invalid_request",
                error_description: "Invalid redirect_uri format",
            },
        };
    }

    return {
        valid: true,
        params: {
            client_id: clientId,
            redirect_uri: redirectUri,
            response_type: responseType,
            scope,
            state,
            code_challenge: codeChallenge,
            code_challenge_method: codeChallengeMethod,
        },
    };
}

/**
 * Builds error redirect URL with OIDC error parameters
 */
export function buildErrorRedirect(redirectUri: string, error: ValidationError & { state?: string }): string {
    const url = new URL(redirectUri);
    url.searchParams.set("error", error.error);
    url.searchParams.set("error_description", error.error_description);
    if (error.state) {
        url.searchParams.set("state", error.state);
    }
    return url.toString();
}

/**
 * Builds success redirect URL with authorization code
 */
export function buildSuccessRedirect(redirectUri: string, code: string, state?: string): string {
    const url = new URL(redirectUri);
    url.searchParams.set("code", code);
    if (state) {
        url.searchParams.set("state", state);
    }
    return url.toString();
}
