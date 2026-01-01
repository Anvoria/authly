import { z } from "zod";

/**
 * Schema for /auth/me response
 */
export const meSuccessResponseSchema = z.object({
    success: z.literal(true),
    data: z.object({
        user: z
            .object({
                id: z.string(),
                username: z.string(),
                first_name: z.string(),
                last_name: z.string(),
                email: z.string().nullable(),
                is_active: z.boolean(),
                created_at: z.string(),
                updated_at: z.string(),
            })
            .loose(),
        permissions: z.record(z.string(), z.number().int().min(0).max(Number.MAX_SAFE_INTEGER)).optional(),
    }),
    message: z.string(),
});

/**
 * Schema for /auth/me response error
 */
export const meErrorResponseSchema = z.object({
    success: z.literal(false),
    error: z.string(),
});

/**
 * Schema for /auth/me response
 */
export const meResponseSchema = z.union([meSuccessResponseSchema, meErrorResponseSchema]);

/**
 * Type inferred from /auth/me response
 */
export type MeResponse = z.infer<typeof meResponseSchema>;
