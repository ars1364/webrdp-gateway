# Engineering criteria: status and enforcement

**Enforced** = a CI gate fails the build. **Tested** = covered by a test that runs in CI.
**Done** = implemented, but only review keeps it that way. **N/A** = does not apply to
this product, with the reason. Keep this file honest; CI checks the parts it can.

| # | Criterion | Status | Where / how |
|---|---|---|---|
| 1 | RBAC | Tested | roles `admin` / `user`; `/metrics` admin-only (`adminOnly`), `middleware_test.go` |
| 1 | Least privilege | Done | non-root distroless/unprivileged images, `cap_drop: ALL`, DB app role is not superuser, GITHUB_TOKEN `contents: read` by default |
| 1 | Creds isolated behind ports | Done | `core.Repository` port; secrets only cross as sealed bytes; Postgres on an `internal` network, no published port |
| 1 | Tenancy (repo-layer scoping) | Enforced + Tested | `scripts/ci/repo_scoping.py` fails any user-owned query without `user_id = $1`; `TestCrossUserIsolation`, `TestRepoScopingAndConstraints` |
| 2 | Versioning | Done | `/api/v1` |
| 2 | Correlation ID (UUID v4) | Tested | `correlationID` middleware; v1/garbage replaced; `TestCorrelationAndHeaders` |
| 2 | Host-header allowlist | Tested | `hostValidator` (`ALLOWED_HOSTS`) → 421 |
| 2 | CORS allowlist | Tested | same-origin only: no ACAO headers, preflight 403, Origin must equal `ALLOWED_ORIGIN` |
| 2 | Consistent error shape | Tested | `writeErr` envelope; tests assert `error.code` |
| 2 | Idempotency on mutations | Tested | `Idempotency-Key` (UUID v4) required; `idempotency_keys` table + TTL + reaper; `TestIdempotency`, `TestIdempotencyStore` |
| 2 | Bounded pagination | Tested | `per_page` ≤ 500, `meta.total`; `TestPagination` |
| 2 | UUID ids, no serials exposed, anti-enumeration | Tested | UUID PKs/FKs; `audit_log.id` serial is never serialized; bad/foreign ids → 404 (`TestBadIDsAreNotFound`, isolation tests); v1 UUIDs banned by `backend-rules.sh` |
| 3 | Rate limiting | Done | nginx `limit_req` (login 10/min, api 20/s) + app per-IP login and per-user ticket limiters + account lockout |
| 3 | CSRF | Tested | SameSite=Strict + Origin check (`docs/SECURITY.md`) |
| 3 | Secure headers / CSP / HSTS | Tested | API headers asserted in tests; web CSP with build-time script hashes; HSTS at nginx |
| 3 | SSRF | Tested | `netguard.Resolve` + resolved-IP pinning; `TestResolve`, `TestCreateTicket` |
| 3 | Secret rotation + retention | Tested | multi-key `seal` + `server rekey` (one txn), `TestRotation`; retention reaper (`docs/SECURITY.md`) |
| 4 | SBOM | Enforced | syft SPDX for repo + both images (CI artifact) and BuildKit SBOM/provenance attestations on images |
| 4 | Pinned deps | Enforced | go.sum, package-lock, Docker bases by digest, Actions by commit SHA; Dependabot keeps them fresh |
| 4 | Vuln scan / SAST | Enforced | govulncheck, npm audit, trivy fs + image (HIGH/CRITICAL), CodeQL (Go + TS). SonarQube: N/A (no server); CodeQL fills the SAST role |
| 4 | gitleaks | Enforced | full-history scan on every push/PR |
| 4 | Pen-test readiness | Done | `docs/SECURITY.md` scope + test-account procedure |
| 5 | Hexagonal ports/adapters | Done | `core` (types + ports) ← `httpapi` (driving) / `store`, `guac` (driven); handlers tested with fakes |
| 5 | No god files | Enforced | `scripts/ci/file-size.sh` (300 lines) |
| 5 | Panel owns state | Done | Postgres is the source of truth; guacd is a stateless protocol engine |
| 5 | LRO / async mutations | N/A | no long-running mutations: RDP sessions are streaming, not jobs |
| 5 | Reconcile-not-declare | N/A | no external resources are provisioned |
| 6 | Upstream timeouts | Done | guacd dial 5 s, handshake 20 s, idle 60 s, WS write 15 s, server read/header timeouts, DNS 5 s |
| 6 | Retries / circuit breaker | N/A | the only upstream is a user-chosen RDP host; a failed connect is surfaced (502), not retried behind the user |
| 6 | Concurrency / race safety | Tested | `go test -race`; TOTP step is an optimistic lock; idempotency claim is `ON CONFLICT DO NOTHING`; migrations take an advisory lock |
| 6 | Long-lived HTTP deadline + cap | Done | `MAX_TUNNEL_DURATION` (8 h), `MAX_TUNNELS` / `_PER_USER` |
| 6 | Graceful degradation | Done | guacd down → 502 with a clear message; SIGTERM → graceful shutdown |
| 7 | Migration safety | Enforced + Tested | ledger table + advisory lock, one txn per file, `.up/.down` pairs (`migrations.sh`), up→down→up in CI (`TestMigrationsReversible`) |
| 7 | DB constraints | Tested | CHECKs on port/role/security, unique live username, FKs |
| 7 | Append-only ledger | Tested | audit trigger blocks UPDATE |
| 7 | Metering / billing | N/A | no money flows in this product |
| 8 | Env config, stateless | Done | `config.Load` env-only; `.env.example`; sessions in DB |
| 8 | Air-gap readiness | Enforced | `scripts/ci/airgap.sh` vs `docs/AIRGAPPED.md`; no CDN, system fonts |
| 8 | Observability | Done | JSON logs with correlation_id, `/metrics` (admin), container healthchecks. Tracing: N/A for now (single service) |
| 8 | Feature flags | Done | `guac.Features` (clipboard, file transfer) surfaced in `/auth/me` |
| 9 | Design tokens / palette (light + dark) | Enforced | `lib/tokens.ts` + `globals.css`; ESLint bans inline hex |
| 9 | Component reuse | Done | `ui.tsx`, shared `Modal` shell (viewer + confirm) |
| 9 | Accessibility | Enforced | `eslint-plugin-jsx-a11y` strict; labelled inputs, `role=dialog`, focus restore |
| 9 | RTL / i18n | N/A (absent by design) | English only, no half-wired i18n; layout uses logical `ms-*` utilities so RTL can be added |
| 9 | Responsive | Done | single-column → grid at `sm` |
| 10 | Table-driven handler tests | Enforced | `scripts/ci/handler-tests.sh` |
| 10 | Critical-flow coverage | Tested | login, connection CRUD, ticket, tunnel auth, idempotency, isolation |
| 10 | Local CI gate | Done | `make ci` runs the same checks |
| 10 | Frontend Vitest | Tested | `lib/api.test.ts` |
| 11 | Doc currency | Enforced | `docs-drift.sh` (routes ↔ `docs/API.md`), `airgap.sh` (outbound ↔ `docs/AIRGAPPED.md`) |
| B | Secrets never in JSON | Enforced | `backend-rules.sh` (core types must tag secrets `json:"-"`), `TestConnectionNeverLeaksSecret` |
| B | Edge UUID validation | Tested | path ids validated before any query |
| B | Queue tables need expiry + reaper | Done | `idempotency_keys.expires_at` + reaper; tickets expire in memory |
| B | Live push on every mutation | N/A | no live-push channel in this product |
| B | Multi-step promote in one txn | Done | `rekey` reseal, each migration |
| F | Icon-only buttons need aria-label | Enforced | jsx-a11y |
| F | Post-mutation navigation via router | Enforced | ESLint bans `window.location` |
| F | DataTable truncate + title | Done | list rows truncate with `title` |
| CI | Self-hosted runner isolation | N/A | GitHub-hosted runners only; CODEOWNERS + least-privilege GITHUB_TOKEN |
