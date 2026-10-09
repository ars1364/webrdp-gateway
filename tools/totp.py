#!/usr/bin/env python3
"""Print the current 6-digit TOTP code for a base32 seed (RFC 6238, SHA-1, 30 s).

Usage:
  python3 tools/totp.py SEED
  TOTP_SECRET=SEED python3 tools/totp.py
  python3 tools/totp.py SEED --watch      # refresh every second

The seed is never stored by this script. Keep it in a secret manager.
"""
import base64
import hashlib
import hmac
import os
import struct
import sys
import time


def totp(seed: str, at: float | None = None, step: int = 30) -> str:
    seed = seed.strip().replace(" ", "").upper()
    key = base64.b32decode(seed + "=" * (-len(seed) % 8))
    counter = int((time.time() if at is None else at) // step)
    mac = hmac.new(key, struct.pack(">Q", counter), hashlib.sha1).digest()
    off = mac[-1] & 0x0F
    code = (struct.unpack(">I", mac[off:off + 4])[0] & 0x7FFFFFFF) % 1_000_000
    return f"{code:06d}"


def main() -> int:
    args = [a for a in sys.argv[1:] if not a.startswith("--")]
    seed = args[0] if args else os.environ.get("TOTP_SECRET", "")
    if not seed:
        print(__doc__, file=sys.stderr)
        return 2
    # RFC 6238 self-check so a broken install never prints wrong codes.
    rfc = base64.b32encode(b"12345678901234567890").decode()
    assert totp(rfc, 59) == "287082", "self-test failed"
    if "--watch" not in sys.argv:
        print(totp(seed))
        return 0
    try:
        while True:
            left = 30 - int(time.time()) % 30
            print(f"\r{totp(seed)}  ({left:2d}s)", end="", flush=True)
            time.sleep(1)
    except KeyboardInterrupt:
        print()
    return 0


if __name__ == "__main__":
    sys.exit(main())
