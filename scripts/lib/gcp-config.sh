#!/usr/bin/env bash

gcp_project_from_tf() {
	local root="${LAB_ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)}"
	local file="$root/infra/terraform/gcloud/main.tf"
	[[ -f "$file" ]] || return 1
	local line
	line="$(grep -E '[[:space:]]*gcp_project[[:space:]]*=' "$file" | head -1 || true)"
	[[ -n "$line" ]] || return 1
	echo "$line" | sed -n 's/.*gcp_project[[:space:]]*=[[:space:]]*"\([^"]*\)".*/\1/p'
}

gcp_project() {
	if [[ -n "${GCP_PROJECT:-}" ]]; then
		echo "$GCP_PROJECT"
		return
	fi
	local root="${LAB_ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)}"
	local project=""
	if project="$(tofu -chdir="$root/infra/terraform/gcloud" output -raw gcp_project 2>/dev/null)" && [[ -n "$project" ]]; then
		echo "$project"
		return
	fi
	project="$(gcp_project_from_tf || true)"
	if [[ -n "$project" ]]; then
		echo "$project"
		return
	fi
	return 1
}

gcp_project_is_placeholder() {
	case "${1:-}" in
	your-gcp-project | your-project | "") return 0 ;;
	*) return 1 ;;
	esac
}

require_configured_gcp_project() {
	local project
	project="$(gcp_project || true)"
	if gcp_project_is_placeholder "$project"; then
		echo "gcp_project is unset or still a template value (${project:-<empty>})." >&2
		echo "Set GCP_PROJECT, or set locals.gcp_project in infra/terraform/gcloud/main.tf, then retry." >&2
		exit 1
	fi
}

ci_mode() {
	[[ "${CI:-}" == "true" || -n "${GITHUB_ACTIONS:-}" || -n "${GITLAB_CI:-}" ]]
}
