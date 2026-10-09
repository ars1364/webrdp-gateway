# API: capability catalogue

Base path `/api/v1` (URL-versioned). JSON in, JSON out. CI (`scripts/ci/docs-drift.sh`)
fails if this table and `backend/internal/httpapi/server.go` disagree.

**Every response** carries `X-Correlation-ID` (UUID v4: your valid v4 is echoed, otherwise one is minted).
**Errors** always use `{"error":{"code":"SCREAMING_SNAKE","message":"…","details":[{"field","message"}]}}`.
**Mutations** (`POST/PUT/DELETE /connections…`) require `Idempotency-Key: <uuid v4>`; a replay returns the
stored response with `Idempotent-Replayed: true`, the same key on a different body is `422 IDEMPOTENCY_KEY_REUSED`.
**Lists** take `?page=1&per_page=50` (max 500) and return `meta.total/page/per_page`.
**Identifiers** are UUID v4 everywhere; no database serial is ever exposed.

| Route | Auth | Purpose |
|---|---|---|
| `GET /healthz` | none | Liveness for the container probe (not exposed by nginx) |
| `GET /metrics` | session, role `admin` | Prometheus text: requests, active RDP sessions, login failures |
| `POST /api/v1/auth/login` | none, rate limited | `{username,password,totp}` → `__Host-` session cookie |
| `POST /api/v1/auth/logout` | session | Revoke the current session |
| `GET /api/v1/auth/me` | session | Current user, role, session expiry, feature flags |
| `GET /api/v1/connections` | session | Paginated saved connections (password never returned; `has_password`) |
| `POST /api/v1/connections` | session + Idempotency-Key | Create a saved connection (`quality`: `high` / `balanced` / `low` bandwidth profile) |
| `PUT /api/v1/connections/{id}` | session + Idempotency-Key | Replace one; `password: null` keeps the stored one |
| `DELETE /api/v1/connections/{id}` | session + Idempotency-Key | Soft-delete and destroy its sealed password |
| `POST /api/v1/tunnel/ticket` | session, rate limited | `{connection_id}` or `{ad_hoc:{host,port,…}}` → single-use 30 s ticket (SSRF-checked) |
| `GET /api/v1/tunnel` | session + `?ticket=` | WebSocket (`guacamole` subprotocol) bridged to guacd → RDP |
| `GET /api/v1/audit` | session | Paginated audit trail for the current user |
| `GET /api/v1/recordings` | session | Paginated server-side session recordings of the current user |
| `GET /api/v1/recordings/{id}/file` | session (owner) | The guacd recording file, for the in-browser player (audited as `recording.view`) |
| `DELETE /api/v1/recordings/{id}` | session, role `admin` + Idempotency-Key | Delete a recording and its file (audited) |

Error codes: `UNAUTHENTICATED` 401 · `FORBIDDEN` 403 · `BAD_ORIGIN` 403 · `CORS_DISABLED` 403 ·
`BAD_HOST` 421 · `NOT_FOUND` 404 · `VALIDATION_FAILED` 422 · `TARGET_REJECTED` 422 ·
`IDEMPOTENCY_KEY_REQUIRED` 400 · `IDEMPOTENCY_KEY_REUSED` 422 · `IDEMPOTENCY_IN_PROGRESS` 409 ·
`INVALID_CREDENTIALS` 401 · `ACCOUNT_LOCKED` 423 · `RATE_LIMITED` 429 · `TOO_MANY_SESSIONS` 429 ·
`BAD_TICKET` 403 · `RDP_CONNECT_FAILED` 502 · `INTERNAL` 500.
