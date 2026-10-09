"use client";

import { useRouter } from "next/navigation";
import { useState, type FormEvent } from "react";
import { api, ApiError } from "@/lib/api";
import { Alert, Button, Card, Field } from "@/components/ui";

export default function LoginPage() {
  const router = useRouter();
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [totp, setTotp] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  async function submit(e: FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError("");
    try {
      await api.login(username, password, totp);
      router.replace("/");
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Network error.");
      setTotp("");
    } finally {
      setBusy(false);
    }
  }

  return (
    <main className="flex min-h-screen items-center justify-center p-4">
      <Card className="w-full max-w-sm">
        <h1 className="text-xl font-semibold">WebRDP Gateway</h1>
        <p className="mb-5 mt-1 text-sm text-ink-2">Sign in with your password and authenticator code.</p>
        <form onSubmit={submit} className="flex flex-col gap-4">
          <Field label="Username" autoComplete="username" required value={username}
            onChange={(e) => setUsername(e.target.value)} />
          <Field label="Password" type="password" autoComplete="current-password" required value={password}
            onChange={(e) => setPassword(e.target.value)} />
          <Field label="Authenticator code" inputMode="numeric" autoComplete="one-time-code" pattern="[0-9]{6}"
            maxLength={6} required value={totp} onChange={(e) => setTotp(e.target.value.replace(/\D/g, ""))} />
          {error && <Alert>{error}</Alert>}
          <Button type="submit" disabled={busy}>{busy ? "Signing in…" : "Sign in"}</Button>
        </form>
      </Card>
    </main>
  );
}
