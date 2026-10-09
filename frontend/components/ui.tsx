// Small shared primitives so every form looks the same.
import type { ButtonHTMLAttributes, InputHTMLAttributes, ReactNode } from "react";

export function Card({ children, className = "" }: { children: ReactNode; className?: string }) {
  return (
    <section className={`rounded-xl border border-line bg-surface p-5 shadow-sm ${className}`}>{children}</section>
  );
}

type FieldProps = InputHTMLAttributes<HTMLInputElement> & { label: string; error?: string };

export function Field({ label, error, id, className = "", ...rest }: FieldProps) {
  const fid = id ?? `f-${label.toLowerCase().replace(/\W+/g, "-")}`;
  return (
    <label htmlFor={fid} className={`flex flex-col gap-1 text-sm ${className}`}>
      <span className="font-medium text-ink-2">{label}</span>
      <input
        id={fid}
        aria-invalid={!!error}
        className="h-10 rounded-lg border border-line bg-surface px-3 text-ink placeholder:text-muted focus:border-primary"
        {...rest}
      />
      {error && <span className="text-xs text-red-700">{error}</span>}
    </label>
  );
}

type BtnProps = ButtonHTMLAttributes<HTMLButtonElement> & { variant?: "primary" | "ghost" | "danger" };

export function Button({ variant = "primary", className = "", ...rest }: BtnProps) {
  const look = {
    primary: "bg-primary text-white hover:bg-primary-hover",
    ghost: "border border-line bg-surface text-ink hover:bg-canvas",
    danger: "border border-red-200 bg-surface text-red-700 hover:bg-red-50",
  }[variant];
  return (
    <button
      className={`inline-flex h-10 items-center justify-center gap-2 rounded-lg px-4 text-sm font-medium transition-colors disabled:opacity-50 ${look} ${className}`}
      {...rest}
    />
  );
}

export function Alert({ children }: { children: ReactNode }) {
  return (
    <p role="alert" className="rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-800">
      {children}
    </p>
  );
}
