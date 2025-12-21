import { useMutation, useQuery } from "@tanstack/react-query";
import { validateAuthorizationRequest, confirmAuthorization } from "@/authly/lib/api";
import { ConfirmAuthorizationRequest } from "@/authly/lib/schemas/oidc";
import { ReadonlyURLSearchParams } from "next/navigation";

export const oidcKeys = {
    all: ["oidc"] as const,
    validate: (params: string) => [...oidcKeys.all, "validate", params] as const,
};

/**
 * Hook that validates an OpenID Connect authorization request from URL search parameters.
 *
 * @param searchParams - The Next.js `ReadonlyURLSearchParams` containing the OIDC authorization request query string
 * @returns The React Query result for the validation request; on success contains the validated authorization data
 */
export function useValidateAuthorization(searchParams: ReadonlyURLSearchParams) {
    return useQuery({
        queryKey: oidcKeys.validate(searchParams.toString()),
        queryFn: () => validateAuthorizationRequest(searchParams),
        enabled: !!searchParams.toString(),
        retry: false,
    });
}

/**
 * Initiates a mutation to confirm an OIDC authorization request.
 *
 * @returns The React Query mutation object used to perform and track a confirm-authorization operation with a `ConfirmAuthorizationRequest` payload.
 */
export function useConfirmAuthorization() {
    return useMutation({
        mutationFn: (data: ConfirmAuthorizationRequest) => confirmAuthorization(data),
    });
}