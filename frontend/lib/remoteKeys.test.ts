import { describe, expect, it, vi } from "vitest";
import type * as G from "guacamole-common-js";
import { keysymFor, openInExplorer, TRANSFER_PATH } from "./remoteKeys";

describe("keysymFor", () => {
  it.each([["a", 0x61], ["\\", 0x5c], ["T", 0x54], ["é", 0xe9], ["ب", 0x01000628]])("%s", (ch, want) =>
    expect(keysymFor(ch)).toBe(want));
});

describe("openInExplorer", () => {
  it("presses Win+R, types the UNC path, then Enter", async () => {
    vi.useFakeTimers();
    const events: Array<[number, number]> = [];
    const client = { sendKeyEvent: (p: number, k: number) => events.push([p, k]) } as unknown as G.Client;
    const done = openInExplorer(client);
    await vi.runAllTimersAsync();
    await done;
    vi.useRealTimers();
    expect(events.slice(0, 4)).toEqual([[1, 0xffeb], [1, 0x72], [0, 0x72], [0, 0xffeb]]);
    const typed = events.slice(4, -2).filter(([p]) => p === 1).map(([, k]) => String.fromCharCode(k)).join("");
    expect(typed).toBe(TRANSFER_PATH);
    expect(events.slice(-2)).toEqual([[1, 0xff0d], [0, 0xff0d]]);
  });
});
