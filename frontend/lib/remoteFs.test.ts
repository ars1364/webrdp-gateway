import { describe, expect, it, vi } from "vitest";
import type * as G from "guacamole-common-js";
import { receiveStream, RemoteFs } from "./remoteFs";
import type { Transfer } from "./fileTransfer";

const INDEX = "application/vnd.glyptodon.guacamole.stream-index+json";

function fakeStream() {
  const acks: string[] = [];
  return { acks, stream: { sendAck: (m: string) => acks.push(m) } as unknown as G.InputStream };
}

function guacWith(readerScript: (r: Record<string, (...a: unknown[]) => void>) => void, json?: object) {
  return {
    Object: { STREAM_INDEX_MIMETYPE: INDEX },
    BlobReader: class {
      constructor() { queueMicrotask(() => readerScript(this as unknown as Record<string, (...a: unknown[]) => void>)); }
      getBlob() { return new Blob(["x"]); }
    },
    JSONReader: class {
      constructor() { queueMicrotask(() => readerScript(this as unknown as Record<string, (...a: unknown[]) => void>)); }
      getJSON() { return json; }
    },
  } as unknown as typeof G;
}

describe("receiveStream", () => {
  it("acks Ready first and Received for every chunk (guacd flow control)", async () => {
    vi.stubGlobal("URL", { createObjectURL: () => "blob:x", revokeObjectURL: () => {} });
    vi.stubGlobal("document", { createElement: () => ({ click() {}, remove() {} }), body: { appendChild() {} } });
    const { acks, stream } = fakeStream();
    const updates: Transfer[] = [];
    const Guac = guacWith((r) => { r.onprogress(4); r.onprogress(6); r.onend(); });
    receiveStream(Guac, stream, "text/plain", "a.txt", (t) => updates.push(t));
    await new Promise((r) => setTimeout(r, 0));
    expect(acks).toEqual(["Ready", "Received", "Received"]);
    expect(updates.at(-1)).toMatchObject({ state: "done", bytes: 10, name: "a.txt", direction: "download" });
    vi.unstubAllGlobals();
  });
});

describe("RemoteFs.list", () => {
  it.each([
    ["sorts folders first, then names", { "/b.txt": "text/plain", "/Download": INDEX, "/a.txt": "text/plain" },
      ["Download", "a.txt", "b.txt"], [true, false, false]],
    ["empty folder", {}, [], []],
  ])("%s", async (_n, index, names, dirs) => {
    const { acks, stream } = fakeStream();
    const Guac = guacWith((r) => { r.onprogress(1); r.onend(); }, index);
    const object = { requestInputStream: (_p: string, cb: (s: G.InputStream, m: string) => void) => cb(stream, INDEX) } as unknown as G.Object;
    const entries = await new RemoteFs(Guac, object).list("/");
    expect(entries.map((e) => e.name)).toEqual(names);
    expect(entries.map((e) => e.dir)).toEqual(dirs);
    expect(acks).toEqual(["Ready", "Received"]);
  });

  it("rejects (and refuses the stream) when the path is a file", async () => {
    const { acks, stream } = fakeStream();
    const object = { requestInputStream: (_p: string, cb: (s: G.InputStream, m: string) => void) => cb(stream, "text/plain") } as unknown as G.Object;
    await expect(new RemoteFs(guacWith(() => {}), object).list("/a.txt")).rejects.toThrow("not a directory");
    expect(acks).toEqual(["Not a directory"]);
  });
});
