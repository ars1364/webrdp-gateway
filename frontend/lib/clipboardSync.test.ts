import { afterEach, describe, expect, it, vi } from "vitest";
import type * as G from "guacamole-common-js";
import { ClipboardSync, type ClipStatus } from "./clipboardSync";

// Minimal Guacamole fakes: capture what would go over the tunnel.
function fakes() {
  const sent: string[] = [];
  let onclipboard: ((s: unknown, m: string) => void) | null = null;
  const Guac = {
    StringWriter: class { constructor(public s: unknown) {} sendText(t: string) { sent.push(t); } sendEnd() {} },
    StringReader: class {
      ontext: ((t: string) => void) | null = null;
      onend: (() => void) | null = null;
      constructor(s: { feed: (r: unknown) => void }) { s.feed(this); }
    },
  } as unknown as typeof G;
  const client = {
    createClipboardStream: () => ({}),
    set onclipboard(f: (s: unknown, m: string) => void) { onclipboard = f; },
  } as unknown as G.Client;
  const remoteCopy = (text: string) => onclipboard?.({
    feed: (r: { ontext: (t: string) => void; onend: () => void }) => queueMicrotask(() => { r.ontext(text); r.onend(); }),
    sendAck: vi.fn(),
  }, "text/plain");
  return { Guac, client, sent, remoteCopy };
}

function browser({ perm = "granted", local = "", writeOk = true }: { perm?: string; local?: string; writeOk?: boolean }) {
  const written: string[] = [];
  vi.stubGlobal("navigator", {
    userAgent: "Chrome", platform: "Linux",
    permissions: { query: async () => ({ state: perm }) },
    clipboard: {
      readText: async () => local,
      writeText: async (t: string) => { if (!writeOk) throw new Error("blocked"); written.push(t); },
    },
  });
  return { written };
}

afterEach(() => vi.unstubAllGlobals());
const tick = () => new Promise((r) => setTimeout(r, 0));

describe("ClipboardSync", () => {
  it.each<[string, { upload: boolean; download: boolean; perm: string }, ClipStatus]>([
    ["both off", { upload: false, download: false, perm: "granted" }, "off"],
    ["granted", { upload: true, download: true, perm: "granted" }, "synced"],
    ["prompt", { upload: true, download: true, perm: "prompt" }, "needs-permission"],
    ["download only", { upload: false, download: true, perm: "prompt" }, "ready"],
  ])("status: %s", async (_n, o, want) => {
    browser({ perm: o.perm });
    const f = fakes();
    const statuses: ClipStatus[] = [];
    await new ClipboardSync(f.Guac, f.client, { ...o, onRemoteText: () => {}, onStatus: (s) => statuses.push(s) }).start();
    expect(statuses.at(-1)).toBe(want);
  });

  it("native paste sends to remote; disabled upload sends nothing", async () => {
    browser({});
    for (const upload of [true, false]) {
      const f = fakes();
      const s = new ClipboardSync(f.Guac, f.client, { upload, download: true, onRemoteText: () => {}, onStatus: () => {} });
      await s.start();
      s.pasted("hello");
      expect(f.sent).toEqual(upload ? ["hello"] : []);
    }
  });

  it("gesture pulls the local clipboard only when it changed", async () => {
    browser({ local: "abc" });
    const f = fakes();
    const s = new ClipboardSync(f.Guac, f.client, { upload: true, download: true, onRemoteText: () => {}, onStatus: () => {} });
    await s.start();
    await s.gesture();
    await new Promise((r) => setTimeout(r, 510));
    await s.gesture(); // same text → no resend
    expect(f.sent).toEqual(["abc"]);
  });

  it("never pushes an empty or stale local clipboard over a pending remote copy", async () => {
    browser({ local: "old local text", writeOk: false });
    const f = fakes();
    const s = new ClipboardSync(f.Guac, f.client, { upload: true, download: true, onRemoteText: () => {}, onStatus: () => {} });
    await s.start();
    f.remoteCopy("fresh remote");
    await tick(); await tick();
    await s.gesture(); // write still blocked → must not send "old local text"
    expect(f.sent).toEqual([]);
  });

  it("remote copy: written locally, or queued and flushed on the next gesture", async () => {
    const b = browser({ writeOk: false });
    const f = fakes();
    const seen: Array<[string, boolean]> = [];
    const s = new ClipboardSync(f.Guac, f.client, { upload: true, download: true,
      onRemoteText: (t, ok) => seen.push([t, ok]), onStatus: () => {} });
    await s.start();
    f.remoteCopy("from windows");
    await tick(); await tick();
    expect(seen).toEqual([["from windows", false]]);
    (navigator.clipboard as unknown as { writeText: (t: string) => Promise<void> }).writeText = async (t) => { b.written.push(t); };
    await s.gesture();
    expect(b.written).toEqual(["from windows"]);
    expect(seen.at(-1)).toEqual(["from windows", true]);
    expect(f.sent).toEqual([]); // never echoed back to Windows
  });
});
