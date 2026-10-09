# Air-gap readiness

Everything the running system or its build reaches over the network. CI
(`scripts/ci/airgap.sh`) fails when code or deploy config references a host
or image that is not listed here. Update this file **in the same PR** as any
outbound-network change.

## Runtime (the deployed stack)

| Dependency | Why | Air-gapped alternative |
|---|---|---|
| RDP targets (user-supplied IP:port) | the product | targets inside the enclave |
| none else | the browser bundle is static, system fonts, no CDN, no telemetry | n/a |

## Images (mirror these, pinned by digest in compose / Dockerfiles)

- `postgres` (16-alpine)
- `guacamole/guacd` (1.5.5)
- `golang` (1.27-alpine, build stage)
- `gcr.io/distroless/static-debian12` (nonroot, runtime)
- `node` (22-alpine, build stage)
- `nginxinc/nginx-unprivileged` (1.29-alpine)
- `ghcr.io/ars1364/webrdp-gateway-api`, `ghcr.io/ars1364/webrdp-gateway-web` (this repo's CI)

Set the registry prefix to your mirror (for example `docker.cloudinative.com/`) in compose and the Dockerfiles.

## Host bootstrap (`deploy/host-setup.sh`)

| Host | Why | Air-gapped alternative |
|---|---|---|
| `www.cloudflare.com` | `ips-v4` / `ips-v6` lists for the origin allowlist | ship the lists, skip if not behind Cloudflare |
| Let's Encrypt + Cloudflare DNS API (via certbot) | TLS certificate | internal CA certificate in `/etc/letsencrypt/live/<domain>/` |
| Ubuntu apt mirrors | nginx, docker, certbot | local apt mirror (Nexus `apt` proxy) |

## Build (CI only)

GitHub Actions, Go module proxy, npm registry, Docker Hub, GHCR, gcr.io. Pre-build images
outside the enclave and import them (`docker save | docker load`) or push to the mirror.
