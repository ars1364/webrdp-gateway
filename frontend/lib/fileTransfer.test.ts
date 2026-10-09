import { describe, expect, it } from "vitest";
import type * as G from "guacamole-common-js";
import { formatBytes, uploadFile, type Transfer } from "./fileTransfer";

describe("formatBytes", () => {
  it.each([
    [0, "0 B"], [1023, "1023 B"], [1536, "1.5 KB"], [5 * 1024 ** 2, "5.0 MB"], [3 * 1024 ** 3, "3.00 GB"],
  ])("%d → %s", (n, want) => expect(formatBytes(n)).toBe(want));
});

describe("uploadFile", () => {
  function run(script: (w: Record<string, (...a: unknown[]) => void>) => void) {
    const updates: Transfer[] = [];
    let writer: Record<string, (...a: unknown[]) => void> = {};
    const Guac = {
      BlobWriter: class {
        constructor() { writer = this as unknown as typeof writer; }
        sendBlob() { queueMicrotask(() => script(writer)); }
      },
    } as unknown as typeof G;
    const client = { createFileStream: (_m: string, name: string) => ({ name }) } as unknown as G.Client;
    uploadFile(Guac, client, new File(["hello world"], "a.txt", { type: "text/plain" }), (t) => updates.push(t));
    return new Promise<Transfer[]>((r) => setTimeout(() => r(updates), 5));
  }

  it.each([
    ["completes", (w: Record<string, (...a: unknown[]) => void>) => { w.onprogress({}, 5); w.oncomplete({}); }, "done", 11],
    ["server refuses", (w: Record<string, (...a: unknown[]) => void>) => w.onack({ isError: () => true, message: "Upload disabled" }), "error", 0],
    ["read error", (w: Record<string, (...a: unknown[]) => void>) => w.onerror({}, 0, new Error("disk")), "error", 0],
  ])("%s", async (_n, script, state, bytes) => {
    const updates = await run(script);
    const last = updates.at(-1)!;
    expect(updates[0].state).toBe("running");
    expect(last.state).toBe(state);
    expect(last.bytes).toBe(bytes);
    expect(last.name).toBe("a.txt");
  });
});
