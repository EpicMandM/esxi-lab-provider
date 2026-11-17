"""Scheduled cost guard: prune Secret Manager versions and disable unused APIs.

Keeps this lab project inside Always Free bounds. Invoked by Cloud Scheduler.
"""

from __future__ import annotations

import json
import os
from typing import Any

import google.auth
from google.cloud import secretmanager
from googleapiclient.discovery import build

# Secrets owned by this lab — keep only the newest N non-destroyed versions.
SECRET_IDS = ("esxi-lab-env", "esxi-lab-wg0")

# APIs required by the lab + this function itself (never disable).
KEEP_APIS = frozenset(
    {
        "calendar-json.googleapis.com",
        "gmail.googleapis.com",
        "monitoring.googleapis.com",
        "forms.googleapis.com",
        "sheets.googleapis.com",
        "script.googleapis.com",
        "storage.googleapis.com",
        "storage-api.googleapis.com",
        "storage-component.googleapis.com",
        "secretmanager.googleapis.com",
        "iam.googleapis.com",
        "iamcredentials.googleapis.com",
        "cloudresourcemanager.googleapis.com",
        "serviceusage.googleapis.com",
        "cloudapis.googleapis.com",
        "servicemanagement.googleapis.com",
        "logging.googleapis.com",
        # cost-guard runtime
        "cloudfunctions.googleapis.com",
        "cloudscheduler.googleapis.com",
        "run.googleapis.com",
        "cloudbuild.googleapis.com",
        "artifactregistry.googleapis.com",
        "eventarc.googleapis.com",
        "pubsub.googleapis.com",
    }
)

# High-risk / unused APIs to disable when found enabled.
DISABLE_APIS = (
    "analyticshub.googleapis.com",
    "bigquery.googleapis.com",
    "bigqueryconnection.googleapis.com",
    "bigquerydatapolicy.googleapis.com",
    "bigquerydatatransfer.googleapis.com",
    "bigquerymigration.googleapis.com",
    "bigqueryreservation.googleapis.com",
    "bigquerystorage.googleapis.com",
    "chat.googleapis.com",
    "cloudtrace.googleapis.com",
    "containerregistry.googleapis.com",
    "dataform.googleapis.com",
    "dataplex.googleapis.com",
    "datastore.googleapis.com",
    "drive.googleapis.com",
    "sql-component.googleapis.com",
    "telemetry.googleapis.com",
)


def _project() -> str:
    return os.environ.get("GCP_PROJECT") or os.environ.get("GOOGLE_CLOUD_PROJECT") or ""


def _keep() -> int:
    raw = os.environ.get("KEEP_SECRET_VERSIONS", "1")
    try:
        n = int(raw)
    except ValueError:
        n = 1
    return max(1, n)


def prune_secret(client: secretmanager.SecretManagerServiceClient, project: str, secret_id: str, keep: int) -> list[str]:
    parent = f"projects/{project}/secrets/{secret_id}"
    versions = [
        v
        for v in client.list_secret_versions(request={"parent": parent})
        if v.state != secretmanager.SecretVersion.State.DESTROYED
    ]
    versions.sort(key=lambda v: v.create_time, reverse=True)
    destroyed: list[str] = []
    for version in versions[keep:]:
        client.destroy_secret_version(request={"name": version.name})
        destroyed.append(version.name)
    return destroyed


def disable_unused_apis(project: str) -> list[str]:
    creds, _ = google.auth.default(scopes=["https://www.googleapis.com/auth/cloud-platform"])
    usage = build("serviceusage", "v1", credentials=creds, cache_discovery=False)
    disabled: list[str] = []
    for api in DISABLE_APIS:
        if api in KEEP_APIS:
            continue
        name = f"projects/{project}/services/{api}"
        try:
            svc = usage.services().get(name=name).execute()
        except Exception:
            continue
        if svc.get("state") != "ENABLED":
            continue
        try:
            # Never cascade — dependent disables can turn off Scheduler/Functions.
            usage.services().disable(
                name=name,
                body={"disableDependentServices": False},
            ).execute()
            disabled.append(api)
        except Exception as exc:
            disabled.append(f"{api}:error:{exc}")
    return disabled


def run() -> dict[str, Any]:
    project = _project()
    if not project:
        raise RuntimeError("GCP_PROJECT / GOOGLE_CLOUD_PROJECT is unset")

    keep = _keep()
    sm = secretmanager.SecretManagerServiceClient()
    pruned: dict[str, list[str]] = {}
    for secret_id in SECRET_IDS:
        try:
            pruned[secret_id] = prune_secret(sm, project, secret_id, keep)
        except Exception as exc:  # secret may not exist yet
            pruned[secret_id] = [f"error: {exc}"]

    try:
        disabled = disable_unused_apis(project)
    except Exception as exc:
        disabled = [f"error: {exc}"]
    return {
        "project": project,
        "keep_secret_versions": keep,
        "pruned": pruned,
        "disabled_apis": disabled,
    }


def cost_guard(request):  # noqa: ARG001 — Cloud Functions HTTP signature
    try:
        result = run()
        return (json.dumps(result), 200, {"Content-Type": "application/json"})
    except Exception as exc:
        return (json.dumps({"error": str(exc)}), 500, {"Content-Type": "application/json"})
