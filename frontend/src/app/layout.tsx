import type { Metadata, Viewport } from "next";
import { Sora, JetBrains_Mono } from "next/font/google";
import "./globals.css";

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
 * Root layout for the application that applies global font variables and wraps page content.
 *
 * @param children - The application content to render inside the layout.
 * @returns The root HTML structure (`<html>` and `<body>`) containing `children`.
 */
export default function RootLayout({ children }: { children: React.ReactNode }) {
    return (
        <html lang="en" className={`${sora.variable} ${mono.variable}`}>
            <body className="font-display antialiased">{children}</body>
        </html>
    );
}
