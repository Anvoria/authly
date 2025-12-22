import * as React from "react";
import { cn } from "@/authly/lib/utils";

export interface InputProps extends Omit<React.InputHTMLAttributes<HTMLInputElement>, "prefix" | "suffix"> {
    label?: string;
    description?: string;
    helperText?: string;
    error?: string;
    prefix?: React.ReactNode;
    suffix?: React.ReactNode;
}

const Input = React.forwardRef<HTMLInputElement, InputProps>(
    ({ className, label, description, helperText, error, prefix, suffix, id, disabled, ...props }, ref) => {
        const generatedId = React.useId();
        const inputId = id ?? generatedId;
        const helperId = helperText ? `${inputId}-helper` : undefined;
        const errorId = error ? `${inputId}-error` : undefined;
        const describedBy = [helperId, errorId].filter(Boolean).join(" ") || undefined;

        return (
            <div className="space-y-2 w-full">
                {label && (
                    <label
                        className="block text-xs font-medium text-white/60 uppercase tracking-widest ml-1"
                        htmlFor={inputId}
                    >
                        {label}
                        {props.required && <span className="text-red-500 ml-1">*</span>}
                    </label>
                )}

                {description && <p className="text-xs text-white/60 ml-1">{description}</p>}

                <div className={cn("relative", disabled && "opacity-50")}>
                    {prefix && (
                        <span className="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3 text-white/50">
                            {prefix}
                        </span>
                    )}

                    <input
                        id={inputId}
                        ref={ref}
                        className={cn(
                            "flex h-11 w-full border border-white/10 bg-white/5 px-4 text-sm text-white transition-[border-color,background-color] duration-200",
                            "rounded-none outline-none focus:outline-none",
                            "placeholder:text-white/30 hover:border-white/20",
                            "focus-visible:border-white/50 focus-visible:bg-white/10",
                            "disabled:cursor-not-allowed disabled:opacity-50",
                            prefix && "pl-10",
                            suffix && "pr-10",
                            error && "border-red-500 focus-visible:border-red-500",
                            className,
                        )}
                        aria-invalid={Boolean(error)}
                        aria-describedby={describedBy}
                        disabled={disabled}
                        {...props}
                    />

                    {suffix && (
                        <span className="pointer-events-none absolute inset-y-0 right-0 flex items-center pr-3 text-white/50">
                            {suffix}
                        </span>
                    )}
                </div>

                {helperText && (
                    <p className="text-xs text-white/60 ml-1" id={helperId}>
                        {helperText}
                    </p>
                )}

                {error && (
                    <p className="text-xs font-medium text-red-500 ml-1" id={errorId}>
                        {error}
                    </p>
                )}
            </div>
        );
    },
);
Input.displayName = "Input";

export default Input;
