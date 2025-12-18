"use client";

export default class LocalStorageTokenService {
    private constructor() {}
    private static readonly OIDC_CODE_VERIFIER_KEY = "Authly-OIDC-Verifier";

    private static getItem(key: string): string | null {
        try {
            if (typeof window !== "undefined" && window.localStorage) {
                return localStorage.getItem(key);
            }
        } catch (e) {
            console.warn(`Failed to get item ${key} from localStorage`, e);
        }
        return null;
    }

    private static setItem(key: string, value: string): void {
        try {
            if (typeof window !== "undefined" && window.localStorage) {
                if (value) {
                    localStorage.setItem(key, value);
                } else {
                    localStorage.removeItem(key);
                }
            }
        } catch (e) {
            console.warn(`Failed to set item ${key} to localStorage`, e);
        }
    }

    private static removeItem(key: string): void {
        try {
            if (typeof window !== "undefined" && window.localStorage) {
                localStorage.removeItem(key);
            }
        } catch (e) {
            console.warn(`Failed to remove item ${key} from localStorage`, e);
        }
    }

    public static get oidcCodeVerifier(): string | null {
        return this.getItem(this.OIDC_CODE_VERIFIER_KEY);
    }

    public static setOidcCodeVerifier(verifier: string) {
        this.setItem(this.OIDC_CODE_VERIFIER_KEY, verifier);
    }

    public static clear() {
        this.removeItem(this.OIDC_CODE_VERIFIER_KEY);
    }
}
