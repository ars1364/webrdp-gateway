#!/usr/bin/env bash
# Gate: nginx discards inherited add_header lines in any location that sets
# its own, silently dropping CSP/HSTS. Forbid add_header inside location{}.
set -euo pipefail
fail=0
for f in frontend/nginx.conf deploy/nginx/*.tmpl; do
  bad=$(awk '/location[^{]*\{/{d++} d>0 && /add_header/{print FILENAME":"NR": "$0} /\}/{if(d>0)d--}' "$f")
  [ -z "$bad" ] || { echo "::error file=$f::add_header inside a location drops server-level security headers:"; echo "$bad"; fail=1; }
done
exit $fail
