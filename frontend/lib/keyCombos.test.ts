import { describe, expect, it } from "vitest";
import type * as G from "guacamole-common-js";
import { COMBOS, K, sendCombo, typeText } from "./keyCombos";

const rec = () => {
  const ev: Array<[number, number]> = [];
  return { ev, client: { sendKeyEvent: (p: number, k: number) => ev.push([p, k]) } as unknown as G.Client };
};

describe("sendCombo", () => {
  it.each(COMBOS.map((c) => [c.label, c.keys] as const))("%s presses in order, releases in reverse", (_l, keys) => {
    const { ev, client } = rec();
    sendCombo(client, keys);
    expect(ev).toEqual([...keys.map((k) => [1, k]), ...[...keys].reverse().map((k) => [0, k])]);
  });
});

describe("typeText", () => {
  it.each([
    ["plain", "ab", [0x61, 0x62]],
    ["newline → Enter", "a\r\nb", [0x61, K.Enter, 0x62]],
    ["tab", "a\tb", [0x61, K.Tab, 0x62]],
    ["unicode", "é", [0xe9]],
  ])("%s", async (_n, text, want) => {
    const { ev, client } = rec();
    await typeText(client, text, new AbortController().signal, () => {});
    expect(ev.filter(([p]) => p === 1).map(([, k]) => k)).toEqual(want);
  });

  it("stops when aborted", async () => {
    const { ev, client } = rec();
    const ac = new AbortController();
    ac.abort();
    await typeText(client, "hello", ac.signal, () => {});
    expect(ev).toEqual([]);
  });
});
