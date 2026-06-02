#!/usr/bin/env bash

read_tfvar() {
	local name="$1"
	local file="$2"
	grep -E "^${name}[[:space:]]*=" "$file" | head -1 \
		| cut -d= -f2- | sed 's/^[[:space:]]*//; s/[[:space:]]*$//; s/^"//; s/"$//; s/#.*//; s/[[:space:]]*$//'
}

lab_config_file() {
	local root="${LAB_ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)}"
	echo "${LAB_CONFIG:-$root/infra/terraform/lab-common.auto.tfvars}"
}

gcp_project() {
	if [[ -n "${GCP_PROJECT:-}" ]]; then
		echo "$GCP_PROJECT"
		return
	fi
	local file
	file="$(lab_config_file)"
	[[ -f "$file" ]] || return 1
	read_tfvar gcp_project "$file"
}

lab_env_secret_id() {
	if [[ -n "${LAB_ENV_SECRET_ID:-}" ]]; then
		echo "$LAB_ENV_SECRET_ID"
		return
	fi
	local file
	file="$(lab_config_file)"
	[[ -f "$file" ]] || return 1
	read_tfvar lab_env_secret_id "$file"
}

ci_mode() {
	[[ "${CI:-}" == "true" || -n "${GITHUB_ACTIONS:-}" || -n "${GITLAB_CI:-}" ]]
}
