#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck source=lib/gcp-config.sh
source "$ROOT/scripts/lib/gcp-config.sh"

ENV_FILE="${ENV_FILE:-$ROOT/secrets.env}"
PROJECT="$(gcp_project || true)"
SECRET_ID="$(lab_env_secret_id || true)"

if [[ -z "$PROJECT" ]]; then
	echo "Set GCP_PROJECT or gcp_project in $(lab_config_file)" >&2
	exit 1
fi
if [[ -z "$SECRET_ID" ]]; then
	echo "Set LAB_ENV_SECRET_ID or lab_env_secret_id in $(lab_config_file)" >&2
	exit 1
fi

SECRET_KEYS=(
	ESXI_PASSWORD
	OPNSENSE_API_KEY
	OPNSENSE_API_SECRET
	WIREGUARD_SERVER_PRIVATE_KEY
	SMTP_PASSWORD
)

secret_value() {
	local key="$1"
	local line=""
	if [[ -f "$ENV_FILE" ]]; then
		line="$(grep -E "^${key}=" "$ENV_FILE" | head -1 || true)"
		if [[ "$line" =~ ^[A-Z_]+=.+$ ]]; then
			echo "${line#*=}"
			return
		fi
	fi
	if [[ -n "${!key:-}" ]]; then
		echo "${!key}"
	fi
}

if ! gcloud secrets describe "$SECRET_ID" --project="$PROJECT" &>/dev/null; then
	echo "Secret $SECRET_ID not found in $PROJECT — run task infra:gcloud" >&2
	exit 1
fi

TMP="$(mktemp)"
trap 'rm -f "$TMP"' EXIT

{
	echo "# Uploaded $(date -u +%Y-%m-%dT%H:%M:%SZ)"
	for key in "${SECRET_KEYS[@]}"; do
		val="$(secret_value "$key")"
		if [[ -n "$val" ]]; then
			echo "${key}=${val}"
		fi
	done
} >"$TMP"

if ! grep -qE '^[A-Z_]+=.+' "$TMP"; then
	echo "No credentials in $ENV_FILE or env (need: ${SECRET_KEYS[*]})" >&2
	exit 1
fi

gcloud secrets versions add "$SECRET_ID" \
	--project="$PROJECT" \
	--data-file="$TMP"

echo "Uploaded to projects/$PROJECT/secrets/$SECRET_ID"
