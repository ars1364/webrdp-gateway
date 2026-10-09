#!/usr/bin/env python3
"""Gate: every SQL string in the store adapter that touches a user-owned
table filters by `user_id = $1`. A function that is deliberately
system-wide (retention reaper, KEK rotation) must say so with a
`// unscoped: <reason>` line in its doc comment."""
import pathlib
import re
import sys

OWNED = re.compile(r"\b(connections|audit_log|idempotency_keys|sessions)\b")
SCOPED = re.compile(r"user_id\s*=\s*\$1")
fail = False
for path in sorted(pathlib.Path("backend/internal/store").glob("*.go")):
    if path.name.endswith("_test.go"):
        continue
    src = path.read_text()
    # split into functions with their doc comments
    for m in re.finditer(r"((?:^//.*\n)*)^func [^\n]*\{\n(.*?)^\}", src, re.S | re.M):
        doc, body = m.group(1), m.group(2)
        name = re.search(r"func (?:\([^)]*\) )?(\w+)", m.group(0)[len(doc):]).group(1)
        if "unscoped:" in doc:
            continue
        for sql in re.findall(r"`([^`]*)`", body):
            if not re.search(r"\b(SELECT|UPDATE|DELETE|INSERT)\b", sql):
                continue
            if not OWNED.search(sql):
                continue
            if sql.lstrip().upper().startswith("INSERT"):
                continue  # inserts take user_id as a column value
            if "token_hash" in sql:
                continue  # sessions are looked up by their secret hash
            if not SCOPED.search(sql):
                print(f"::error file={path}::{name}: query on a user-owned table without `user_id = $1`:"
                      f" {' '.join(sql.split())[:120]}")
                fail = True
sys.exit(1 if fail else 0)
