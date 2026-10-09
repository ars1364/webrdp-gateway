"use client";

import { useState, type FormEvent } from "react";
import { ApiError, type Connection, type ConnectionInput, type FieldError, type Security } from "@/lib/api";
import { Alert, Button, Card, Field } from "./ui";

type Props = {
  editing: Connection | null;
  onConnect: (input: ConnectionInput) => Promise<void>;
  onSave: (input: ConnectionInput, id: string | null) => Promise<void>;
  onCancelEdit: () => void;
};

const blank: ConnectionInput = {
  name: "", host: "", port: 3389, username: "", domain: "", password: "", security: "any", ignore_cert: true,
};

function fromConn(c: Connection): ConnectionInput {
  return { ...c, password: null };
}

// One form for both jobs: "Connect now" (ad-hoc, nothing stored) and
// "Save" (stored, password encrypted server-side).
export function ConnectionForm({ editing, onConnect, onSave, onCancelEdit }: Props) {
  const [v, setV] = useState<ConnectionInput>(editing ? fromConn(editing) : blank);
  const [errors, setErrors] = useState<FieldError[]>([]);
  const [msg, setMsg] = useState("");
  const [busy, setBusy] = useState(false);
  const set = <K extends keyof ConnectionInput>(k: K, val: ConnectionInput[K]) => setV((p) => ({ ...p, [k]: val }));
  const err = (f: string) => errors.find((e) => e.field === f)?.message;

  async function run(fn: () => Promise<void>) {
    setBusy(true);
    setErrors([]);
    setMsg("");
    try {
      await fn();
    } catch (e) {
      if (e instanceof ApiError) {
        setErrors(e.details);
        setMsg(e.message);
      } else setMsg("Network error.");
    } finally {
      setBusy(false);
    }
  }

  function connect(e: FormEvent) {
    e.preventDefault();
    run(() => onConnect(v));
  }

  return (
    <Card>
      <h2 className="mb-4 text-base font-semibold">{editing ? `Edit “${editing.name}”` : "Connect to a remote desktop"}</h2>
      <form onSubmit={connect} className="grid grid-cols-1 gap-4 sm:grid-cols-6">
        <Field className="sm:col-span-4" label="IP address or hostname" required placeholder="203.0.113.10"
          value={v.host} error={err("host")} onChange={(e) => set("host", e.target.value)} />
        <Field className="sm:col-span-2" label="Port" type="number" min={1} max={65535} required
          value={v.port} error={err("port")} onChange={(e) => set("port", Number(e.target.value))} />
        <Field className="sm:col-span-3" label="Username" autoComplete="off" value={v.username}
          error={err("username")} onChange={(e) => set("username", e.target.value)} />
        <Field className="sm:col-span-3" label="Password" type="password" autoComplete="new-password"
          placeholder={editing?.has_password ? "•••••• (unchanged)" : ""} value={v.password ?? ""}
          error={err("password")} onChange={(e) => set("password", e.target.value)} />
        <Field className="sm:col-span-2" label="Domain (optional)" value={v.domain}
          onChange={(e) => set("domain", e.target.value)} />
        <label className="flex flex-col gap-1 text-sm sm:col-span-2">
          <span className="font-medium text-ink-2">Security</span>
          <select className="h-10 rounded-lg border border-line bg-white px-3" value={v.security}
            onChange={(e) => set("security", e.target.value as Security)}>
            <option value="any">Auto (negotiate)</option>
            <option value="nla">NLA</option>
            <option value="tls">TLS</option>
            <option value="rdp">RDP (legacy)</option>
          </select>
        </label>
        <label className="flex items-end gap-2 pb-2 text-sm sm:col-span-2">
          <input type="checkbox" className="size-4 accent-primary" checked={v.ignore_cert}
            onChange={(e) => set("ignore_cert", e.target.checked)} />
          <span>Accept self-signed certificate</span>
        </label>
        <Field className="sm:col-span-4" label="Name (to save it)" placeholder="Office PC" value={v.name}
          error={err("name")} onChange={(e) => set("name", e.target.value)} />
        <div className="flex flex-wrap items-end gap-2 sm:col-span-2">
          <Button type="submit" disabled={busy}>Connect</Button>
          <Button type="button" variant="ghost" disabled={busy}
            onClick={() => run(() => onSave(v, editing?.id ?? null))}>{editing ? "Update" : "Save"}</Button>
          {editing && <Button type="button" variant="ghost" onClick={onCancelEdit}>Cancel</Button>}
        </div>
        {msg && <div className="sm:col-span-6"><Alert>{msg}</Alert></div>}
      </form>
    </Card>
  );
}
