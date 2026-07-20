# ESXi Lab Provider

Hourly systemd job: read Google Calendar events, restore ESXi VM snapshots, rotate passwords, rotate WireGuard keys on OPNsense, email credentials to attendees.

## Prerequisites

- Dev container (Debian image: Go, OpenTofu, gcloud, gh, govc, Task)
- GCP project with Calendar + Gmail APIs
- ESXi host and OPNsense (WireGuard optional)

Prebuilt dev container image (linux/amd64): `ghcr.io/epicmandm/esxi-lab-provider/devcontainer:latest` (built on a biweekly cron schedule when `.devcontainer/` changed in the last 14 days, or via **Actions → Devcontainer → Run workflow**).

## Setup

**Re-clone / existing lab** — one command:

```bash
task infra:apply
```

Pulls site settings from GCS remote state, credentials from Secret Manager, and generates `.env` + `api/data/user_config.toml`. WireGuard server public key is derived automatically.

**Local overrides** (different URLs, rotated secrets, first-time bootstrap):

```bash
task infra:init   # creates lab.auto.tfvars + secrets.env from remote (or examples)
# edit the files, then:
task infra:apply
```

| Source | What it holds |
|--------|----------------|
| GCS `lab` state | ESXi/OPNsense URLs, WireGuard topology |
| Secret Manager `esxi-lab-env` | Passwords and API keys |
| Secret Manager `esxi-lab-wg0` | Operator WireGuard client config (`wg0.conf`) |
| `lab.auto.tfvars` (optional) | Local override of site settings |
| `secrets.env` (optional) | Local override of credentials |
| Generated | `.env`, `user_config.toml`, `service-account.json` |

### Provision

```bash
task infra:apply
```

Prompts for GCP login on first run if needed. Then it creates GCP resources, uploads `secrets.env` + operator `wg0.conf` to Secret Manager, brings up WireGuard, provisions ESXi/WireGuard peers, and generates `.env` + `api/data/user_config.toml`.

First-time WireGuard: place your operator client config at `/etc/wireguard/wg0.conf` (or `wg0.conf` in the repo root) before `infra:apply` so `secrets:push` can upload it. Later clones pull it via `wg:apply`.

For the devcontainer, set `esxi_url = "https://127.0.0.1:10443"` in `lab.auto.tfvars` (the example default). `infra:esxi` starts the WireGuard tunnel automatically when that URL is used.

### CI

Set these instead of interactive login:

| Variable | Purpose |
|----------|---------|
| `GCP_PROJECT` | GCP project (overrides `lab-common.auto.tfvars`) |
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

Deploy and other remote tasks auto-apply WireGuard from Secret Manager and set up SSH/sudo if needed (password prompt on first SSH authorize / sudoers).

Override deploy target: `DEPLOY_HOST`, `DEPLOY_USER`, `DEPLOY_PORT`.

## Configuration model

| What | Where |
|------|-------|
| Lab URLs, WireGuard topology | GCS remote state (override: `lab.auto.tfvars`) |
| Passwords and API keys | Secret Manager `esxi-lab-env` (override: `secrets.env`) |
| Operator WireGuard client | Secret Manager `esxi-lab-wg0` |
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
