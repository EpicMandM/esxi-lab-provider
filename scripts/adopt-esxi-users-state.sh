#!/usr/bin/env bash
# Adopt already-provisioned ESXi users into Terraform state without running govc.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIR="$ROOT/infra/terraform/esxi-users"

python3 - "$DIR" <<'PY'
import json
import random
import subprocess
import sys

workdir = sys.argv[1]

state = json.loads(
    subprocess.check_output(["tofu", "state", "pull"], cwd=workdir, text=True)
)

def resource(name, rtype):
    for res in state.get("resources", []):
        if res.get("mode") == "managed" and res.get("name") == name and res.get("type") == rtype:
            return res
    return None

role = resource("esxi_role", "null_resource")
if not role or not role.get("instances"):
    raise SystemExit("esxi_role missing from state — run task infra:esxi against a reachable ESXi host first")

role_triggers = role["instances"][0]["attributes"]["triggers"]
passwords = {}
for res in state.get("resources", []):
    if res.get("type") != "random_password" or res.get("name") != "user_passwords":
        continue
    for inst in res.get("instances", []):
        passwords[inst["index_key"]] = inst["attributes"]["result"]

users = {
    f"lab-user-{i}": {
        "index": i,
        "vm_fortigate": f"Pod-{i}_FortiGate",
        "vm_client_deb": f"Pod-{i}_Client_Deb",
        "password": passwords[i - 1],
    }
    for i in range(1, 5)
}

fortigate_pods = {1, 2, 3}
role_name = role_triggers["role_name"]

def sensitive_triggers(*keys):
    return [
        [
            {"type": "get_attr", "value": "triggers"},
            {"type": "index", "value": {"value": key, "type": "string"}},
        ]
        for key in keys
    ]

def append_instance(resources, res_name, key, triggers, sensitive_keys=()):
    res = next(
        (r for r in resources if r.get("type") == "null_resource" and r.get("name") == res_name),
        None,
    )
    if res is None:
        res = {
            "mode": "managed",
            "type": "null_resource",
            "name": res_name,
            "provider": 'provider["registry.opentofu.org/hashicorp/null"]',
            "instances": [],
        }
        resources.append(res)

    for inst in res.get("instances", []):
        if inst.get("index_key") == key:
            inst["attributes"]["triggers"] = triggers
            if sensitive_keys:
                inst["sensitive_attributes"] = sensitive_triggers(*sensitive_keys)
            return

    instance = {
        "index_key": key,
        "schema_version": 0,
        "attributes": {
            "id": str(random.randint(10**18, 10**19 - 1)),
            "triggers": triggers,
        },
    }
    if sensitive_keys:
        instance["sensitive_attributes"] = sensitive_triggers(*sensitive_keys)

    res["instances"].append(instance)

resources = state.setdefault("resources", [])

for username, user in users.items():
    append_instance(
        resources,
        "esxi_users",
        username,
        {
            "username": username,
            "password": user["password"],
            "esxi_url": role_triggers["esxi_url"],
            "esxi_username": role_triggers["esxi_username"],
            "esxi_password": role_triggers["esxi_password"],
        },
        ("password", "esxi_password"),
    )

    append_instance(
        resources,
        "client_deb_permissions",
        username,
        {
            "username": username,
            "vm_name": user["vm_client_deb"],
            "role_name": role_name,
            "esxi_url": role_triggers["esxi_url"],
            "esxi_username": role_triggers["esxi_username"],
            "esxi_password": role_triggers["esxi_password"],
        },
        ("esxi_password",),
    )

    if user["index"] in fortigate_pods:
        append_instance(
            resources,
            "fortigate_permissions",
            username,
            {
                "username": username,
                "vm_name": user["vm_fortigate"],
                "role_name": role_name,
                "esxi_url": role_triggers["esxi_url"],
                "esxi_username": role_triggers["esxi_username"],
                "esxi_password": role_triggers["esxi_password"],
            },
            ("esxi_password",),
        )

state["serial"] = int(state.get("serial", 0)) + 1

subprocess.run(
    ["tofu", "state", "push", "-force", "-"],
    cwd=workdir,
    input=json.dumps(state),
    text=True,
    check=True,
)
print("Adopted existing ESXi users into Terraform state (no govc changes).")
PY
