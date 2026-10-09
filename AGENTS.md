# webrdp-gateway: agent notes

Browser RDP gateway: Go API + guacd + Next.js static UI + Postgres. See README for architecture.

## Commands
- Local gate (same as CI): `make ci`; Postgres tests: `TEST_DATABASE_URL=... make integration`
- Backend: `cd backend && gofmt -l . && go vet ./... && go test -race ./...`
- Frontend: `cd frontend && npx tsc --noEmit && npm run lint && npm test && npm run build`
- Deploy (host): `/opt/webrdp/deploy.sh [sha]`. Images come from CI (ghcr.io/ars1364/webrdp-gateway-{api,web}).

## Invariants
- The user's explicit request wins over these notes; if one conflicts, do the rest, say which rule, offer the override.
- Never publish a container port on anything but 127.0.0.1. Postgres stays on the internal network.
- Every connections query is scoped `WHERE user_id = $1`. Never trust ids from the body for ownership.
- RDP targets go through `netguard.Resolve`; guacd gets the resolved IP, never the user's hostname.
- Credentials never go in URLs or logs. The tunnel uses single-use tickets.
- No secrets in the repo (it is public). Real values live in Warden; `.env.example` has placeholders.
- Max 300 lines per file; split instead.
- New route → add it to `docs/API.md`. New outbound host/image → `docs/AIRGAPPED.md`. New handler file → table-driven test. New migration → `.up.sql` + `.down.sql`. CI enforces all four.
- Handlers depend on `core` ports only; SQL lives in `store` and filters `user_id = $1` (or documents `// unscoped:`).
- Keep `docs/CRITERIA.md` honest when a criterion's status changes.

## Definition of done
Checks pass → CI green → deployed → verified on https://rdp.rmdashrf.com with evidence (command output / screenshot).
