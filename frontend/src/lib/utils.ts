import { type ClassValue, clsx } from "clsx";
import { twMerge } from "tailwind-merge";

/**
 * Combines class name inputs and returns a merged Tailwind-compatible class string.
 *
 * @param inputs - Class names or values accepted by `clsx` (strings, arrays, objects, etc.)
 * @returns The resulting class string with Tailwind class conflicts resolved
 */
export function cn(...inputs: ClassValue[]) {
    return twMerge(clsx(inputs));
}
