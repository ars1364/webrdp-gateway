"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useCallback, useEffect, useState } from "react";
import { api, ApiError, type Me, type Recording } from "@/lib/api";
import { formatBytes, saveBlob } from "@/lib/fileTransfer";
import { ConfirmDialog } from "@/components/ConfirmDialog";
import { RecordingPlayer } from "@/components/RecordingPlayer";
import { formatElapsed } from "@/components/useCapture";
import { Alert, Button, Card } from "@/components/ui";

function duration(r: Recording): string {
  if (!r.ended_at) return "in progress";
  return formatElapsed(Math.round((Date.parse(r.ended_at) - Date.parse(r.started_at)) / 1000));
}

export default function RecordingsPage() {
  const router = useRouter();
  const [me, setMe] = useState<Me | null>(null);
  const [items, setItems] = useState<Recording[] | null>(null);
  const [playing, setPlaying] = useState<Recording | null>(null);
  const [toDelete, setToDelete] = useState<Recording | null>(null);
  const [error, setError] = useState("");

  const load = useCallback(async () => {
    try {
      setMe(await api.me());
      setItems(await api.listRecordings());
    } catch (e) {
      if (e instanceof ApiError && e.status === 401) router.replace("/login/");
      else setError("Could not load recordings.");
    }
  }, [router]);

  useEffect(() => { void load(); }, [load]);

  async function download(r: Recording) {
    try {
      saveBlob(await api.recordingBlob(r.id), `webrdp-${r.id}.guac`);
    } catch {
      setError("Download failed.");
    }
  }

  async function remove(r: Recording) {
    setToDelete(null);
    try {
      await api.deleteRecording(r.id);
      setItems(await api.listRecordings());
    } catch (e) {
      setError(e instanceof ApiError ? e.message : "Delete failed.");
    }
  }

  return (
    <main className="mx-auto flex max-w-4xl flex-col gap-5 p-4 sm:p-8">
      <header className="flex items-center gap-3">
        <h1 className="text-xl font-semibold">Session recordings</h1>
        <Link href="/" className="ms-auto text-sm text-primary underline-offset-2 hover:underline">← Back to connections</Link>
      </header>
      {error && <Alert>{error}</Alert>}
      {me && !me.features?.session_recording && (
        <Alert>Session recording is turned off by the administrator. Older recordings stay listed until they expire.</Alert>
      )}
      <Card>
        {items === null ? (
          <p className="text-sm text-muted">Loading…</p>
        ) : items.length === 0 ? (
          <p className="text-sm text-muted">No recordings yet. New sessions are recorded while the feature is on.</p>
        ) : (
          <ul className="divide-y divide-line">
            {items.map((r) => (
              <li key={r.id} className="flex flex-wrap items-center gap-3 py-3">
                <div className="min-w-0 flex-1">
                  <p className="truncate font-medium" title={r.label}>{r.label || r.target}</p>
                  <p className="truncate text-sm text-ink-2">
                    {new Date(r.started_at).toLocaleString()} · {duration(r)} · {formatBytes(r.size_bytes)} · {r.target}
                  </p>
                </div>
                <Button onClick={() => setPlaying(r)} disabled={!r.ended_at}>Play</Button>
                <Button variant="ghost" onClick={() => void download(r)} disabled={!r.ended_at}>Download</Button>
                {me?.role === "admin" && (
                  <Button variant="danger" onClick={() => setToDelete(r)} aria-label={`Delete recording ${r.label}`}>Delete</Button>
                )}
              </li>
            ))}
          </ul>
        )}
      </Card>
      {toDelete && (
        <ConfirmDialog title="Delete this recording?" body="The recording file is destroyed. This is logged in the audit trail."
          confirmLabel="Delete" onConfirm={() => void remove(toDelete)} onCancel={() => setToDelete(null)} />
      )}
      {playing && <RecordingPlayer rec={playing} onClose={() => setPlaying(null)} />}
    </main>
  );
}
