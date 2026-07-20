#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck source=lib/gcp-config.sh
source "$ROOT/scripts/lib/gcp-config.sh"

PROJECT="$(gcp_project || true)"
SECRET_ID="esxi-lab-wg0"
DEST="${WG_CONF:-/etc/wireguard/wg0.conf}"

if [[ -z "$PROJECT" ]]; then
	echo "Set GCP_PROJECT or run task infra:gcloud (tofu output gcp_project)" >&2
	exit 1
fi

if ! gcloud secrets describe "$SECRET_ID" --project="$PROJECT" &>/dev/null; then
	echo "Secret $SECRET_ID not found in $PROJECT — run task infra:gcloud, then task secrets:push" >&2
	exit 1
fi

if ! gcloud secrets versions list "$SECRET_ID" \
	--project="$PROJECT" \
	--filter="state=ENABLED" \
	--format="value(name)" 2>/dev/null | grep -q .; then
	echo "Secret $SECRET_ID has no versions — place wg0.conf at /etc/wireguard/wg0.conf (or \$WG_CONF) and run task secrets:push" >&2
	exit 1
fi

TMP="$(mktemp)"
trap 'rm -f "$TMP"' EXIT

gcloud secrets versions access latest \
	--secret="$SECRET_ID" \
	--project="$PROJECT" >"$TMP"

if ! grep -q '^\[Interface\]' "$TMP" || ! grep -q 'PrivateKey' "$TMP"; then
	echo "Invalid WireGuard config in projects/$PROJECT/secrets/$SECRET_ID" >&2
	exit 1
fi

changed=0
if [[ ! -f "$DEST" ]] || ! cmp -s "$TMP" "$DEST"; then
	echo "Installing WireGuard config to $DEST"
	sudo mkdir -p "$(dirname "$DEST")"
	sudo install -m 600 "$TMP" "$DEST"
	changed=1
else
	echo "WireGuard config already current at $DEST"
fi

if ip link show wg0 &>/dev/null; then
	if [[ "$changed" -eq 1 ]]; then
		echo "Reloading wg0 (config changed)..."
		sudo wg-quick down wg0 || true
		sudo wg-quick up wg0
	else
		echo "wg0 already up"
	fi
else
	echo "Bringing up wg0 from $DEST ..."
	sudo wg-quick up wg0
fi

wg show wg0 2>/dev/null | sed -n '1,8p' || true
