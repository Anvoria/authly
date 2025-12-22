"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";

export default function DashboardIndex() {
    const router = useRouter();

    useEffect(() => {
        router.replace("/dashboard/profile");
    }, [router]);

    return (
        <div className="min-h-screen bg-black flex items-center justify-center text-white font-mono text-sm tracking-widest">
            REDIRECTING...
        </div>
    );
}
