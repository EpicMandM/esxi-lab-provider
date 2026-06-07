#!/usr/bin/env python3
"""Terraform external data source: derive or fetch WireGuard server public key."""

from __future__ import annotations

import json
import re
import subprocess
import sys
import urllib.error
import urllib.request
from base64 import b64encode

KEY_RE = re.compile(r"^[A-Za-z0-9+/]{43}=$")


def emit(public_key: str, source: str) -> None:
    json.dump({"public_key": public_key, "source": source}, sys.stdout)
    sys.stdout.write("\n")


def valid_key(value: str) -> bool:
    return bool(value and KEY_RE.match(value))


def derive_from_private_key(key: str) -> str | None:
    try:
        proc = subprocess.run(
            ["wg", "pubkey"],
            input=key,
            text=True,
            capture_output=True,
            check=True,
        )
    except (subprocess.CalledProcessError, FileNotFoundError):
        return None
    pubkey = proc.stdout.strip()
    return pubkey if valid_key(pubkey) else None


def fetch_from_opnsense(
    opnsense_url: str,
    api_key: str,
    api_secret: str,
    server_name: str,
) -> str | None:
    if not (opnsense_url and api_key and api_secret):
        return None

    auth = b64encode(f"{api_key}:{api_secret}".encode()).decode()
    req = urllib.request.Request(
        f"{opnsense_url.rstrip('/')}/api/wireguard/server/search_server",
        data=b"{}",
        headers={
            "Content-Type": "application/json",
            "Authorization": f"Basic {auth}",
        },
        method="POST",
    )
    try:
        with urllib.request.urlopen(req, timeout=15) as resp:
            payload = json.load(resp)
    except (urllib.error.URLError, TimeoutError, json.JSONDecodeError):
        return None

    rows = payload.get("rows") or []
    if server_name:
        rows = [row for row in rows if row.get("name") == server_name]
    for row in rows:
        pubkey = (row.get("pubkey") or row.get("public_key") or "").strip()
        if valid_key(pubkey):
            return pubkey
    return None


def main() -> int:
    query = json.load(sys.stdin)
    private_key = (query.get("private_key") or "").strip()
    opnsense_url = (query.get("opnsense_url") or "").strip()
    opnsense_api_key = (query.get("opnsense_api_key") or "").strip()
    opnsense_api_secret = (query.get("opnsense_api_secret") or "").strip()
    server_name = (query.get("server_name") or "").strip()

    if private_key:
        derived = derive_from_private_key(private_key)
        if derived:
            emit(derived, "private_key")
            return 0

    fetched = fetch_from_opnsense(
        opnsense_url,
        opnsense_api_key,
        opnsense_api_secret,
        server_name,
    )
    if fetched:
        emit(fetched, "opnsense")
        return 0

    print(
        "Set WIREGUARD_SERVER_PRIVATE_KEY in Secret Manager or provide OPNsense API access.",
        file=sys.stderr,
    )
    return 1


if __name__ == "__main__":
    sys.exit(main())
