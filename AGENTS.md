# webrdp-gateway: agent notes

Browser RDP gateway: Go API + guacd + Next.js static UI + Postgres. See README for architecture.

## Commands
- Backend: `cd backend && gofmt -l . && go vet ./... && go test ./...`
- Frontend: `cd frontend && npx tsc --noEmit && npm run build`
- Deploy (host): `/opt/webrdp/deploy.sh [sha]`. Images come from CI (ghcr.io/ars1364/webrdp-gateway-{api,web}).

## Invariants
- The user's explicit request wins over these notes; if one conflicts, do the rest, say which rule, offer the override.
- Never publish a container port on anything but 127.0.0.1. Postgres stays on the internal network.
- Every connections query is scoped `WHERE user_id = $1`. Never trust ids from the body for ownership.
- RDP targets go through `netguard.Resolve`; guacd gets the resolved IP, never the user's hostname.
- Credentials never go in URLs or logs. The tunnel uses single-use tickets.
- No secrets in the repo (it is public). Real values live in Warden; `.env.example` has placeholders.
- Max 300 lines per file; split instead.

## Definition of done
Checks pass → CI green → deployed → verified on https://rdp.rmdashrf.com with evidence (command output / screenshot).
