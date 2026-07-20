#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck source=lib/gcp-config.sh
source "$ROOT/scripts/lib/gcp-config.sh"

KEY_FILE=""
cleanup() {
	if [[ -n "$KEY_FILE" && "$KEY_FILE" == /tmp/* ]]; then
		rm -f "$KEY_FILE"
	fi
}
trap cleanup EXIT

require_configured_gcp_project
project="$(gcp_project)"

if [[ -n "${GOOGLE_APPLICATION_CREDENTIALS:-}" && -f "${GOOGLE_APPLICATION_CREDENTIALS}" ]]; then
	KEY_FILE="$GOOGLE_APPLICATION_CREDENTIALS"
elif [[ -n "${GOOGLE_SERVICE_ACCOUNT_KEY:-}" ]]; then
	KEY_FILE="$(mktemp)"
	printf '%s' "$GOOGLE_SERVICE_ACCOUNT_KEY" >"$KEY_FILE"
	export GOOGLE_APPLICATION_CREDENTIALS="$KEY_FILE"
elif [[ -n "${GOOGLE_APPLICATION_CREDENTIALS:-}" ]]; then
	echo "GOOGLE_APPLICATION_CREDENTIALS is set but not a file: ${GOOGLE_APPLICATION_CREDENTIALS}" >&2
	exit 1
fi

if [[ -n "$KEY_FILE" ]]; then
	gcloud auth activate-service-account --key-file="$KEY_FILE" --quiet
	gcloud config set project "$project"
	echo "GCP auth OK (service account, project: $project)"
	exit 0
fi

if ci_mode; then
	echo "CI requires GOOGLE_APPLICATION_CREDENTIALS or GOOGLE_SERVICE_ACCOUNT_KEY" >&2
	exit 1
fi

have_user=false
have_adc=false
gcloud auth print-access-token >/dev/null 2>&1 && have_user=true
gcloud auth application-default print-access-token >/dev/null 2>&1 && have_adc=true

if ! $have_user; then
	echo "GCP login required..."
	gcloud auth login --update-adc
elif ! $have_adc; then
	echo "Application Default Credentials required for Terraform..."
	gcloud auth application-default login
fi

gcloud config set project "$project"

echo "GCP auth OK (project: $project)"
