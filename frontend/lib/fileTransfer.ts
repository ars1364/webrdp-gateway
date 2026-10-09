// File transfer over the Guacamole "Transfer" drive: uploads stream into the
// drive root. Downloads live in remoteFs.ts (browser + Transfer\Download).
import type * as G from "guacamole-common-js";

export type Transfer = {
  id: number;
  name: string;
  direction: "upload" | "download";
  bytes: number;
  total: number | null; // unknown for downloads
  state: "running" | "done" | "error";
  error?: string;
};

type Update = (t: Transfer) => void;

let seq = 0;

export function uploadFile(Guac: typeof G, client: G.Client, file: File, update: Update): void {
  const t: Transfer = { id: ++seq, name: file.name, direction: "upload", bytes: 0, total: file.size, state: "running" };
  update({ ...t });
  const stream = client.createFileStream(file.type || "application/octet-stream", file.name);
  const writer = new Guac.BlobWriter(stream);
  writer.onprogress = (_b: Blob, offset: number) => update({ ...t, bytes: offset });
  writer.oncomplete = () => update({ ...t, bytes: file.size, state: "done" });
  writer.onerror = (_b: Blob, _o: number, e: Error) => update({ ...t, state: "error", error: e.message });
  writer.onack = (status: G.Status) => {
    if (status.isError()) update({ ...t, state: "error", error: status.message || "Upload refused by the server." });
  };
  writer.sendBlob(file);
}

export function saveBlob(blob: Blob, name: string): void {
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = name;
  a.rel = "noopener";
  document.body.appendChild(a);
  a.click();
  a.remove();
  setTimeout(() => URL.revokeObjectURL(url), 60_000);
}

export function formatBytes(n: number): string {
  if (n < 1024) return `${n} B`;
  if (n < 1024 ** 2) return `${(n / 1024).toFixed(1)} KB`;
  if (n < 1024 ** 3) return `${(n / 1024 ** 2).toFixed(1)} MB`;
  return `${(n / 1024 ** 3).toFixed(2)} GB`;
}
