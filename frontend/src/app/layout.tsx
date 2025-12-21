import type { Metadata, Viewport } from "next";
import { Sora, JetBrains_Mono } from "next/font/google";
import "./globals.css";
import QueryProvider from "@/authly/components/providers/QueryProvider";

const sora = Sora({
    subsets: ["latin"],
    variable: "--font-sora",
    display: "swap",
});

const mono = JetBrains_Mono({
    subsets: ["latin"],
    variable: "--font-mono",
    display: "swap",
});

export const metadata: Metadata = {
    title: {
        default: "Authly",
        template: "%s · Authly",
    },
    description:
        "Authly is a self-hosted authentication and authorization platform with OIDC, RBAC, sessions and fine-grained permissions.",
    applicationName: "Authly",
    generator: "Next.js",
    keywords: ["authentication", "authorization", "oidc", "oauth2", "rbac", "jwt", "identity", "auth service"],
    authors: [{ name: "Anvoria" }],
    creator: "Anvoria",
    publisher: "Anvoria",
};

export const viewport: Viewport = {
    width: "device-width",
    initialScale: 1,
    themeColor: "#000000",
};

/**
 * Provide the application's root HTML structure, apply global font variables, and wrap page content with the QueryProvider.
 *
 * @param children - The application content to render inside the layout.
 * @returns The `<html>` element (lang="en") containing a `<body>` whose children are wrapped by the QueryProvider.
 */
export default function RootLayout({ children }: { children: React.ReactNode }) {
    return (
        <html lang="en" className={`${sora.variable} ${mono.variable}`}>
            <body className="font-display antialiased">
                <QueryProvider>{children}</QueryProvider>
            </body>
        </html>
    );
}