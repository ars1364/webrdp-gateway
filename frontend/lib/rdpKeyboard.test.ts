import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type * as G from "guacamole-common-js";
import { attachKeyboard } from "./rdpKeyboard";
import { K } from "./keyCombos";

type Kb = { onkeydown: ((k: number) => boolean) | null; onkeyup: ((k: number) => void) | null; modifiers: Record<string, boolean>; reset: () => void };
let kb: Kb;
let blur: (() => void) | null = null;

beforeEach(() => {
  vi.stubGlobal("navigator", { platform: "Linux", userAgent: "Chrome" });
  vi.stubGlobal("window", { addEventListener: (_e: string, f: () => void) => { blur = f; }, removeEventListener: () => {} });
  vi.stubGlobal("document", { activeElement: null });
  vi.stubGlobal("HTMLTextAreaElement", class {});
  vi.stubGlobal("HTMLInputElement", class {});
});
afterEach(() => vi.unstubAllGlobals());

function setup(sticky: number[] = []) {
  const ev: Array<[number, number]> = [];
  const Guac = { Keyboard: class { constructor() { kb = this as unknown as Kb; kb.modifiers = { ctrl: false, alt: false, shift: false, meta: false }; } reset() {} } } as unknown as typeof G;
  const client = { sendKeyEvent: (p: number, k: number) => ev.push([p, k]) } as unknown as G.Client;
  const pasteTarget = { focus: () => {}, addEventListener: () => {}, removeEventListener: () => {}, value: "" } as unknown as HTMLTextAreaElement;
  let pending = [...sticky];
  const h = attachKeyboard(Guac, client, { interceptPaste: false, pasteTarget, onGesture: () => {}, onPaste: () => {},
    takeSticky: () => { const s = pending; pending = []; return s; } });
  return { ev, h };
}

describe("attachKeyboard", () => {
  it.each([
    ["Ctrl+Alt+End → Ctrl+Alt+Del", { ctrl: true, alt: true }, K.End, K.Del],
    ["Alt+PgUp → Alt+Tab", { alt: true }, K.PgUp, K.Tab],
    ["plain End stays End", {}, K.End, K.End],
  ])("%s", (_n, mods, key, sent) => {
    const { ev } = setup();
    Object.assign(kb.modifiers, mods);
    kb.onkeydown!(key);
    kb.onkeyup!(key);
    expect(ev).toEqual([[1, sent], [0, sent]]);
  });

  it("applies latched sticky modifiers to the next key, then releases them", () => {
    const { ev } = setup([K.Ctrl, K.Shift]);
    kb.onkeydown!(K.Esc);
    kb.onkeyup!(K.Esc);
    expect(ev).toEqual([[1, K.Ctrl], [1, K.Shift], [1, K.Esc], [0, K.Esc], [0, K.Shift], [0, K.Ctrl]]);
  });

  it("releases held keys when the tab loses focus (no stuck Alt)", () => {
    const { ev } = setup();
    kb.onkeydown!(K.Alt);
    blur!();
    expect(ev).toEqual([[1, K.Alt], [0, K.Alt]]);
  });
});
