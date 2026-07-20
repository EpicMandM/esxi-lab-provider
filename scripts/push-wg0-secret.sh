#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck source=lib/gcp-config.sh
source "$ROOT/scripts/lib/gcp-config.sh"

PROJECT="$(gcp_project || true)"
SECRET_ID="esxi-lab-wg0"
WG_CONF="${WG_CONF:-}"

if [[ -z "$PROJECT" ]]; then
	echo "Set GCP_PROJECT or run task infra:gcloud (tofu output gcp_project)" >&2
	exit 1
fi

resolve_source() {
	if [[ -n "$WG_CONF" ]]; then
		if [[ -r "$WG_CONF" ]] || sudo test -f "$WG_CONF"; then
			echo "$WG_CONF"
			return
		fi
	fi
	if [[ -r /etc/wireguard/wg0.conf ]] || sudo test -f /etc/wireguard/wg0.conf; then
		echo /etc/wireguard/wg0.conf
		return
	fi
	if [[ -f "$ROOT/wg0.conf" ]]; then
		echo "$ROOT/wg0.conf"
		return
	fi
	return 1
}

read_source() {
	local src="$1"
	if [[ -r "$src" ]]; then
		cat "$src"
	else
		sudo cat "$src"
	fi
}

if ! gcloud secrets describe "$SECRET_ID" --project="$PROJECT" &>/dev/null; then
	echo "Secret $SECRET_ID not found in $PROJECT — run task infra:gcloud" >&2
	exit 1
fi

SOURCE=""
if ! SOURCE="$(resolve_source)"; then
	if gcloud secrets versions list "$SECRET_ID" \
		--project="$PROJECT" \
		--filter="state=ENABLED" \
		--format="value(name)" 2>/dev/null | grep -q .; then
		echo "No local wg0.conf — keeping existing Secret Manager version (projects/$PROJECT/secrets/$SECRET_ID)"
		exit 0
	fi
	echo "No local WireGuard config to upload (tried \$WG_CONF, /etc/wireguard/wg0.conf, $ROOT/wg0.conf)" >&2
	exit 1
fi

TMP="$(mktemp)"
trap 'rm -f "$TMP"' EXIT
read_source "$SOURCE" >"$TMP"

if ! grep -q '^\[Interface\]' "$TMP" || ! grep -q 'PrivateKey' "$TMP"; then
	echo "Invalid WireGuard config: $SOURCE" >&2
	exit 1
fi

LATEST="$(gcloud secrets versions access latest \
	--secret="$SECRET_ID" \
	--project="$PROJECT" 2>/dev/null || true)"
NEW="$(cat "$TMP")"
if [[ "$NEW" == "$LATEST" ]]; then
	echo "Secret Manager already up to date (projects/$PROJECT/secrets/$SECRET_ID)"
	exit 0
fi

gcloud secrets versions add "$SECRET_ID" \
	--project="$PROJECT" \
	--data-file="$TMP"

echo "Uploaded $SOURCE to projects/$PROJECT/secrets/$SECRET_ID"
