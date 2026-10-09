#!/usr/bin/env bash
# Gate: every outbound host referenced by runtime code/config must be listed
# in docs/AIRGAPPED.md (so an air-gapped install knows what to mirror).
set -euo pipefail
hosts=$(git ls-files 'backend/**/*.go' 'frontend/app/**' 'frontend/components/**' 'frontend/lib/**' \
    'deploy/**' 'backend/Dockerfile' 'frontend/Dockerfile' ':!**/*_test.go' ':!**/*.test.ts' \
  | xargs grep -ohE 'https?://[a-zA-Z0-9.-]+\.[a-z]{2,}' 2>/dev/null \
  | sed -E 's#https?://##' | sort -u \
  | grep -vE '^(rdp\.example\.com|example\.com|localhost)$' || true)
images=$(git ls-files 'deploy/docker-compose.yml' 'backend/Dockerfile' 'frontend/Dockerfile' \
  | xargs grep -ohE '(FROM|image:) +[a-z0-9./-]+' | awk '{print $2}' | sort -u)
fail=0
for h in $hosts $images; do
  grep -qF "$h" docs/AIRGAPPED.md || { echo "::error file=docs/AIRGAPPED.md::outbound dependency not documented: $h"; fail=1; }
done
exit $fail
