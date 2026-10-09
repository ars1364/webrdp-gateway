"use client";

import { useRef, useState } from "react";
import { MAX_TYPE_CHARS } from "@/lib/keyCombos";
import { Modal } from "./Modal";
import { Button } from "./ui";

type Props = {
  onType: (text: string, signal: AbortSignal, onProgress: (n: number) => void) => Promise<void>;
  onClose: () => void;
};

// "Paste as keystrokes" (TeamViewer pattern): for the Windows login screen,
// UAC prompts and apps that block paste. Text is typed one key at a time.
export function TypeTextDialog({ onType, onClose }: Props) {
  const [text, setText] = useState("");
  const [busy, setBusy] = useState(false);
  const [done, setDone] = useState(0);
  const [note, setNote] = useState("");
  const abort = useRef<AbortController | null>(null);
  const total = Math.min(Array.from(text).length, MAX_TYPE_CHARS);

  async function go(value: string) {
    abort.current = new AbortController();
    setBusy(true);
    setDone(0);
    await onType(value, abort.current.signal, setDone);
    setBusy(false);
    if (!abort.current.signal.aborted) onClose();
  }

  async function fromClipboard() {
    try {
      setText(await navigator.clipboard.readText());
    } catch {
      setNote("The browser blocked clipboard access: paste into the box instead.");
    }
  }

  return (
    <Modal label="Type text into the remote desktop" onClose={() => { abort.current?.abort(); onClose(); }}>
      <h2 className="text-base font-semibold">Type text into Windows</h2>
      <p className="mt-1 text-xs text-ink-2">
        Sends real keystrokes, so it works on the login screen and in admin prompts where paste doesn’t. Click the
        target field in Windows first. Max {MAX_TYPE_CHARS} characters.
      </p>
      <textarea value={text} onChange={(e) => setText(e.target.value)} rows={5} disabled={busy} aria-label="Text to type"
        className="mt-3 w-full rounded-lg border border-line bg-surface p-2 font-mono text-xs" placeholder="Text or password…" />
      {note && <p className="mt-1 text-xs text-ink-2">{note}</p>}
      {busy && (
        <progress className="mt-2 h-1 w-full accent-primary" max={total} value={done} aria-label="Typing progress" />
      )}
      <div className="mt-4 flex flex-wrap justify-end gap-2">
        <Button variant="ghost" onClick={() => void fromClipboard()} disabled={busy}>Use my clipboard</Button>
        {busy ? (
          <Button variant="danger" onClick={() => abort.current?.abort()}>Stop</Button>
        ) : (
          <Button onClick={() => void go(text)} disabled={!text}>Type it</Button>
        )}
      </div>
    </Modal>
  );
}
