#!/usr/bin/env bash
# Resolve lab Terraform variables for infra:apply (remote by default, local override optional).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck source=lib/lab-config.sh
source "$ROOT/scripts/lib/lab-config.sh"

override="$(lab_override_file)"
remote_cache="$(lab_remote_cache_file)"

if [[ -f "$override" ]]; then
	echo "Lab config: using local override (infra/terraform/lab/lab.auto.tfvars)"
	exit 0
fi

if ! lab_has_remote_state; then
	cat >&2 <<EOF
No lab configuration found.

  • Fresh clone of an existing lab: ensure GCP auth works, then re-run task infra:apply
  • First-time setup or local overrides: run task infra:init, edit the created files, then task infra:apply
EOF
	exit 1
fi

lab_write_tfvars_from_remote "$remote_cache"
echo "Lab config: loaded from GCS remote state (infra/terraform/lab/lab.remote.auto.tfvars)"
