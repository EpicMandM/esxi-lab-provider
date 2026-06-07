#!/usr/bin/env bash

gcp_project() {
	if [[ -n "${GCP_PROJECT:-}" ]]; then
		echo "$GCP_PROJECT"
		return
	fi
	local root="${LAB_ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)}"
	tofu -chdir="$root/infra/terraform/gcloud" output -raw gcp_project 2>/dev/null
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
		echo "Run task infra:gcloud, or set GCP_PROJECT, then retry." >&2
		exit 1
	fi
}

ci_mode() {
	[[ "${CI:-}" == "true" || -n "${GITHUB_ACTIONS:-}" || -n "${GITLAB_CI:-}" ]]
}
