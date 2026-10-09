#!/usr/bin/env bash
# Gate: mechanical backend safety rules.
set -euo pipefail
fail=0
err() { echo "::error::$*"; fail=1; }
src=$(git ls-files 'backend/**/*.go' ':!**/*_test.go')

# 1. Domain/response types never serialize secrets (use json:"-" + has_X).
grep -nE '^\s+[A-Za-z]*(Password|Secret|Token|Enc|Hash|KEK)[A-Za-z]*\s+[^ ]+\s+`json:"[^-]' \
  backend/internal/core/*.go | grep -v 'Has[A-Z]' && err 'secret-like field serialized to JSON; tag it json:"-" and expose has_X instead'

# 2. No SQL built with fmt.Sprintf / string concat of user input.
grep -nE 'Sprintf\(\s*`?\s*(SELECT|INSERT|UPDATE|DELETE)' $src && err 'SQL via Sprintf: use $N parameters'

# 3. Repo-layer scoping (see repo_scoping.py).
python3 scripts/ci/repo_scoping.py || fail=1

# 4. No UUID v1 (leaks MAC + time); no math/rand for ids or tokens.
grep -nE 'uuid\.NewUUID\(|NewV1\(' $src && err 'UUID v1 is banned; use v4/v7'
grep -lE '"math/rand' $src && err 'math/rand in production code; use crypto/rand'

# 5. Outbound dials have timeouts.
grep -nE 'net\.Dial\(|http\.Get\(' $src | grep -v healthcheck && err 'dial without timeout; use net.Dialer{Timeout} / http.Client{Timeout}'
exit $fail
