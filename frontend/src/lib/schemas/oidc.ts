import { z } from "zod";

/**
 * Schema for client info
 */
export const clientInfoSchema = z.object({
    id: z.string(),
    name: z.string(),
    logo_url: z.url().optional(),
    redirect_uris: z.array(z.string().url()),
    allowed_scopes: z.array(z.string()),
    active: z.boolean(),
});

/**
 * Type inferred from clientInfoSchema
 */
export type ClientInfo = z.infer<typeof clientInfoSchema>;

/**
 * Schema for validate authorization request response
 */
export const validateAuthorizationRequestResponseSchema = z.object({
    valid: z.boolean(),
    client: clientInfoSchema.optional(),
    error: z.string().optional(),
    error_description: z.string().optional(),
});

/**
 * Type inferred from validateAuthorizationRequestResponseSchema
 */
export type ValidateAuthorizationRequestResponse = z.infer<typeof validateAuthorizationRequestResponseSchema>;

/**
 * Schema for token request (OAuth2 token endpoint)
 */
export const tokenRequestSchema = z.object({
    grant_type: z.string(),
    code: z.string(),
    redirect_uri: z.string(),
    client_id: z.string(),
    client_secret: z.string().optional(),
    code_verifier: z.string().optional(),
});

/**
 * Type inferred from tokenRequestSchema
 */
export type TokenRequest = z.infer<typeof tokenRequestSchema>;

/**
 * Schema for token response (success)
 */
export const tokenSuccessResponseSchema = z.object({
    access_token: z.string(),
    token_type: z.string(),
    expires_in: z.number().optional(),
    refresh_token: z.string().optional(),
    id_token: z.string().optional(),
    scope: z.string().optional(),
});

/**
 * Schema for token response (error - OIDC format)
 */
export const tokenErrorResponseSchema = z.object({
    error: z.string(),
    error_description: z.string().optional(),
});

/**
 * Schema for token response (union)
 */
export const tokenResponseSchema = z.union([tokenSuccessResponseSchema, tokenErrorResponseSchema]);

/**
 * Type inferred from tokenResponseSchema
 */
export type TokenResponse = z.infer<typeof tokenResponseSchema>;

/**
 * Schema for confirm authorization request (for approve button)
 */
export const confirmAuthorizationRequestSchema = z.object({
    client_id: z.string(),
    redirect_uri: z.string(),
    response_type: z.string(),
    scope: z.string(),
    state: z.string(),
    code_challenge: z.string().optional(),
    code_challenge_method: z.string().optional(),
});

/**
 * Type inferred from confirmAuthorizationRequestSchema
 */
export type ConfirmAuthorizationRequest = z.infer<typeof confirmAuthorizationRequestSchema>;

/**
 * Schema for confirm authorization response
 */
export const confirmAuthorizationSuccessResponseSchema = z.object({
    success: z.literal(true),
    redirect_uri: z.string(),
});

/**
 * Schema for confirm authorization response (error)
 */
export const confirmAuthorizationErrorResponseSchema = z.object({
    success: z.literal(false),
    error: z.string(),
    error_description: z.string().optional(),
});

/**
 * Schema for confirm authorization response
 */
export const confirmAuthorizationResponseSchema = z.union([
    confirmAuthorizationSuccessResponseSchema,
    confirmAuthorizationErrorResponseSchema,
]);

/**
 * Type inferred from confirmAuthorizationResponseSchema
 */
export type ConfirmAuthorizationResponse = z.infer<typeof confirmAuthorizationResponseSchema>;
