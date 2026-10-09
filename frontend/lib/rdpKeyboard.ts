// Keyboard wiring for an RDP session: forwards keys to guacd, keeps keys
// typed into our own inputs local, maps macOS Cmd → Ctrl, and intercepts
// Ctrl/Cmd+V so the local clipboard reaches Windows BEFORE the keystroke.
import type * as G from "guacamole-common-js";
import { isMac, KEYSYM } from "./clipboardSync";

type Opts = {
  interceptPaste: boolean;
  pasteTarget: HTMLTextAreaElement;
  onGesture: () => void;
  onPaste: (text: string) => void;
};

const PASTE_FALLBACK_MS = 400; // no paste event (empty/non-text clipboard) → send V anyway

export function attachKeyboard(Guac: typeof G, client: G.Client, o: Opts): () => void {
  const kb = new Guac.Keyboard(document);
  const mac = isMac();
  let heldV: number | null = null; // V keysym waiting for the paste event
  let vDelivered = false;
  let timer: ReturnType<typeof setTimeout> | undefined;

  const typingLocally = () => {
    const a = document.activeElement;
    return a !== o.pasteTarget && (a instanceof HTMLTextAreaElement || a instanceof HTMLInputElement);
  };
  const map = (k: number) => (mac && KEYSYM.metaKeys.has(k) ? KEYSYM.ControlL : k);
  const deliverV = () => {
    clearTimeout(timer);
    if (heldV === null) return;
    client.sendKeyEvent(1, heldV);
    client.sendKeyEvent(0, heldV);
    heldV = null;
    vDelivered = true;
  };

  kb.onkeydown = (raw: number) => {
    if (typingLocally()) return true;
    o.onGesture();
    if (document.activeElement !== o.pasteTarget) o.pasteTarget.focus({ preventScroll: true });
    const k = map(raw);
    const mod = kb.modifiers.ctrl || kb.modifiers.meta;
    if (o.interceptPaste && mod && (k === KEYSYM.v || k === KEYSYM.V)) {
      heldV = k;
      vDelivered = false;
      timer = setTimeout(deliverV, PASTE_FALLBACK_MS);
      return true; // let the browser fire a native paste event on pasteTarget
    }
    client.sendKeyEvent(1, k);
    return false;
  };

  kb.onkeyup = (raw: number) => {
    if (typingLocally()) return;
    const k = map(raw);
    if ((k === KEYSYM.v || k === KEYSYM.V) && (heldV !== null || vDelivered)) {
      if (heldV !== null) deliverV();
      vDelivered = false;
      return;
    }
    client.sendKeyEvent(0, k);
  };

  const onPaste = (e: ClipboardEvent) => {
    e.preventDefault();
    const text = e.clipboardData?.getData("text/plain") ?? "";
    o.pasteTarget.value = "";
    if (!text) return deliverV();
    o.onPaste(text);
    // Give guacd a beat to announce the new clipboard to Windows (CLIPRDR)
    // before the V keystroke asks for it.
    clearTimeout(timer);
    timer = setTimeout(deliverV, 60);
  };
  o.pasteTarget.addEventListener("paste", onPaste);

  return () => {
    clearTimeout(timer);
    o.pasteTarget.removeEventListener("paste", onPaste);
    kb.onkeydown = null;
    kb.onkeyup = null;
    kb.reset();
  };
}
