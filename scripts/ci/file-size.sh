#!/usr/bin/env bash
# Gate: no source file over 300 lines (no god files). Split instead.
set -euo pipefail
MAX=${MAX_LINES:-300}
fail=0
while IFS= read -r f; do
  n=$(wc -l < "$f")
  if [ "$n" -gt "$MAX" ]; then echo "::error file=$f::$f is $n lines (max $MAX). Split it."; fail=1; fi
done < <(git ls-files '*.go' '*.ts' '*.tsx' '*.mjs' '*.sh' '*.sql' '*.py' '*.css' | grep -v -e '^frontend/next-env.d.ts$')
exit $fail
