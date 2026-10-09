"use client";

import type { ClipStatus } from "@/lib/clipboardSync";

type Props = { status: ClipStatus; onAllow: () => void; onOpenPanel: () => void };

const LABEL: Record<ClipStatus, string> = {
  off: "Clipboard off",
  ready: "Clipboard: Ctrl+V / copy",
  synced: "Clipboard synced",
  "needs-permission": "Allow clipboard sync",
  blocked: "Clipboard blocked",
};

const HINT: Record<ClipStatus, string> = {
  off: "Disabled by the administrator.",
  ready: "Ctrl/Cmd+V pastes your text into Windows; copies in Windows land in your clipboard. Open for the manual panel.",
  synced: "Your clipboard follows you into Windows on every click and key press.",
  "needs-permission": "Click to let the browser share your clipboard automatically.",
  blocked: "The browser denied clipboard access: use the lock icon in the address bar to allow it, then reconnect. Ctrl/Cmd+V still works.",
};

// Session-bar status indicator (Teleport / Chrome Remote Desktop pattern).
export function ClipboardChip({ status, onAllow, onOpenPanel }: Props) {
  if (status === "off") return null;
  const dot = status === "synced" ? "bg-green-500" : status === "blocked" ? "bg-red-500" : "bg-yellow-500";
  return (
    <button type="button" title={HINT[status]} onClick={status === "needs-permission" ? onAllow : onOpenPanel}
      className="inline-flex h-8 items-center gap-2 rounded-lg border border-white/15 px-3 text-xs text-white hover:bg-white/10">
      <span aria-hidden="true" className={`size-2 rounded-full ${dot}`} />
      {LABEL[status]}
    </button>
  );
}
