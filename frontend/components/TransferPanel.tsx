"use client";

import { useRef } from "react";
import { formatBytes, type Transfer } from "@/lib/fileTransfer";
import { Button } from "./ui";

type Props = {
  transfers: Transfer[];
  canUpload: boolean;
  canDownload: boolean;
  maxMB: number;
  onPick: (files: FileList) => void;
  onClose: () => void;
};

// Files panel (Kasm / Guacamole pattern): upload button, how-to for
// downloads, and a live list of transfers with progress.
export function TransferPanel({ transfers, canUpload, canDownload, maxMB, onPick, onClose }: Props) {
  const input = useRef<HTMLInputElement>(null);
  return (
    <aside aria-label="Files"
      className="absolute end-3 top-14 z-10 flex max-h-[70vh] w-[min(24rem,calc(100vw-1.5rem))] flex-col gap-3 overflow-y-auto rounded-xl border border-line bg-surface p-4 text-sm text-ink shadow-sm">
      <div className="flex items-center">
        <h2 className="font-semibold">Files</h2>
        <Button variant="ghost" className="ms-auto h-8" onClick={onClose} aria-label="Close files panel">Close</Button>
      </div>
      {canUpload && (
        <div className="flex flex-col gap-2">
          <p className="text-xs text-ink-2">
            Drop files anywhere on the session, or pick them here. They appear in Windows under
            <b> This PC → Transfer</b>. Limit {maxMB} MB per session.
          </p>
          <input ref={input} type="file" multiple className="hidden" aria-label="Choose files to upload"
            onChange={(e) => { if (e.target.files?.length) onPick(e.target.files); e.target.value = ""; }} />
          <Button className="h-8 self-start" onClick={() => input.current?.click()}>Upload files…</Button>
        </div>
      )}
      {canDownload && (
        <p className="text-xs text-ink-2">
          To download: in Windows, copy files into <b>Transfer\Download</b>. Your browser saves them automatically.
        </p>
      )}
      {transfers.length > 0 && (
        <ul className="flex flex-col gap-2" aria-label="Transfers">
          {transfers.map((t) => {
            const pct = t.total ? Math.round((t.bytes / Math.max(t.total, 1)) * 100) : null;
            return (
              <li key={t.id} className="flex flex-col gap-1">
                <div className="flex gap-2 text-xs">
                  <span aria-hidden="true">{t.direction === "upload" ? "↑" : "↓"}</span>
                  <span className="min-w-0 flex-1 truncate" title={t.name}>{t.name}</span>
                  <span className={t.state === "error" ? "text-red-700" : "text-ink-2"}>
                    {t.state === "error" ? "Failed" : t.state === "done" ? "Done" : pct !== null ? `${pct}%` : formatBytes(t.bytes)}
                  </span>
                </div>
                {t.state === "running" && (
                  <progress className="h-1 w-full accent-primary" max={t.total ?? undefined}
                    value={t.total ? t.bytes : undefined} aria-label={`Progress for ${t.name}`} />
                )}
                {t.error && <span className="text-xs text-red-700">{t.error}</span>}
              </li>
            );
          })}
        </ul>
      )}
    </aside>
  );
}
