#!/usr/bin/env bash
# Gate: every *_handlers.go ships with a table-driven *_handlers_test.go.
set -euo pipefail
fail=0
for f in backend/internal/httpapi/*_handlers.go; do
  t="${f%.go}_test.go"
  if [ ! -f "$t" ]; then echo "::error file=$f::missing $t"; fail=1; continue; fi
  if ! grep -q 'tests := \[\]struct' "$t"; then echo "::error file=$t::$t has no table-driven test (tests := []struct{...})"; fail=1; fi
done
exit $fail
