#!/usr/bin/env bash

lab_config_dir() {
	local root="${LAB_ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)}"
	echo "$root/infra/terraform/lab"
}

lab_override_file() {
	echo "$(lab_config_dir)/lab.auto.tfvars"
}

lab_remote_cache_file() {
	echo "$(lab_config_dir)/lab.remote.auto.tfvars"
}

lab_has_remote_state() {
	local dir
	dir="$(lab_config_dir)"
	tofu -chdir="$dir" init -input=false >/dev/null 2>&1 || return 1
	tofu -chdir="$dir" output -json esxi_url >/dev/null 2>&1
}

lab_write_tfvars_from_remote() {
	local dest="$1"
	local dir outputs
	dir="$(lab_config_dir)"
	outputs="$(tofu -chdir="$dir" output -json 2>/dev/null)" || return 1
	LAB_TFVARS_DEST="$dest" LAB_TFVARS_JSON="$outputs" python3 <<'PY'
import json
import os
import sys

outputs = json.loads(os.environ["LAB_TFVARS_JSON"])
dest = os.environ["LAB_TFVARS_DEST"]

# Terraform input variables (wireguard_server_public_key is derived, not stored).
keys = [
    ("esxi_url", "string"),
    ("esxi_admin_username", "string"),
    ("opnsense_url", "string"),
    ("smtp_username", "string"),
    ("smtp_from", "string"),
    ("wireguard_server_name", "string"),
    ("wireguard_server_port", "number"),
    ("wireguard_public_endpoint", "string"),
    ("wireguard_server_tunnel_address", "string"),
    ("wireguard_server_mtu", "number"),
    ("wireguard_server_dns", "list"),
    ("wireguard_existing_peer_ids", "list"),
    ("peer1_tunnel_address", "string"),
    ("peer2_tunnel_address", "string"),
    ("peer3_tunnel_address", "string"),
    ("peer4_tunnel_address", "string"),
]

missing = [name for name, _ in keys if name not in outputs]
if missing:
    print(f"Remote lab state missing outputs: {', '.join(missing)}", file=sys.stderr)
    sys.exit(1)


def hcl_string(value: str) -> str:
    return json.dumps(value)


def hcl_list(values) -> str:
    if not values:
        return "[]"
    items = ", ".join(hcl_string(str(v)) for v in values)
    return f"[{items}]"


lines = [
    "# Generated from GCS lab remote state — do not commit.",
    "# Run task infra:init to create lab.auto.tfvars for local overrides.",
    "",
]
for name, kind in keys:
    value = outputs[name]["value"]
    if kind == "string":
        lines.append(f'{name} = {hcl_string(value)}')
    elif kind == "number":
        lines.append(f"{name} = {value}")
    elif kind == "list":
        lines.append(f"{name} = {hcl_list(value)}")

with open(dest, "w", encoding="utf-8") as fh:
    fh.write("\n".join(lines) + "\n")
PY
}
