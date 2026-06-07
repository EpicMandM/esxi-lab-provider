#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck source=lib/gcp-config.sh
source "$ROOT/scripts/lib/gcp-config.sh"

ENV_FILE="${ENV_FILE:-$ROOT/secrets.env}"
PROJECT="$(gcp_project || true)"
SECRET_ID="esxi-lab-env"

if [[ -z "$PROJECT" ]]; then
	echo "Set GCP_PROJECT or run task infra:gcloud (tofu output gcp_project)" >&2
	exit 1
fi

SECRET_KEYS=(
	ESXI_PASSWORD
	OPNSENSE_API_KEY
	OPNSENSE_API_SECRET
	WIREGUARD_SERVER_PRIVATE_KEY
	SMTP_PASSWORD
)

declare -A MERGED=()

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

remote_secret_versions() {
	gcloud secrets versions list "$SECRET_ID" \
		--project="$PROJECT" \
		--filter="state=ENABLED" \
		--format="value(name)" \
		--sort-by="~createTime" 2>/dev/null || true
}

load_remote_secrets() {
	local version data line key val
	for version in $(remote_secret_versions); do
		data="$(gcloud secrets versions access "$version" \
			--secret="$SECRET_ID" \
			--project="$PROJECT" 2>/dev/null || true)"
		[[ -n "$data" ]] || continue
		while IFS= read -r line || [[ -n "$line" ]]; do
			line="$(trim "$line")"
			[[ -z "$line" || "$line" == \#* ]] && continue
			if [[ "$line" =~ ^([A-Z_]+)=(.*)$ ]]; then
				key="${BASH_REMATCH[1]}"
				val="${BASH_REMATCH[2]}"
				if [[ -n "$val" && -z "${MERGED[$key]+x}" ]]; then
					MERGED["$key"]="$val"
				fi
			fi
		done <<<"$data"
	done
}

trim() {
	local s="$1"
	s="${s#"${s%%[![:space:]]*}"}"
	s="${s%"${s##*[![:space:]]}"}"
	echo "$s"
}

has_local_credentials() {
	local key val
	for key in "${SECRET_KEYS[@]}"; do
		val="$(secret_value "$key")"
		if [[ -n "$val" ]]; then
			return 0
		fi
	done
	return 1
}

remote_has_credentials() {
	local key
	load_remote_secrets
	for key in "${SECRET_KEYS[@]}"; do
		if [[ -n "${MERGED[$key]:-}" ]]; then
			return 0
		fi
	done
	return 1
}

render_secret_file() {
	echo "# Uploaded $(date -u +%Y-%m-%dT%H:%M:%SZ)"
	local key
	for key in "${SECRET_KEYS[@]}"; do
		if [[ -n "${MERGED[$key]:-}" ]]; then
			echo "${key}=${MERGED[$key]}"
		fi
	done
}

if ! gcloud secrets describe "$SECRET_ID" --project="$PROJECT" &>/dev/null; then
	echo "Secret $SECRET_ID not found in $PROJECT — run task infra:gcloud" >&2
	exit 1
fi

load_remote_secrets

local_changed=0
for key in "${SECRET_KEYS[@]}"; do
	val="$(secret_value "$key")"
	if [[ -n "$val" && "${MERGED[$key]:-}" != "$val" ]]; then
		MERGED["$key"]="$val"
		local_changed=1
	fi
done

if [[ "$local_changed" -eq 0 ]] && ! has_local_credentials; then
	if remote_has_credentials; then
		echo "No credentials in $ENV_FILE or env — using existing Secret Manager version (projects/$PROJECT/secrets/$SECRET_ID)"
		exit 0
	fi
	echo "No credentials in $ENV_FILE or env (need: ${SECRET_KEYS[*]})" >&2
	exit 1
fi

TMP="$(mktemp)"
trap 'rm -f "$TMP"' EXIT
render_secret_file >"$TMP"

if ! grep -qE '^[A-Z_]+=.+' "$TMP"; then
	echo "No credentials to upload (need: ${SECRET_KEYS[*]})" >&2
	exit 1
fi

LATEST_KEYS="$(gcloud secrets versions access latest \
	--secret="$SECRET_ID" \
	--project="$PROJECT" 2>/dev/null | grep -E '^[A-Z_]+=' | sort || true)"
NEW_KEYS="$(grep -E '^[A-Z_]+=' "$TMP" | sort || true)"
if [[ "$NEW_KEYS" == "$LATEST_KEYS" ]]; then
	echo "Secret Manager already up to date (projects/$PROJECT/secrets/$SECRET_ID)"
	exit 0
fi

gcloud secrets versions add "$SECRET_ID" \
	--project="$PROJECT" \
	--data-file="$TMP"

echo "Uploaded to projects/$PROJECT/secrets/$SECRET_ID"
