// Special keys and combos for the remote desktop (X11 keysyms).
import type * as G from "guacamole-common-js";
import { keysymFor } from "./remoteKeys";

export const K = {
  Ctrl: 0xffe3, Alt: 0xffe9, Shift: 0xffe1, Win: 0xffeb,
  Esc: 0xff1b, Tab: 0xff09, Del: 0xffff, End: 0xff57, Home: 0xff50, PgUp: 0xff55,
  Enter: 0xff0d, Print: 0xff61, F1: 0xffbe,
} as const;

export type Modifier = "Ctrl" | "Alt" | "Shift" | "Win";
export const MODIFIERS: Modifier[] = ["Ctrl", "Alt", "Shift", "Win"];

export type Combo = { id: string; label: string; hint: string; keys: number[] };

const letter = (c: string) => keysymFor(c);

export const COMBOS: Combo[] = [
  { id: "cad", label: "Ctrl+Alt+Del", hint: "Security screen", keys: [K.Ctrl, K.Alt, K.Del] },
  { id: "taskmgr", label: "Ctrl+Shift+Esc", hint: "Task Manager", keys: [K.Ctrl, K.Shift, K.Esc] },
  { id: "win", label: "Win", hint: "Start menu", keys: [K.Win] },
  { id: "run", label: "Win+R", hint: "Run…", keys: [K.Win, letter("r")] },
  { id: "explorer", label: "Win+E", hint: "File Explorer", keys: [K.Win, letter("e")] },
  { id: "desktop", label: "Win+D", hint: "Show desktop", keys: [K.Win, letter("d")] },
  { id: "lock", label: "Win+L", hint: "Lock Windows", keys: [K.Win, letter("l")] },
  { id: "winx", label: "Win+X", hint: "Power-user menu", keys: [K.Win, letter("x")] },
  { id: "snip", label: "Win+Shift+S", hint: "Snipping tool", keys: [K.Win, K.Shift, letter("s")] },
  { id: "alttab", label: "Alt+Tab", hint: "Switch windows", keys: [K.Alt, K.Tab] },
  { id: "altf4", label: "Alt+F4", hint: "Close window", keys: [K.Alt, K.F1 + 3] },
  { id: "print", label: "PrtScr", hint: "Screenshot (in Windows)", keys: [K.Print] },
  { id: "altprint", label: "Alt+PrtScr", hint: "Window screenshot (in Windows)", keys: [K.Alt, K.Print] },
  { id: "esc", label: "Esc", hint: "Escape", keys: [K.Esc] },
];

export const FKEYS = Array.from({ length: 12 }, (_, i) => ({ label: `F${i + 1}`, keysym: K.F1 + i }));

// Presses keys in order, releases in reverse: a correct chord for Windows.
export function sendCombo(client: G.Client, keys: number[]): void {
  keys.forEach((k) => client.sendKeyEvent(1, k));
  [...keys].reverse().forEach((k) => client.sendKeyEvent(0, k));
}

const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms));
export const MAX_TYPE_CHARS = 4096;

// Types text as real keystrokes (works on the Windows login screen / UAC,
// where the clipboard is unavailable). Newlines become Enter.
export async function typeText(client: G.Client, text: string, signal: AbortSignal, onProgress: (done: number) => void): Promise<void> {
  const chars = Array.from(text.replace(/\r\n?/g, "\n")).slice(0, MAX_TYPE_CHARS);
  for (let i = 0; i < chars.length; i++) {
    if (signal.aborted) return;
    const k = chars[i] === "\n" ? K.Enter : chars[i] === "\t" ? K.Tab : keysymFor(chars[i]);
    client.sendKeyEvent(1, k);
    client.sendKeyEvent(0, k);
    if (i % 20 === 19) {
      onProgress(i + 1);
      await sleep(30); // let RDP keep up; avoids dropped characters
    }
  }
  onProgress(chars.length);
}
