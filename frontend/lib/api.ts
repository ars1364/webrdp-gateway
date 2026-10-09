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
  has_password: boolean;
  updated_at: string;
};

export type Security = "any" | "nla" | "tls" | "rdp";

export type ConnectionInput = {
  name: string;
  host: string;
  port: number;
  username: string;
  domain: string;
  password?: string | null;
  security: Security;
  ignore_cert: boolean;
};

export type Me = { username: string; role: string; expires_at: string };

async function call<T>(method: string, path: string, body?: unknown): Promise<T> {
  const res = await fetch(`/api/v1${path}`, {
    method,
    credentials: "same-origin",
    headers: body === undefined ? {} : { "Content-Type": "application/json" },
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
  createConnection: (c: ConnectionInput) => call<Connection>("POST", "/connections", c),
  updateConnection: (id: string, c: ConnectionInput) => call<Connection>("PUT", `/connections/${id}`, c),
  deleteConnection: (id: string) => call<{ ok: boolean }>("DELETE", `/connections/${id}`),
  ticketFor: (connectionId: string) =>
    call<{ ticket: string }>("POST", "/tunnel/ticket", { connection_id: connectionId }),
  ticketAdHoc: (c: Omit<ConnectionInput, "name">) =>
    call<{ ticket: string }>("POST", "/tunnel/ticket", { ad_hoc: { ...c, name: "" } }),
};
