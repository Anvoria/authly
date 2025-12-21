"use client";

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ReactQueryDevtools } from "@tanstack/react-query-devtools";
import { useState } from "react";

/**
 * Provides a TanStack Query client to descendant components and renders the React Query Devtools.
 *
 * The created QueryClient uses default query options: queries have a staleTime of 60,000 ms (1 minute)
 * and will retry failed queries once.
 *
 * @param children - React nodes to be rendered inside the provider
 * @returns A React element that wraps `children` with a QueryClientProvider and includes ReactQueryDevtools
 */
export default function QueryProvider({ children }: { children: React.ReactNode }) {
    const [queryClient] = useState(
        () =>
            new QueryClient({
                defaultOptions: {
                    queries: {
                        staleTime: 60 * 1000, // 1 minute
                        retry: 1,
                    },
                },
            }),
    );

    return (
        <QueryClientProvider client={queryClient}>
            {children}
            <ReactQueryDevtools initialIsOpen={false} />
        </QueryClientProvider>
    );
}
