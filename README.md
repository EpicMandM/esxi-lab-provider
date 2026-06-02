# ESXi Lab Provider

Hourly systemd job: read Google Calendar events, restore ESXi VM snapshots, rotate passwords, rotate WireGuard keys on OPNsense, email credentials to attendees.

## Prerequisites

- Dev container (Alpine image: Go, OpenTofu, gcloud, gh, govc, Task)
- GCP project with Calendar + Gmail APIs
- ESXi host and OPNsense (WireGuard optional)

Prebuilt dev container image: `ghcr.io/epicmandm/esxi-lab-provider/devcontainer:latest` (built on push to `main` when `.devcontainer/` changes).

## Setup

```bash
task infra:init
```

Edit the files `infra:init` creates:

| File | What to set |
|------|-------------|
| `infra/terraform/lab-common.auto.tfvars` | GCP project, Secret Manager secret id |
| `infra/terraform/lab-esxi.auto.tfvars` | ESXi URL and admin username |
| `infra/terraform/lab-network.auto.tfvars` | OPNsense URL |
| `infra/terraform/lab-smtp.auto.tfvars` | SMTP username/from (if using email) |
| `infra/terraform/wireguard-config.auto.tfvars` | WireGuard server keys, endpoint, peer IPs |
| `secrets.env` | ESXi password, OPNsense API keys, WG private key, SMTP password |

### Provision

```bash
task infra:apply
```

Prompts for GCP login on first run if needed. Then it creates GCP resources, uploads `secrets.env` to Secret Manager, provisions ESXi/WireGuard, and generates `.env` + `api/data/user_config.toml`.

### CI

Set these instead of interactive login:

| Variable | Purpose |
|----------|---------|
| `GCP_PROJECT` | GCP project (overrides `lab-common.auto.tfvars`) |
| `LAB_ENV_SECRET_ID` | Secret Manager secret id (optional override) |
| `GOOGLE_APPLICATION_CREDENTIALS` | Path to service account JSON key |
| `GOOGLE_SERVICE_ACCOUNT_KEY` | Service account JSON inline (alternative to file path) |
| `ESXI_PASSWORD`, `OPNSENSE_API_KEY`, … | Upload secrets without `secrets.env` |

With `CI=true`, auth fails fast if no service account credentials are provided.

```bash
export CI=true
export GCP_PROJECT=your-project
export GOOGLE_SERVICE_ACCOUNT_KEY='{"type":"service_account",...}'
export ESXI_PASSWORD=...
task infra:apply
```

### Share calendar

Share your lab calendar with the service account email:

```bash
cd infra/terraform/gcloud && tofu output service_account_email
```

Grant **Make changes to events**.

### Deploy scheduler

```bash
task deploy:local
task status
task logs
task test-run
```

Override deploy target: `DEPLOY_HOST`, `DEPLOY_USER`, `DEPLOY_PORT`.

## Configuration model

| What | Where |
|------|-------|
| Lab URLs, WireGuard topology | `lab-*.auto.tfvars`, `wireguard-config.auto.tfvars` |
| Passwords and API keys | `secrets.env` → Secret Manager |
| App runtime files | Generated `.env`, `user_config.toml` |

## Tasks

```bash
task test
task test:coverage
task infra:apply      # full provisioning pipeline
task tunnel           # local ESXi tunnel via WireGuard
```

## Layout

```
api/                      Go scheduler
infra/terraform/
  gcloud/                 GCP + Secret Manager
  esxi-users/             ESXi accounts + VM ACLs
  opnsense-wireguard/     WireGuard on OPNsense
  app-config/             Generates .env + user_config.toml
scripts/push-lab-env-secret.sh
Taskfile.yml
```

## License

MIT
