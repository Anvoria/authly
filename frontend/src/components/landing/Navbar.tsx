"use client";

import { useState, useEffect } from "react";
import Link from "next/link";
import Image from "next/image";
import { Menu, X } from "lucide-react";
import Button from "@/authly/components/ui/Button";
import { useAuthStatus } from "@/authly/lib/hooks/useAuth";

/**
 * Renders the top navigation bar with responsive layout, scroll-aware styling, mobile collapse behavior, and authentication-aware CTAs.
 *
 * The component updates its visual style when the page is scrolled, shows a desktop navigation and CTA area, and provides a collapsible mobile menu. CTA buttons and menu actions adapt based on authentication status.
 *
 * @returns A JSX element representing the navigation bar.
 */
export default function Navbar() {
    const [isOpen, setIsOpen] = useState(false);
    const [scrolled, setScrolled] = useState(false);
    const { data: authStatus } = useAuthStatus();

    useEffect(() => {
        const handleScroll = () => {
            setScrolled(window.scrollY > 20);
        };
        window.addEventListener("scroll", handleScroll);
        return () => window.removeEventListener("scroll", handleScroll);
    }, []);

    const navLinks = [
        { href: "#features", label: "FEATURES" },
        { href: "#docs", label: "DOCS" },
        {
            href: "https://github.com/anvoria/authly",
            label: "GITHUB",
            external: true,
        },
    ];

    return (
        <>
            <nav
                className={`fixed top-0 left-0 right-0 z-50 transition-all duration-300 ${
                    scrolled
                        ? "bg-black/95 backdrop-blur-md border-b border-white/10"
                        : "bg-black/80 backdrop-blur-sm border-b border-white/5"
                }`}
            >
                <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
                    <div className="flex items-center justify-between h-16 md:h-20">
                        {/* Logo */}
                        <Link href="/" className="flex items-center gap-3 group transition-opacity hover:opacity-80">
                            <div className="relative w-7 h-7 md:w-8 md:h-8 shrink-0">
                                <Image
                                    src="/images/logo.svg"
                                    alt="Authly Logo"
                                    fill
                                    className="object-contain filter brightness-0 invert"
                                    priority
                                />
                            </div>
                            <span className="text-lg md:text-xl font-medium tracking-[0.2em] text-white hidden sm:block">
                                AUTHLY
                            </span>
                        </Link>

                        {/* Desktop Navigation */}
                        <div className="hidden lg:flex items-center space-x-8 xl:space-x-12">
                            {navLinks.map((link) => (
                                <Link
                                    key={link.href}
                                    href={link.href}
                                    target={link.external ? "_blank" : undefined}
                                    rel={link.external ? "noopener noreferrer" : undefined}
                                    className="text-sm tracking-wide text-white/60 hover:text-white transition-colors duration-200 relative group"
                                >
                                    {link.label}
                                    <span className="absolute bottom-0 left-0 w-0 h-px bg-white group-hover:w-full transition-all duration-300" />
                                </Link>
                            ))}
                        </div>

                        {/* Desktop CTA Buttons */}
                        <div className="hidden lg:flex items-center gap-3">
                            {authStatus?.authenticated ? (
                                <Button href="/dashboard" variant="primary" size="sm">
                                    DASHBOARD
                                </Button>
                            ) : (
                                <>
                                    <Button href="/login" variant="ghost" size="sm">
                                        SIGN IN
                                    </Button>
                                    <Button href="/register" variant="primary" size="sm">
                                        GET STARTED
                                    </Button>
                                </>
                            )}
                        </div>

                        {/* Mobile Menu Button */}
                        <button
                            onClick={() => setIsOpen(!isOpen)}
                            className="lg:hidden flex items-center justify-center w-11 h-11 cursor-pointer transition-opacity hover:opacity-80"
                            aria-label="Toggle menu"
                            aria-expanded={isOpen}
                        >
                            {isOpen ? (
                                <X className="w-6 h-6 text-white" strokeWidth={2} strokeLinecap="square" />
                            ) : (
                                <Menu className="w-6 h-6 text-white" strokeWidth={2} strokeLinecap="square" />
                            )}
                        </button>
                    </div>
                </div>

                {/* Mobile Menu */}
                <div
                    className={`lg:hidden overflow-hidden transition-all duration-300 ease-in-out ${
                        isOpen ? "max-h-96 opacity-100" : "max-h-0 opacity-0"
                    }`}
                >
                    <div className="px-4 pb-6 pt-4 space-y-3 border-t border-white/10">
                        {navLinks.map((link) => (
                            <Link
                                key={link.href}
                                href={link.href}
                                target={link.external ? "_blank" : undefined}
                                rel={link.external ? "noopener noreferrer" : undefined}
                                onClick={() => setIsOpen(false)}
                                className="block text-sm tracking-wide text-white/60 hover:text-white transition-colors py-2"
                            >
                                {link.label}
                            </Link>
                        ))}
                        <div className="flex flex-col gap-3 pt-4 border-t border-white/5">
                            {authStatus?.authenticated ? (
                                <Button
                                    href="/dashboard"
                                    variant="primary"
                                    size="sm"
                                    fullWidth
                                    onClick={() => setIsOpen(false)}
                                >
                                    DASHBOARD
                                </Button>
                            ) : (
                                <>
                                    <Button
                                        href="/login"
                                        variant="ghost"
                                        size="sm"
                                        fullWidth
                                        onClick={() => setIsOpen(false)}
                                    >
                                        SIGN IN
                                    </Button>
                                    <Button
                                        href="/register"
                                        variant="primary"
                                        size="sm"
                                        fullWidth
                                        onClick={() => setIsOpen(false)}
                                    >
                                        GET STARTED
                                    </Button>
                                </>
                            )}
                        </div>
                    </div>
                </div>
            </nav>
        </>
    );
}
