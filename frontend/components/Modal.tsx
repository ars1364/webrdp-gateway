"use client";

import { useEffect, useRef, type ReactNode } from "react";

type Props = {
  label: string;
  onClose: () => void;
  children: ReactNode;
  variant?: "dialog" | "fullscreen";
  closeOnEscape?: boolean;
};

// The one modal shell: backdrop, aria-modal, Escape, focus on open and
// focus restore on close. Never open-code a backdrop elsewhere.
export function Modal({ label, onClose, children, variant = "dialog", closeOnEscape = true }: Props) {
  const panel = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const previous = document.activeElement as HTMLElement | null;
    panel.current?.focus();
    const onKey = (e: KeyboardEvent) => {
      if (closeOnEscape && e.key === "Escape") onClose();
    };
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("keydown", onKey);
      previous?.focus();
    };
  }, [onClose, closeOnEscape]);

  if (variant === "fullscreen") {
    return (
      <div ref={panel} tabIndex={-1} role="dialog" aria-modal="true" aria-label={label}
        className="fixed inset-0 z-50 flex flex-col bg-stage outline-none">
        {children}
      </div>
    );
  }
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
      <div ref={panel} tabIndex={-1} role="dialog" aria-modal="true" aria-label={label}
        className="w-full max-w-md rounded-xl border border-line bg-surface p-5 shadow-sm outline-none">
        {children}
      </div>
    </div>
  );
}
