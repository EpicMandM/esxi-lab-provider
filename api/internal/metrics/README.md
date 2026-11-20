# Metrics

All application metrics are defined in [`metrics.go`](metrics.go) and pushed
directly to **Google Cloud Monitoring** via the
[OpenTelemetry GoogleCloud exporter](https://github.com/GoogleCloudPlatform/opentelemetry-operations-go).
No collector daemon is required.

## Architecture

```
Go binary  (systemd timer, exits after each run)
  └─▶  GoogleCloud OTel exporter  (in-process)
         └─▶  Google Cloud Monitoring API
```

Because the binary exits after every run, `MeterProvider.Shutdown()` is called
in `main.go` before exit to flush all pending data points.

Counters and histograms are exported as **CUMULATIVE** (Cloud Monitoring
custom/workload metric constraint). Dashboard widgets use `ALIGN_DELTA` so
each timer run shows as a discrete increment. UpDownCounters map to **GAUGE**
and use `ALIGN_MEAN` on the dashboard.

## Configuration

| Environment variable | Required | Description |
|----------------------|----------|-------------|
| `GOOGLE_APPLICATION_CREDENTIALS` | Yes (or `SERVICE_ACCOUNT_PATH`) | Path to `service-account.json` |
| `GOOGLE_CLOUD_PROJECT` / `GCP_PROJECT` | Recommended | Pins the Monitoring project (also set by deploy systemd unit) |

The required IAM role is `roles/monitoring.metricWriter`, bound in
`infra/terraform/gcloud/main.tf`.

## Metric Catalog

OTel names below land under **`workload.googleapis.com/<name>`** in Cloud
Monitoring (exporter default — not `custom.googleapis.com`).

Resource type is typically `generic_task` with `service.name=esxi-lab-scheduler`.

### Tier 1 — Business Outcomes

| Metric | Type | Unit | Labels | Description |
|--------|------|------|--------|-------------|
| `lab.run.duration` | Histogram | `s` | `status`: `success`/`failure` | Total wall-clock time of one orchestrator run |
| `lab.run.total` | Counter | — | `status`: `success`/`failure` | Number of completed runs |
| `lab.vm.restore.total` | Counter | — | `status`: `success`/`failure` | VM snapshot restores attempted |
| `lab.vm.restore.duration` | Histogram | `s` | — | Wall-clock duration of the restore + password-rotation batch |
| `lab.calendar.events.active` | UpDownCounter (gauge) | — | — | Active calendar events detected this run |
| `lab.email.send.total` | Counter | — | `status`, `has_attachment` | Credential emails sent |
| `lab.password.rotation.total` | Counter | — | `status`: `success`/`failure` | ESXi user password rotations performed |

### Tier 2 — Operational Visibility

| Metric | Type | Unit | Labels | Description |
|--------|------|------|--------|-------------|
| `lab.wireguard.key.rotation.total` | Counter | — | `status` | WireGuard key-pair rotations attempted |
| `lab.wireguard.peer.registration.total` | Counter | — | `status` | OPNsense peer registrations attempted |
| `lab.calendar.fetch.duration` | Histogram | `s` | `status` | Latency of the Google Calendar API call |

### Tier 3 — Inventory / Nice-to-Have

| Metric | Type | Unit | Labels | Description |
|--------|------|------|--------|-------------|
| `lab.vm.inventory.total` | UpDownCounter (gauge) | — | — | VMs discovered in the ESXi inventory |

## Dashboard

Terraform creates **ESXi Lab Provider — metrics**
(`infra/terraform/gcloud/dashboards/lab_metrics.json`) with aligners matched to
each metric kind. Apply via `task infra:gcloud`.

## Instrumentation Locations

| Metric | Source file | Function |
|--------|------------|----------|
| `lab.run.duration`, `lab.run.total` | `orchestrator.go` | `Run()` / `recordRunOutcome` |
| `lab.vm.restore.total`, `lab.vm.restore.duration` | `orchestrator.go` | `RestoreVMs()` |
| `lab.calendar.events.active` | `orchestrator.go` | `FetchActiveEventsAt()` |
| `lab.calendar.fetch.duration` | `orchestrator.go` | `FetchActiveEventsAt()` |
| `lab.email.send.total` | `orchestrator.go` | `RestoreVMs()` — per email send |
| `lab.password.rotation.total` | `orchestrator.go` | `RestoreVMs()` |
| `lab.wireguard.key.rotation.total` | `orchestrator.go` | `RestoreVMs()` |
| `lab.wireguard.peer.registration.total` | `orchestrator.go` | `RestoreVMs()` |
| `lab.vm.inventory.total` | `orchestrator.go` | `FetchVMInventory()` |

## Graceful Degradation

Metrics are **optional**. If `InitProvider` fails (e.g. wrong credentials,
no network access to Cloud Monitoring), the binary logs a warning and
continues without metrics. All instrumentation sites guard against a `nil`
`*Metrics` pointer.

## Running Without Metrics Locally

Simply do not set `GOOGLE_APPLICATION_CREDENTIALS`. The initialisation will
fail gracefully and the application will run in metrics-disabled mode.
