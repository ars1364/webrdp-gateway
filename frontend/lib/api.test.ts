import { afterEach, describe, expect, it, vi } from "vitest";
import { api, ApiError, toInput, type ConnectionInput } from "./api";

const conn: ConnectionInput = {
  name: "n", host: "203.0.113.1", port: 6579, username: "u", domain: "", password: "p",
  security: "any", ignore_cert: true,
};

function mockFetch(status: number, body: unknown) {
  const fn = vi.fn(async () => new Response(JSON.stringify(body), { status }));
  vi.stubGlobal("fetch", fn);
  return fn;
}

afterEach(() => vi.unstubAllGlobals());

describe("toInput", () => {
  it("drops fields the API would reject as unknown", () => {
    const dirty = { ...conn, id: "x", has_password: true, updated_at: "t" } as unknown as ConnectionInput;
    expect(Object.keys(toInput(dirty)).sort()).toEqual(
      ["domain", "host", "ignore_cert", "name", "password", "port", "security", "username"]);
  });
});

describe("call", () => {
  it.each([
    ["POST", () => api.createConnection(conn), true],
    ["DELETE", () => api.deleteConnection("id"), true],
    ["GET", () => api.listConnections(), false],
  ])("%s sends Idempotency-Key only on mutations", async (_m, run, want) => {
    const fn = mockFetch(200, { data: [] });
    await run();
    const init = (fn.mock.calls[0] as unknown as [string, RequestInit])[1];
    const key = (init.headers as Record<string, string>)["Idempotency-Key"];
    expect(Boolean(key)).toBe(want);
    if (want) expect(key).toMatch(/^[0-9a-f-]{36}$/);
  });

  it("maps the error envelope to ApiError with field details", async () => {
    mockFetch(422, { error: { code: "VALIDATION_FAILED", message: "bad", details: [{ field: "port", message: "1-65535" }] } });
    const err = await api.createConnection(conn).catch((e) => e);
    expect(err).toBeInstanceOf(ApiError);
    expect(err.status).toBe(422);
    expect(err.code).toBe("VALIDATION_FAILED");
    expect(err.details[0].field).toBe("port");
  });
});
