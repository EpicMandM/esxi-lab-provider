#!/usr/bin/env bash
# Write merged Secret Manager credentials to secrets.env (used by infra:init).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck source=lib/gcp-config.sh
source "$ROOT/scripts/lib/gcp-config.sh"

ENV_FILE="${ENV_FILE:-$ROOT/secrets.env}"
PROJECT="$(gcp_project || true)"
SECRET_ID="esxi-lab-env"

SECRET_KEYS=(
	ESXI_PASSWORD
	OPNSENSE_API_KEY
	OPNSENSE_API_SECRET
	WIREGUARD_SERVER_PRIVATE_KEY
	SMTP_PASSWORD
)

if [[ -z "$PROJECT" ]]; then
	echo "Set GCP_PROJECT or run task infra:gcloud (tofu output gcp_project)" >&2
	exit 1
fi

if ! gcloud secrets describe "$SECRET_ID" --project="$PROJECT" &>/dev/null; then
	echo "Secret $SECRET_ID not found in $PROJECT — run task infra:gcloud first" >&2
	exit 1
fi

declare -A MERGED=()

remote_secret_versions() {
	gcloud secrets versions list "$SECRET_ID" \
		--project="$PROJECT" \
		--filter="state=ENABLED" \
		--format="value(name)" \
		--sort-by="~createTime" 2>/dev/null || true
}

for version in $(remote_secret_versions); do
	data="$(gcloud secrets versions access "$version" \
		--secret="$SECRET_ID" \
		--project="$PROJECT" 2>/dev/null || true)"
	[[ -n "$data" ]] || continue
	while IFS= read -r line || [[ -n "$line" ]]; do
		line="${line#"${line%%[![:space:]]*}"}"
		line="${line%"${line##*[![:space:]]}"}"
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

{
	echo "# Pulled from Secret Manager $(date -u +%Y-%m-%dT%H:%M:%SZ)"
	echo "# Edit values here to override remote credentials on task infra:apply"
	for key in "${SECRET_KEYS[@]}"; do
		echo "${key}=${MERGED[$key]:-}"
	done
} >"$ENV_FILE"
chmod 600 "$ENV_FILE"

echo "Wrote $ENV_FILE from projects/$PROJECT/secrets/$SECRET_ID"
