// Keyboard wiring for an RDP session: forwards keys to guacd, keeps keys
// typed into our own inputs local, maps macOS Cmd → Ctrl, intercepts
// Ctrl/Cmd+V so the local clipboard reaches Windows BEFORE the keystroke,
// applies sticky modifiers and fallback hotkeys, and releases held keys when
// the tab loses focus (no stuck Alt/Ctrl after you switch windows).
import type * as G from "guacamole-common-js";
import { isMac, KEYSYM } from "./clipboardSync";
import { K } from "./keyCombos";

type Opts = {
  interceptPaste: boolean;
  pasteTarget: HTMLTextAreaElement;
  onGesture: () => void;
  onPaste: (text: string) => void;
  takeSticky: () => number[]; // latched modifiers to apply to the next key (and clear)
};

export type KeyboardHandle = { detach: () => void; releaseAll: () => void };

const PASTE_FALLBACK_MS = 400; // no paste event (empty/non-text clipboard) → send V anyway
const MOD_KEYS = new Set([0xffe1, 0xffe2, 0xffe3, 0xffe4, 0xffe9, 0xffea, 0xffe7, 0xffe8, 0xffeb, 0xffec]);

export function attachKeyboard(Guac: typeof G, client: G.Client, o: Opts): KeyboardHandle {
  const kb = new Guac.Keyboard(document);
  const mac = isMac();
  const down = new Set<number>();          // keys we told Windows are pressed
  const substituted = new Map<number, number>(); // physical keysym → keysym actually sent
  const stickyFor = new Map<number, number[]>(); // key → latched modifiers to release with it
  let heldV: number | null = null; // V keysym waiting for the paste event
  let vDelivered = false;
  let timer: ReturnType<typeof setTimeout> | undefined;

  const press = (k: number) => { down.add(k); client.sendKeyEvent(1, k); };
  const release = (k: number) => { down.delete(k); client.sendKeyEvent(0, k); };
  const typingLocally = () => {
    const a = document.activeElement;
    return a !== o.pasteTarget && (a instanceof HTMLTextAreaElement || a instanceof HTMLInputElement);
  };
  const map = (k: number) => (mac && KEYSYM.metaKeys.has(k) ? KEYSYM.ControlL : k);
  const deliverV = () => {
    clearTimeout(timer);
    if (heldV === null) return;
    press(heldV);
    release(heldV);
    heldV = null;
    vDelivered = true;
  };

  // AVD/Teradici fallbacks for combos the local OS swallows.
  const substitute = (k: number): number => {
    const m = kb.modifiers;
    if (m.ctrl && m.alt && k === K.End) return K.Del;      // Ctrl+Alt+End → Ctrl+Alt+Del
    if (m.alt && !m.ctrl && k === K.PgUp) return K.Tab;    // Alt+PgUp → Alt+Tab
    if (m.alt && !m.ctrl && k === K.Home) {                // Alt+Home → Win
      [0xffe9, 0xffea].forEach((a) => down.has(a) && release(a));
      return K.Win;
    }
    return k;
  };

  kb.onkeydown = (raw: number) => {
    if (typingLocally()) return true;
    o.onGesture();
    if (document.activeElement !== o.pasteTarget) o.pasteTarget.focus({ preventScroll: true });
    let k = map(raw);
    const mod = kb.modifiers.ctrl || kb.modifiers.meta;
    if (o.interceptPaste && mod && (k === KEYSYM.v || k === KEYSYM.V)) {
      heldV = k;
      vDelivered = false;
      timer = setTimeout(deliverV, PASTE_FALLBACK_MS);
      return true; // let the browser fire a native paste event on pasteTarget
    }
    if (!MOD_KEYS.has(k)) {
      const sticky = o.takeSticky().filter((s) => !down.has(s));
      if (sticky.length) {
        sticky.forEach(press);
        stickyFor.set(raw, sticky);
      }
      const sent = substitute(k);
      if (sent !== k) substituted.set(raw, sent);
      k = sent;
    }
    press(k);
    return false;
  };

  kb.onkeyup = (raw: number) => {
    if (typingLocally()) return;
    const k = substituted.get(raw) ?? map(raw);
    substituted.delete(raw);
    if ((k === KEYSYM.v || k === KEYSYM.V) && (heldV !== null || vDelivered)) {
      if (heldV !== null) deliverV();
      vDelivered = false;
      return;
    }
    release(k);
    const sticky = stickyFor.get(raw);
    if (sticky) {
      [...sticky].reverse().forEach(release);
      stickyFor.delete(raw);
    }
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

  const releaseAll = () => {
    [...down].forEach(release);
    substituted.clear();
    stickyFor.clear();
    kb.reset();
  };
  window.addEventListener("blur", releaseAll);

  return {
    releaseAll,
    detach: () => {
      clearTimeout(timer);
      window.removeEventListener("blur", releaseAll);
      o.pasteTarget.removeEventListener("paste", onPaste);
      releaseAll();
      kb.onkeydown = null;
      kb.onkeyup = null;
    },
  };
}
