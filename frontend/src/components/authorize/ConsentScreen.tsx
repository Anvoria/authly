"use client";

import Button from "@/authly/components/ui/Button";

export interface ConsentScreenProps {
    clientName: string;
    clientLogoUrl?: string;
    scopes: string[];
    onApprove: () => void;
    onDeny: () => void;
    isLoading?: boolean;
}

/**
 * Render an OAuth consent screen prompting the user to approve or deny an application's requested scopes.
 *
 * @param clientName - Application display name shown in the title and description
 * @param clientLogoUrl - Optional URL of the application's logo; rendered when provided
 * @param scopes - Array of requested scope strings; common scopes are shown with human-readable descriptions
 * @param onApprove - Callback invoked when the user approves the request
 * @param onDeny - Callback invoked when the user denies the request
 * @param isLoading - When true, disables actions and replaces the approve label with "Authorizing..."
 * @returns A React element representing the consent UI
 */
export default function ConsentScreen({
    clientName,
    clientLogoUrl,
    scopes,
    onApprove,
    onDeny,
    isLoading = false,
}: ConsentScreenProps) {
    const scopeDescriptions: Record<string, string> = {
        openid: "Access your identity",
        profile: "Access your profile information",
        email: "Access your email address",
    };

    return (
        <div className="space-y-6">
            <div className="space-y-1">
                <h2 className="text-xl font-semibold text-white">Authorize Application</h2>
                <p className="text-sm text-white/60">
                    <span className="font-medium text-white">{clientName}</span> wants to access your account
                </p>
            </div>

            {clientLogoUrl && (
                <div className="flex justify-center">
                    {/* eslint-disable-next-line @next/next/no-img-element */}
                    <img src={clientLogoUrl} alt={clientName} className="h-16 w-16 rounded-lg object-cover" />
                </div>
            )}

            {scopes.length > 0 && (
                <div className="space-y-3">
                    <h3 className="text-sm font-medium text-white">This application will be able to:</h3>
                    <ul className="space-y-2">
                        {scopes.map((scope) => (
                            <li key={scope} className="flex items-start gap-2 text-sm text-white/80">
                                <span className="mt-1 h-1.5 w-1.5 rounded-full bg-white/40" />
                                <span>{scopeDescriptions[scope] || scope}</span>
                            </li>
                        ))}
                    </ul>
                </div>
            )}

            <div className="flex gap-3 pt-2">
                <Button fullWidth variant="secondary" onClick={onDeny} disabled={isLoading}>
                    Deny
                </Button>
                <Button fullWidth variant="primary" onClick={onApprove} disabled={isLoading}>
                    {isLoading ? "Authorizing..." : "Approve"}
                </Button>
            </div>
        </div>
    );
}
