import { useMutation, useQuery } from "@tanstack/react-query";
import { validateAuthorizationRequest, confirmAuthorization } from "@/authly/lib/api";
import { ConfirmAuthorizationRequest } from "@/authly/lib/schemas/oidc";
import { ReadonlyURLSearchParams } from "next/navigation";

export const oidcKeys = {
    all: ["oidc"] as const,
    validate: (params: string) => [...oidcKeys.all, "validate", params] as const,
};

export function useValidateAuthorization(searchParams: ReadonlyURLSearchParams) {
    return useQuery({
        queryKey: oidcKeys.validate(searchParams.toString()),
        queryFn: () => validateAuthorizationRequest(searchParams),
        enabled: !!searchParams.toString(),
        retry: false,
    });
}

export function useConfirmAuthorization() {
    return useMutation({
        mutationFn: (data: ConfirmAuthorizationRequest) => confirmAuthorization(data),
    });
}
