#!/usr/bin/env bash
# Pull and (re)start the stack on the host: ./deploy.sh [image-tag]
set -euo pipefail
cd /opt/webrdp
[ -n "${1:-}" ] && sed -i "s/^IMAGE_TAG=.*/IMAGE_TAG=$1/" .env
docker compose pull --quiet
docker compose up -d --remove-orphans
docker image prune -f >/dev/null
docker compose ps
