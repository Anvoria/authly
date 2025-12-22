"use client";

import React, { useEffect } from "react";
import Image from "next/image";

export interface AuthorizeLayoutProps {
    children: React.ReactNode;
}

/**
 * Layout wrapper for authorization screens that renders a centered, themed card with a header and body area.
 *
 * @param children - Content to render inside the layout's body area.
 * @returns The layout element containing the header (logo and title) and the provided children
 */
export default function AuthorizeLayout({ children }: AuthorizeLayoutProps) {
    useEffect(() => {
        document.body.classList.add("overflow-hidden");
        return () => {
            document.body.classList.remove("overflow-hidden");
        };
    }, []);

    return (
        <div className="min-h-screen w-full flex items-center justify-center bg-black p-4">
            <div className="w-full max-w-md relative">
                <div className="relative bg-black/80 backdrop-blur-xl border border-white/10 shadow-[0_8px_32px_rgba(0,0,0,0.5)] overflow-hidden">
                    <div className="absolute inset-0 bg-linear-to-br from-white/2 to-transparent pointer-events-none" />

                    <div className="relative z-10">
                        <div className="flex items-center gap-3 px-8 pt-8 pb-6 border-b border-white/10">
                            <div className="relative w-10 h-10">
                                <Image src="/images/logo.svg" alt="Authly Logo" fill className="object-contain" />
                            </div>
                            <span className="text-2xl font-bold tracking-tight text-white">Authly</span>
                        </div>

                        <div className="px-8 py-8">{children}</div>
                    </div>
                </div>
            </div>
        </div>
    );
}
