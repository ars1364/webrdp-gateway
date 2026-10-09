# Security operations

## Secrets: storage, rotation, retention

| Secret | Lives in | Rotate |
|---|---|---|
| `APP_KEK` (seals RDP passwords + TOTP seeds) | `/opt/webrdp/.env` (0600), copy in Warden | 1) move the old key to `APP_KEK_PREVIOUS` + `APP_KEK_PREVIOUS_ID`; 2) set a new `APP_KEK` with a new `APP_KEK_ID`; 3) restart; 4) `docker compose run --rm api rekey` (one transaction); 5) remove `APP_KEK_PREVIOUS`, restart |
| `APP_DB_PASSWORD` | `.env` | `ALTER ROLE webrdp PASSWORD …`, update `.env`, restart api |
| `POSTGRES_SUPERUSER_PASSWORD` | `.env` | `ALTER ROLE postgres PASSWORD …` (only used for bootstrap) |
| User password / TOTP | DB (argon2id / sealed) | re-create the user (`create-admin`) |
| Cloudflare API token | `/etc/letsencrypt/cloudflare-webrdp.ini` (0600) | new token in Cloudflare, rewrite the file |
| TLS certificate | `/etc/letsencrypt` | certbot timer renews automatically; nginx reloads via the deploy hook |

| Data | Retention | Enforced by |
|---|---|---|
| Sessions | until expiry (`SESSION_TTL`, 12 h) | hourly reaper |
| Idempotency keys | 24 h (`IDEMPOTENCY_TTL`) | hourly reaper |
| Audit log | 180 days (`AUDIT_RETENTION_DAYS`) | hourly reaper; rows are append-only (DB trigger blocks UPDATE) |
| Transfer drive files | session lifetime; folder deleted on disconnect, all purged at API start; capped at `MAX_DRIVE_MB` (session closed when exceeded) | `drive.Manager` |
| Deleted connections | row kept (soft delete), sealed password destroyed on delete | `DeleteConnection` |

## CSRF

The session is a cookie, so CSRF applies. Two independent controls: `SameSite=Strict` on the
`__Host-` cookie, and an exact `Origin` match on every state-changing request and on the
WebSocket upgrade. No CORS headers are ever sent; preflights get `403 CORS_DISABLED`.

## Pen-test readiness

- Scope: `https://rdp.rmdashrf.com` (Cloudflare-fronted; origin answers Cloudflare only).
- Test account: create with `create-admin -role user`; delete afterwards.
- Expected hardening to verify: see the README security table and `docs/CRITERIA.md`.
- Report to the repository owner; do not file public issues for vulnerabilities.
