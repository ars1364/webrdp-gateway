// Same-origin API client. The session lives in an HttpOnly cookie, so no
// token ever touches JavaScript.

export type FieldError = { field: string; message: string };

export class ApiError extends Error {
  constructor(
    public status: number,
    public code: string,
    message: string,
    public details: FieldError[] = [],
  ) {
    super(message);
  }
}

export type Connection = {
  id: string;
  name: string;
  host: string;
  port: number;
  username: string;
  domain: string;
  security: Security;
  ignore_cert: boolean;
  quality: Quality;
  has_password: boolean;
  updated_at: string;
};

export type Security = "any" | "nla" | "tls" | "rdp";

// Bandwidth profile: "low" = 16-bit colour, no wallpaper/theming/smoothing.
export type Quality = "high" | "balanced" | "low";

export type ConnectionInput = {
  name: string;
  host: string;
  port: number;
  username: string;
  domain: string;
  password?: string | null;
  security: Security;
  ignore_cert: boolean;
  quality: Quality;
};

export type Me = {
  username: string;
  role: string;
  expires_at: string;
  max_drive_mb: number;
  features: { clipboard_upload: boolean; clipboard_download: boolean; file_upload: boolean; file_download: boolean };
};

// Send exactly the DTO fields: the API rejects unknown keys (mass-assignment guard).
export function toInput(c: ConnectionInput): ConnectionInput {
  const { name, host, port, username, domain, password, security, ignore_cert, quality } = c;
  return { name, host, port, username, domain, password, security, ignore_cert, quality };
}

async function call<T>(method: string, path: string, body?: unknown): Promise<T> {
  const headers: Record<string, string> = {};
  if (body !== undefined) headers["Content-Type"] = "application/json";
  // Every mutation carries a fresh UUID v4 so a retried request is deduped.
  if (method !== "GET") headers["Idempotency-Key"] = crypto.randomUUID();
  const res = await fetch(`/api/v1${path}`, {
    method,
    credentials: "same-origin",
    headers,
    body: body === undefined ? undefined : JSON.stringify(body),
  });
  const json = await res.json().catch(() => ({}));
  if (!res.ok) {
    const e = json?.error ?? {};
    throw new ApiError(res.status, e.code ?? "HTTP_" + res.status, e.message ?? res.statusText, e.details ?? []);
  }
  return json.data as T;
}

export const api = {
  login: (username: string, password: string, totp: string) =>
    call<Me>("POST", "/auth/login", { username, password, totp }),
  logout: () => call<{ ok: boolean }>("POST", "/auth/logout"),
  me: () => call<Me>("GET", "/auth/me"),
  listConnections: () => call<Connection[]>("GET", "/connections"),
  createConnection: (c: ConnectionInput) => call<Connection>("POST", "/connections", toInput(c)),
  updateConnection: (id: string, c: ConnectionInput) => call<Connection>("PUT", `/connections/${id}`, toInput(c)),
  deleteConnection: (id: string) => call<{ ok: boolean }>("DELETE", `/connections/${id}`),
  ticketFor: (connectionId: string) =>
    call<{ ticket: string }>("POST", "/tunnel/ticket", { connection_id: connectionId }),
  ticketAdHoc: (c: ConnectionInput) =>
    call<{ ticket: string }>("POST", "/tunnel/ticket", { ad_hoc: toInput(c) }),
};
