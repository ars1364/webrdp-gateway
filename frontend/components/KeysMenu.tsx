"use client";

import { COMBOS, FKEYS, MODIFIERS, type Modifier } from "@/lib/keyCombos";
import { Button } from "./ui";

type Props = {
  sticky: Set<Modifier>;
  onToggleSticky: (m: Modifier) => void;
  onCombo: (keys: number[]) => void;
  onTypeText: () => void;
  onReleaseAll: () => void;
  onClose: () => void;
};

// Special keys panel: system combos, F-keys, sticky modifiers (AppStream
// pattern) and fallback-hotkey help (AVD pattern).
export function KeysMenu({ sticky, onToggleSticky, onCombo, onTypeText, onReleaseAll, onClose }: Props) {
  const send = (keys: number[]) => { onCombo(keys); onClose(); };
  return (
    <aside aria-label="Keyboard"
      className="absolute end-3 top-14 z-10 flex max-h-[75vh] w-[min(26rem,calc(100vw-1.5rem))] flex-col gap-3 overflow-y-auto rounded-xl border border-line bg-surface p-4 text-sm text-ink shadow-sm">
      <div className="flex items-center">
        <h2 className="font-semibold">Keyboard</h2>
        <Button variant="ghost" className="ms-auto h-8" onClick={onClose} aria-label="Close keyboard panel">Close</Button>
      </div>

      <section className="flex flex-col gap-2">
        <h3 className="text-xs font-medium text-ink-2">Shortcuts</h3>
        <div className="grid grid-cols-2 gap-2">
          {COMBOS.map((c) => (
            <Button key={c.id} variant="ghost" className="h-auto flex-col items-start gap-0 py-1.5 text-start" onClick={() => send(c.keys)}>
              <span className="text-xs font-semibold">{c.label}</span>
              <span className="text-[11px] font-normal text-ink-2">{c.hint}</span>
            </Button>
          ))}
        </div>
      </section>

      <section className="flex flex-col gap-2">
        <h3 className="text-xs font-medium text-ink-2">Function keys</h3>
        <div className="grid grid-cols-6 gap-1.5">
          {FKEYS.map((f) => (
            <Button key={f.label} variant="ghost" className="h-8 px-0" onClick={() => send([f.keysym])}>{f.label}</Button>
          ))}
        </div>
      </section>

      <section className="flex flex-col gap-2">
        <h3 className="text-xs font-medium text-ink-2">Sticky keys: tap one or more, then press any key</h3>
        <div className="grid grid-cols-4 gap-1.5">
          {MODIFIERS.map((m) => (
            <Button key={m} variant={sticky.has(m) ? "primary" : "ghost"} className="h-8" aria-pressed={sticky.has(m)}
              onClick={() => onToggleSticky(m)}>
              {m}
            </Button>
          ))}
        </div>
      </section>

      <div className="flex flex-wrap gap-2">
        <Button className="h-8" onClick={() => { onTypeText(); onClose(); }}>Type text…</Button>
        <Button variant="ghost" className="h-8" onClick={() => { onReleaseAll(); onClose(); }}>Release all keys</Button>
      </div>

      <section className="rounded-lg bg-canvas p-3 text-xs text-ink-2">
        <p className="mb-1 font-medium text-ink">Hotkeys that always work</p>
        <ul className="list-inside list-disc">
          <li><b>Ctrl+Alt+End</b> → Ctrl+Alt+Del</li>
          <li><b>Alt+PgUp</b> → Alt+Tab</li>
          <li><b>Alt+Home</b> → Windows key</li>
          <li><b>Fullscreen</b> (Chrome/Edge) sends Win, Alt+Tab and Esc straight to Windows. Hold Esc to exit.</li>
        </ul>
      </section>
    </aside>
  );
}
