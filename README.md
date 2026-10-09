# webrdp-gateway

Remote desktop (Windows RDP) in the browser. Type an IP and a **custom port**,
or pick a saved connection, and the desktop opens in a browser tab. No client install needed.

```
browser ──HTTPS/WSS──> Cloudflare ──> nginx (host, :443) ──> web  (static Next.js)
                                                     └──> api  (Go) ──> guacd ──RDP──> target IP:port
                                                               └──> Postgres (internal network)
```

- **Frontend:** Next.js 16 static export + React 19 + Tailwind 4, `guacamole-common-js` for the canvas.
- **Backend:** Go (stdlib `net/http`), speaks the Guacamole protocol to `guacd` and bridges it to a WebSocket.
- **RDP engine:** official `guacamole/guacd` image (FreeRDP inside). Our API and UI remain the source of truth; guacd is only the protocol engine.
- **Database:** PostgreSQL 16: users, sessions, saved connections, audit log.

## Security model

| Layer | Control |
|---|---|
| Edge | Cloudflare proxy; origin `ufw` allows 443 from Cloudflare ranges only, nginx returns 444 to anything else; unknown SNI gets no certificate |
| TLS | Let's Encrypt via DNS-01 (Cloudflare), TLS 1.2/1.3, HSTS |
| Login | username + password (argon2id) + TOTP (RFC 6238, replay-protected); 5 failures lock the account for 15 min; per-IP rate limits in nginx and the app |
| Session | random 256-bit token in an `__Host-` cookie, `HttpOnly; Secure; SameSite=Strict`; only its SHA-256 is stored |
| CSRF | SameSite=Strict + exact `Origin` check on every state-changing request and on the WebSocket upgrade |
| Tunnel | needs the session cookie **and** a single-use 30 s ticket; credentials never appear in URLs |
| Targets | any public IP/host + port the user types; loopback, private, link-local/metadata, CGNAT and reserved ranges are refused (SSRF guard), and guacd is handed the resolved IP so DNS rebinding can't swap it |
| Secrets at rest | saved RDP passwords and TOTP seeds sealed with AES-256-GCM (`APP_KEK`), bound to their row id |
| Containers | read-only root FS, `cap_drop: ALL`, `no-new-privileges`, memory limits, ports bound to `127.0.0.1` only |
| Database | no published port, `internal` Docker network, scram-sha-256, app uses a non-superuser role that owns only its DB |
| Frontend | strict CSP with per-build script hashes (no `unsafe-inline` scripts), `frame-ancestors 'none'`, no-referrer |
| Audit | logins, connection changes, every RDP open/close/failure with source IP |

## Engineering gates

Every push runs: mechanical guards (300-line files, table-driven handler tests, migration
up/down pairs, route catalogue drift, air-gap inventory, repo-layer scoping, secret-field and
SQL rules), gitleaks, gofmt/vet/`go test -race`, govulncheck, Postgres integration tests
(migrations up→down→up, cross-user isolation, append-only audit), tsc/ESLint (jsx-a11y,
no inline hex)/Vitest/build/npm audit, trivy (fs + images), SBOMs (syft + BuildKit
attestations) and CodeQL. Actions and base images are pinned by SHA/digest. Run the same
locally with `make ci`. Status of every criterion: [`docs/CRITERIA.md`](docs/CRITERIA.md).

Docs: [API catalogue](docs/API.md) · [Security ops](docs/SECURITY.md) · [Air-gap](docs/AIRGAPPED.md)

## Roadmap

1. ✅ RDP (keyboard, mouse, dynamic resolution, Ctrl+Alt+Del, quality profiles)
2. ✅ Clipboard, text both ways (auto-sync + manual panel; `FEATURE_CLIPBOARD`)
3. ⏳ File transfer (`FEATURE_FILE_TRANSFER`, guacd drive + upload/download)

## Run it

```bash
# on the host, once
sudo DOMAIN=rdp.example.com EMAIL=you@example.com CF_API_TOKEN=... deploy/host-setup.sh
sudo vi /opt/webrdp/.env            # secrets: openssl rand -base64 32
sudo deploy/deploy.sh               # pulls ghcr.io images, starts the stack

# first admin (prints password + TOTP seed ONCE, so store them in your secret manager)
sudo docker compose -f /opt/webrdp/docker-compose.yml run --rm api create-admin -u admin
python3 tools/totp.py <TOTP_SEED>   # current 6-digit code
```

## Develop

```bash
cd backend && go test ./... && go run ./cmd/server     # needs DATABASE_URL, APP_KEK, ALLOWED_ORIGIN
cd frontend && npm ci && npm run dev                   # http://localhost:3333
```
