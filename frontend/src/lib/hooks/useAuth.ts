import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { login, register, getMe, checkAuthStatus } from "@/authly/lib/api";
import { LoginRequest } from "@/authly/lib/schemas/auth/login";
import { RegisterRequest } from "@/authly/lib/schemas/auth/register";

export const authKeys = {
    all: ["auth"] as const,
    me: () => [...authKeys.all, "me"] as const,
    status: () => [...authKeys.all, "status"] as const,
};

/**
 * Fetches the current authenticated user's data and exposes the React Query result.
 *
 * @returns The React Query result for the `authKeys.me()` query — includes the fetched user data (`data`) and query state fields such as `isLoading`, `isError`, `error`, and `refetch`.
 */
export function useMe() {
    return useQuery({
        queryKey: authKeys.me(),
        queryFn: getMe,
        retry: false,
    });
}

/**
 * Provides a mutation hook to perform user login and refresh auth queries on success.
 *
 * @returns The mutation configured to call the login API with a `LoginRequest`. If the response has `success === true`, it invalidates `authKeys.me()` and `authKeys.status()` so related queries are refetched.
 */
export function useLogin() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (data: LoginRequest) => login(data),
        onSuccess: (response) => {
            if (response.success) {
                queryClient.invalidateQueries({ queryKey: authKeys.me() });
                queryClient.invalidateQueries({ queryKey: authKeys.status() });
            }
        },
    });
}

/**
 * Creates a mutation hook to register a new user and invalidate auth-related queries on success.
 *
 * @returns The React Query mutation result for registering a user. When the registration response has `success` equal to `true`, it invalidates the auth `me` and `status` query keys to trigger refetch.
 */
export function useRegister() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (data: RegisterRequest) => register(data),
        onSuccess: (response) => {
            if (response.success) {
                queryClient.invalidateQueries({ queryKey: authKeys.me() });
                queryClient.invalidateQueries({ queryKey: authKeys.status() });
            }
        },
    });
}

/**
 * Fetches the current authentication status.
 *
 * @returns The React Query result for the auth status request; `data` contains the status response when available.
 */
export function useAuthStatus() {
    return useQuery({
        queryKey: authKeys.status(),
        queryFn: checkAuthStatus,
        retry: false,
    });
}
