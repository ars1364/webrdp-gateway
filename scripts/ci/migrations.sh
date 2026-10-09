#!/usr/bin/env bash
# Gate: every migration has an up AND a down file; numbering is unique.
set -euo pipefail
cd backend/internal/db/migrations
fail=0
for up in *.up.sql; do
  [ -f "${up%.up.sql}.down.sql" ] || { echo "::error file=$up::no matching down migration"; fail=1; }
done
for down in *.down.sql; do
  [ -f "${down%.down.sql}.up.sql" ] || { echo "::error file=$down::down without up"; fail=1; }
done
dups=$(ls *.up.sql | cut -d_ -f1 | sort | uniq -d)
[ -z "$dups" ] || { echo "::error::duplicate migration numbers: $dups"; fail=1; }
ls *.sql | grep -qvE '^[0-9]{3}_[a-z0-9_]+\.(up|down)\.sql$' && { echo "::error::bad migration filename"; ls *.sql; fail=1; }
exit $fail
