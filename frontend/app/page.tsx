"use client";

import { useRouter } from "next/navigation";
import { useCallback, useEffect, useState } from "react";
import { api, ApiError, type Connection, type ConnectionInput, type Me } from "@/lib/api";
import { ConnectionForm } from "@/components/ConnectionForm";
import { ConnectionList } from "@/components/ConnectionList";
import { RdpViewer } from "@/components/RdpViewer";
import { ConfirmDialog } from "@/components/ConfirmDialog";
import { Alert, Button } from "@/components/ui";

type Session = { ticket: string; label: string };

export default function Home() {
  const router = useRouter();
  const [me, setMe] = useState<Me | null>(null);
  const [items, setItems] = useState<Connection[]>([]);
  const [editing, setEditing] = useState<Connection | null>(null);
  const [session, setSession] = useState<Session | null>(null);
  const [error, setError] = useState("");
  const [toDelete, setToDelete] = useState<Connection | null>(null);

  const load = useCallback(async () => {
    try {
      setMe(await api.me());
      setItems(await api.listConnections());
    } catch (e) {
      if (e instanceof ApiError && e.status === 401) router.replace("/login/");
      else setError("Could not reach the server.");
    }
  }, [router]);

  useEffect(() => {
    load();
  }, [load]);

  async function connectAdHoc(input: ConnectionInput) {
    const { ticket } = await api.ticketAdHoc(input);
    setSession({ ticket, label: input.name || `${input.host}:${input.port}` });
  }

  async function connectSaved(c: Connection) {
    setError("");
    try {
      const { ticket } = await api.ticketFor(c.id);
      setSession({ ticket, label: c.name });
    } catch (e) {
      setError(e instanceof ApiError ? e.message : "Network error.");
    }
  }

  async function save(input: ConnectionInput, id: string | null) {
    const body = { ...input, password: input.password === "" && id ? null : input.password };
    if (id) await api.updateConnection(id, body);
    else await api.createConnection(body);
    setEditing(null);
    setItems(await api.listConnections());
  }

  async function remove(c: Connection) {
    setToDelete(null);
    try {
      await api.deleteConnection(c.id);
      setItems(await api.listConnections());
    } catch (e) {
      setError(e instanceof ApiError ? e.message : "Network error.");
    }
  }

  async function logout() {
    await api.logout().catch(() => {});
    router.replace("/login/");
  }

  if (!me) {
    return <main className="p-8 text-sm text-muted">{error || "Loading…"}</main>;
  }

  return (
    <main className="mx-auto flex max-w-4xl flex-col gap-5 p-4 sm:p-8">
      <header className="flex items-center gap-3">
        <h1 className="text-xl font-semibold">WebRDP Gateway</h1>
        <span className="ms-auto text-sm text-ink-2">{me.username}</span>
        <Button variant="ghost" onClick={logout}>Sign out</Button>
      </header>
      {error && <Alert>{error}</Alert>}
      <ConnectionForm key={editing?.id ?? "new"} editing={editing} onConnect={connectAdHoc} onSave={save}
        onCancelEdit={() => setEditing(null)} />
      <ConnectionList items={items} onConnect={connectSaved} onEdit={setEditing} onDelete={setToDelete} />
      {toDelete && (
        <ConfirmDialog title={`Delete “${toDelete.name}”?`} body="Its saved password is destroyed. This cannot be undone."
          confirmLabel="Delete" onConfirm={() => remove(toDelete)} onCancel={() => setToDelete(null)} />
      )}
      {session && (
        <RdpViewer ticket={session.ticket} label={session.label} clipboard={{
            upload: me.features?.clipboard_upload ?? false, download: me.features?.clipboard_download ?? false,
          }}
          files={{
            upload: me.features?.file_upload ?? false, download: me.features?.file_download ?? false,
            maxMB: me.max_drive_mb ?? 0,
          }}
          onClose={() => setSession(null)} />
      )}
    </main>
  );
}
