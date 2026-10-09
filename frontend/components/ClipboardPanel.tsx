"use client";

import { useState } from "react";
import { MAX_CLIPBOARD_BYTES, writeLocal } from "@/lib/clipboard";
import { Button } from "./ui";

type Props = {
  remoteText: string;
  canSend: boolean;
  onSend: (text: string) => void;
  onClose: () => void;
};

// Manual fallback (Kasm / Bastion pattern) for browsers or policies that
// block automatic sync. Remote content stays hidden until the user asks to
// see it (Guacamole pattern): clipboards often hold secrets.
export function ClipboardPanel({ remoteText, canSend, onSend, onClose }: Props) {
  const [draft, setDraft] = useState("");
  const [note, setNote] = useState("");
  const [revealed, setRevealed] = useState(false);
  const tooBig = new TextEncoder().encode(draft).length > MAX_CLIPBOARD_BYTES;

  async function copy() {
    setNote((await writeLocal(remoteText)) ? "Copied to your clipboard." : "Browser blocked it: select the text and copy.");
  }

  return (
    <aside aria-label="Clipboard"
      className="absolute end-3 top-14 z-10 flex w-[min(24rem,calc(100vw-1.5rem))] flex-col gap-3 rounded-xl border border-line bg-surface p-4 text-sm text-ink shadow-sm">
      <div className="flex items-center">
        <h2 className="font-semibold">Clipboard</h2>
        <Button variant="ghost" className="ms-auto h-8" onClick={onClose} aria-label="Close clipboard panel">Close</Button>
      </div>
      <p className="text-xs text-ink-2">Ctrl/Cmd+V inside the session already pastes your text into Windows. Use this panel only if that fails.</p>
      <div className="flex flex-col gap-1">
        <span className="font-medium text-ink-2">Last copied on the remote desktop</span>
        {revealed ? (
          <textarea readOnly value={remoteText} rows={3} aria-label="Last remote copy" placeholder="Nothing copied yet"
            className="rounded-lg border border-line bg-canvas p-2 font-mono text-xs" />
        ) : (
          <Button variant="ghost" className="h-8 self-start" onClick={() => setRevealed(true)}>
            Show clipboard contents
          </Button>
        )}
      </div>
      <Button variant="ghost" className="h-8 self-start" disabled={!remoteText} onClick={copy}>Copy to my clipboard</Button>
      {canSend && (
        <>
          <label className="flex flex-col gap-1">
            <span className="font-medium text-ink-2">Send to the remote desktop</span>
            <textarea value={draft} rows={3} placeholder="Paste text here" onChange={(e) => setDraft(e.target.value)}
              className="rounded-lg border border-line bg-surface p-2 font-mono text-xs" />
          </label>
          {tooBig && <p className="text-xs text-red-700">Too large: max 256 KB.</p>}
          <Button className="h-8 self-start" disabled={!draft || tooBig}
            onClick={() => { onSend(draft); setNote("Sent. Paste with Ctrl+V on the remote desktop."); setDraft(""); }}>
            Send
          </Button>
        </>
      )}
      {note && <p role="status" className="text-xs text-ink-2">{note}</p>}
    </aside>
  );
}
