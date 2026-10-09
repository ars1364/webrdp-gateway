"use client";

import { useCallback, useEffect, useState } from "react";
import type { Transfer } from "@/lib/fileTransfer";
import type { Entry, RemoteFs } from "@/lib/remoteFs";
import { Button } from "./ui";

type Props = { fs: RemoteFs | null; onTrack: (t: Transfer) => void };

const REFRESH_MS = 3000;

// Lists the Transfer drive so any file Windows puts there can be downloaded
// with one click (Guacamole file-browser pattern). Auto-refreshes while open.
export function FileBrowser({ fs, onTrack }: Props) {
  const [path, setPath] = useState("/");
  const [entries, setEntries] = useState<Entry[] | null>(null);
  const [error, setError] = useState("");

  const load = useCallback(async () => {
    if (!fs) return;
    try {
      setEntries(await fs.list(path));
      setError("");
    } catch {
      setError("Could not read the folder.");
    }
  }, [fs, path]);

  useEffect(() => {
    void load();
    const t = setInterval(() => void load(), REFRESH_MS);
    return () => clearInterval(t);
  }, [load]);

  if (!fs) {
    return <p className="text-xs text-ink-2">Waiting for Windows to attach the Transfer drive…</p>;
  }
  const up = path === "/" ? null : path.replace(/\/[^/]+\/?$/, "") || "/";
  return (
    <div className="flex flex-col gap-2">
      <div className="flex items-center gap-2">
        <span className="font-medium text-ink-2">In the shared folder</span>
        <code className="min-w-0 flex-1 truncate text-xs text-ink-2" title={path}>{path}</code>
        {up && <Button variant="ghost" className="h-7" onClick={() => setPath(up)}>Up</Button>}
        <Button variant="ghost" className="h-7" onClick={() => void load()} aria-label="Refresh file list">↻</Button>
      </div>
      {error && <p className="text-xs text-red-700">{error}</p>}
      {entries && entries.length === 0 && (
        <p className="text-xs text-ink-2">Empty. In Windows, copy files into \\tsclient\Transfer to download them here.</p>
      )}
      {entries && entries.length > 0 && (
        <ul className="flex flex-col divide-y divide-line rounded-lg border border-line" aria-label="Files in the shared folder">
          {entries.map((e) => (
            <li key={e.path} className="flex items-center gap-2 px-2 py-1.5 text-xs">
              <span aria-hidden="true">{e.dir ? "📁" : "📄"}</span>
              <span className="min-w-0 flex-1 truncate" title={e.name}>{e.name}</span>
              {e.dir ? (
                <Button variant="ghost" className="h-7" onClick={() => setPath(e.path)} aria-label={`Open folder ${e.name}`}>Open</Button>
              ) : (
                <Button className="h-7" onClick={() => fs.download(e, onTrack)} aria-label={`Download ${e.name}`}>Download</Button>
              )}
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
