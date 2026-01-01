"use client";

import { useMe } from "@/authly/lib/hooks/useAuth";
import Input from "@/authly/components/ui/Input";
import { useRouter } from "next/navigation";
import { useEffect } from "react";
import { ArrowLeft } from "lucide-react";
import Button from "@/authly/components/ui/Button";

/**
 * Render the profile page for the current user and enforce access control.
 *
 * Displays a read-only view of the user's identity and account information; if the user is not authenticated the component redirects to "/auth/login" and while user data is being fetched it renders a full-screen loading spinner.
 *
 * @returns The React element representing the profile page for the current user.
 */
export default function ProfilePage() {
    const { data: response, isLoading } = useMe();
    const router = useRouter();

    useEffect(() => {
        if (!isLoading && !response?.success) {
            router.push("/auth/login");
        }
    }, [isLoading, response, router]);

    const formatDate = (dateString?: string) => {
        if (!dateString) return "Unknown";
        return new Date(dateString).toLocaleDateString("en-US", {
            year: "numeric",
            month: "long",
            day: "numeric",
        });
    };

    if (isLoading) {
        return (
            <div className="min-h-screen bg-black flex items-center justify-center">
                <div className="flex flex-col items-center gap-4">
                    <div className="h-8 w-8 animate-spin rounded-full border-2 border-white/20 border-t-white"></div>
                </div>
            </div>
        );
    }

    if (!response?.success || !response?.data?.user) {
        return null;
    }

    const user = response.data.user;

    return (
        <div className="min-h-screen bg-black text-white font-sans selection:bg-white/10 p-8 lg:p-12">
            <header className="mb-12 flex items-center justify-between">
                <div>
                    <h1 className="text-3xl font-light tracking-tight text-white mb-2">My Profile</h1>
                    <p className="text-white/40 text-sm">Manage your personal information and account settings.</p>
                </div>
                <Button variant="ghost" size="sm" icon={<ArrowLeft className="w-4 h-4" />} href="/dashboard">
                    Back to Home
                </Button>
            </header>

            <main className="max-w-4xl">
                <section className="mb-12">
                    <h2 className="text-sm font-medium text-white/40 uppercase tracking-widest mb-6 border-b border-white/10 pb-2">
                        Identity
                    </h2>
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-8">
                        <div className="space-y-6">
                            <Input label="Username" value={user.username} disabled readOnly />
                            <Input label="Email Address" value={user.email || ""} disabled readOnly />
                            <Input
                                label="User ID"
                                value={user.id}
                                disabled
                                readOnly
                                className="font-mono text-xs opacity-75"
                            />
                        </div>
                        <div className="space-y-6">
                            <Input label="First Name" value={user.first_name || ""} disabled readOnly />
                            <Input label="Last Name" value={user.last_name || ""} disabled readOnly />
                        </div>
                    </div>
                </section>

                <section>
                    <h2 className="text-sm font-medium text-white/40 uppercase tracking-widest mb-6 border-b border-white/10 pb-2">
                        Account Information
                    </h2>
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-8">
                        <Input
                            label="Member Since"
                            value={formatDate(user.created_at)}
                            disabled
                            readOnly
                            helperText="The date when your account was originally created."
                        />
                    </div>
                </section>
            </main>
        </div>
    );
}
