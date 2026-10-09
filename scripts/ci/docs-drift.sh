#!/usr/bin/env bash
# Gate: docs/API.md (capability catalogue) lists exactly the routes the
# server registers. Adding/removing a route without the doc fails CI.
set -euo pipefail
code=$(grep -ohE 'mux\.Handle(Func)?\("[A-Z]+ [^"]+"' backend/internal/httpapi/server.go \
  | sed -E 's/.*\("([^"]+)"/\1/' | sort -u)
doc=$(grep -oE '^\| `[A-Z]+ /[^`]*`' docs/API.md | sed -E 's/^\| `([^`]+)`/\1/' | sort -u)
if [ "$code" != "$doc" ]; then
  echo "::error file=docs/API.md::route catalogue drift (< code only, > doc only):"
  diff <(echo "$code") <(echo "$doc") || true
  exit 1
fi
echo "docs/API.md matches $(echo "$code" | wc -l) routes"
