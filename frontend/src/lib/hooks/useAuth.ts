import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { login, register, getMe, checkAuthStatus } from "@/authly/lib/api";
import { LoginRequest } from "@/authly/lib/schemas/auth/login";
import { RegisterRequest } from "@/authly/lib/schemas/auth/register";

export const authKeys = {
    all: ["auth"] as const,
    me: () => [...authKeys.all, "me"] as const,
    status: () => [...authKeys.all, "status"] as const,
};

export function useMe() {
    return useQuery({
        queryKey: authKeys.me(),
        queryFn: getMe,
        retry: false,
    });
}

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

export function useAuthStatus() {
    return useQuery({
        queryKey: authKeys.status(),
        queryFn: checkAuthStatus,
        retry: false,
    });
}
