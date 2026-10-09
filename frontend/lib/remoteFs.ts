// Browse and download files on the Transfer drive through the Guacamole
// filesystem object guacd exposes ("onfilesystem"). Downloads must ack every
// blob: guacd only sends the next chunk after an ack (flow control).
import type * as G from "guacamole-common-js";
import type { Transfer } from "./fileTransfer";
import { saveBlob } from "./fileTransfer";

export type Entry = { path: string; name: string; dir: boolean };

const OK = 0x0000;
const UNSUPPORTED = 0x0100;

let seq = 1_000_000; // separate id space from uploads

export class RemoteFs {
  constructor(private Guac: typeof G, private object: G.Object) {}

  get indexMime(): string {
    return this.Guac.Object.STREAM_INDEX_MIMETYPE;
  }

  // Lists one directory. Paths are absolute within the drive ("/" = root).
  list(path = "/"): Promise<Entry[]> {
    return new Promise((resolve, reject) => {
      this.object.requestInputStream(path, (stream: G.InputStream, mimetype: string) => {
        if (mimetype !== this.indexMime) {
          stream.sendAck("Not a directory", UNSUPPORTED);
          return reject(new Error("not a directory"));
        }
        stream.sendAck("Ready", OK);
        const reader = new this.Guac.JSONReader(stream);
        reader.onprogress = () => stream.sendAck("Received", OK);
        reader.onend = () => {
          const index = reader.getJSON() as Record<string, string>;
          const entries = Object.entries(index).map(([p, m]) => ({
            path: p, name: p.split("/").filter(Boolean).pop() ?? p, dir: m === this.indexMime,
          }));
          entries.sort((a, b) => Number(b.dir) - Number(a.dir) || a.name.localeCompare(b.name));
          resolve(entries);
        };
      });
    });
  }

  download(entry: Entry, update: (t: Transfer) => void): void {
    this.object.requestInputStream(entry.path, (stream: G.InputStream, mimetype: string) => {
      receiveStream(this.Guac, stream, mimetype, entry.name, update);
    });
  }
}

// Shared by browser downloads and Transfer\Download auto-downloads.
export function receiveStream(Guac: typeof G, stream: G.InputStream, mimetype: string, name: string,
  update: (t: Transfer) => void): void {
  const t: Transfer = { id: ++seq, name, direction: "download", bytes: 0, total: null, state: "running" };
  update({ ...t });
  const reader = new Guac.BlobReader(stream, mimetype);
  reader.onprogress = (length: number) => {
    t.bytes += length;
    stream.sendAck("Received", OK);
    update({ ...t });
  };
  reader.onend = () => {
    saveBlob(reader.getBlob(), name);
    update({ ...t, state: "done" });
  };
  stream.sendAck("Ready", OK);
}
