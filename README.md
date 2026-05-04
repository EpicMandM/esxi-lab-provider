# ESXi Lab Provider

Automated VMware ESXi lab scheduler with Google Calendar integration, WireGuard VPN provisioning, and email notifications.

Runs on a systemd timer (every hour). When a calendar event is active, the scheduler restores VM snapshots, rotates ESXi user passwords, generates per-user WireGuard configs, registers peers with OPNsense, and emails credentials + VPN config to attendees.

## Architecture

```
Google Calendar ──▶ Orchestrator ──▶ VMware ESXi  (snapshot restore + password rotation)
                        │
                        ├──▶ WireGuard / OPNsense  (key rotation + peer registration)
                        │
                        └──▶ Gmail SMTP  (credentials + .conf attachment)
```

All business logic lives in `internal/orchestrator`. Services (`VMwareClient`, `CalendarClient`, `EmailSender`, `WireGuardManager`) are interfaces defined in `internal/service/interfaces.go`, injected by `cmd/server/main.go`.

## Prerequisites

- VS Code with Dev Containers extension
- Docker
- Google Cloud project with Calendar + Gmail APIs
- VMware ESXi host
- OPNsense firewall (for WireGuard, optional)

## Setup

### 1. Dev Container

```bash
code .
# Reopen in Container when prompted
```

Pre-installed: Go, OpenTofu, gcloud CLI, Task, Ansible, WireGuard tools, pre-commit.

### 2. Google Cloud Auth

```bash
gcloud auth login
gcloud config set project exsi-chat-app-478319
gcloud auth application-default login
```

### 3. Infrastructure Provisioning

The project uses four independent Terraform modules under `infra/terraform/`:

| Module | Purpose |
|--------|---------|
| `gcloud/` | Enables Calendar + Gmail APIs, creates service account, writes `service-account.json` |
| `esxi-users/` | Creates ESXi local accounts (`lab-user-1..N`), assigns per-VM permissions |
| `opnsense-wireguard/` | Creates WireGuard peers on OPNsense, outputs server keys + peer addresses |
| `app-config/` | Reads outputs from the above three and generates `user_config.toml` |

Apply in order:

```bash
cd infra/terraform

# 1. Google Cloud service account
cd gcloud && tofu init && tofu apply && cd ..

# 2. ESXi users + VM mappings
cd esxi-users && tofu init && tofu apply && cd ..

# 3. WireGuard peers on OPNsense
cd opnsense-wireguard && tofu init && tofu apply && cd ..

# 4. Generate user_config.toml (consumes outputs from all above)
cd app-config && tofu init && tofu apply && cd ..
```

### 4. Auto-Generated `user_config.toml`

The `app-config` module renders `user_config.toml` from the template at `infra/terraform/app-config/templates/user_config.toml.tftpl`. It reads remote state from `esxi-users` and `opnsense-wireguard`, plus variables defined in `app-config/variables.tf`.

**Do not edit `user_config.toml` manually** — re-run `tofu apply` in `app-config/` to regenerate.

Generated output (`api/data/user_config.toml`):

```toml
[calendar]
calendar_id = "c_...@group.calendar.google.com"
service_account_path = "/app/config/service-account.json"

[esxi]
url = "https://esxi.example.com"
# snapshot_name = "clean-state"  # omitted = use latest

[esxi.user_vm_mappings]
"lab-user-1" = "Pod-1_FortiGate"
"lab-user-2" = "Pod-2_FortiGate"

[wireguard]
enabled = true
server_public_key = "base64..."
server_endpoint = "vpn.example.com:51820"
opnsense_url = "https://opnsense.local"
auto_register_peers = true
server_tunnel_network = "172.17.18.0/24"
allowed_ips = ["172.17.17.0/24"]
mtu = 1380
client_addresses = ["172.17.18.101/32", "172.17.18.102/32"]
keepalive = 0
```

Data sources for each section:

| Config section | Source |
|----------------|--------|
| `[calendar]` | `app-config/variables.tf` (`calendar_id`, `service_account_path`) |
| `[esxi]` | `esxi-users` state (`esxi_url`, `user_vm_mappings`) + `app-config/variables.tf` (`esxi_snapshot_name`) |
| `[wireguard]` | `opnsense-wireguard` state (`server_public_key`, `server_endpoint`, `opnsense_url`, `peer_tunnel_addresses`) + `app-config/variables.tf` (remaining fields) |

### 5. Share Google Calendar

1. Open [Google Calendar](https://calendar.google.com) → calendar settings → **Share with specific people**
2. Add the service account email (from `cd infra/terraform/gcloud && tofu output service_account_email`)
3. Grant **Make changes to events**

### 6. Environment Variables

Copy and fill `api/.env.example`:

```bash
cp api/.env.example .env
```

```dotenv
# Required
ESXI_URL=https://esxi.example.com
ESXI_USERNAME=root
ESXI_PASSWORD=secret
ESXI_INSECURE=true

# Optional: email notifications
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USERNAME=you@gmail.com
SMTP_PASSWORD=app-password          # or use SMTP_PASSWORD_FILE
SMTP_FROM=you@gmail.com
TEST_EMAIL_ONLY=test@gmail.com      # redirect all emails here (test mode)

# Optional: WireGuard peer auto-registration
OPNSENSE_API_KEY=key
OPNSENSE_API_SECRET=secret
OPNSENSE_INSECURE=true              # self-signed OPNsense cert
```

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `ESXI_URL` | Yes | — | ESXi host URL |
| `ESXI_USERNAME` | Yes | — | ESXi login |
| `ESXI_PASSWORD` | Yes | — | ESXi password |
| `ESXI_INSECURE` | No | `false` | Skip TLS verification |
| `CONFIG_PATH` | No | `./data/user_config.toml` | Path to feature config |
| `SERVICE_ACCOUNT_PATH` | No | from TOML | Overrides `service_account_path` in config |
| `SMTP_HOST` | No | `smtp.gmail.com` | SMTP server |
| `SMTP_PORT` | No | `587` | SMTP port |
| `SMTP_USERNAME` | No | — | SMTP login |
| `SMTP_PASSWORD` | No | — | SMTP password |
| `SMTP_PASSWORD_FILE` | No | — | Read SMTP password from file (overrides `SMTP_PASSWORD`) |
| `SMTP_FROM` | No | `SMTP_USERNAME` | Sender address |
| `TEST_EMAIL_ONLY` | No | — | Redirect all emails to this address |
| `OPNSENSE_API_KEY` | No | — | OPNsense API key |
| `OPNSENSE_API_SECRET` | No | — | OPNsense API secret |
| `OPNSENSE_INSECURE` | No | `false` | Skip OPNsense TLS verification |

## Taskfile Commands

All automation is in `Taskfile.yml`. Run `task --list` for the full list.

### Build & Test

```bash
task build              # CGO_ENABLED=0 GOOS=linux GOARCH=amd64 build
task test               # go test ./... -count=1
task test:verbose        # go test ./... -v -count=1
task test:coverage       # tests + coverage gate (config, logger, orchestrator must be 100%)
```

### Deployment (SSH)

Configured via env vars or Taskfile defaults (`DEPLOY_USER=zhukov`, `DEPLOY_HOST=172.17.17.8`, `DEPLOY_PORT=22`).

```bash
task ssh-keygen         # generate + upload SSH key
task setup-sudo         # passwordless sudo on remote
task setup-logging      # journald 7-day retention

task deploy             # build → scp binary, .env, user_config.toml, service-account.json
                        #       → install systemd service + hourly timer

task status             # systemctl status + list-timers
task logs               # journalctl -n 50 (override: task logs LINES=200)
task test-run           # trigger one-shot run + tail logs
task stop               # stop + disable timer
task clean              # full removal: timer, service, files
```

The deploy task creates a systemd oneshot service (`esxi-lab-scheduler.service`) triggered by a timer (`esxi-lab-scheduler.timer`) at `*:00:00`. Files are deployed to `/opt/esxi-lab/`.

## How It Works

1. **Timer fires** at the top of every hour
2. **Fetch VM inventory** from ESXi (snapshots per VM)
3. **Query Google Calendar** for events active within ±5 minutes of now
4. **Match VMs** — configured user-VM pairs from `[esxi.user_vm_mappings]` first, then fallback to any available inventory VM
5. **Restore snapshots** — revert to named snapshot or latest, power on
6. **Rotate ESXi passwords** — new random 16-char password per user via `HostLocalAccountManager`
7. **Rotate WireGuard keys** — generate Curve25519 keypair, register public key with OPNsense API, generate client `.conf`
8. **Email credentials** — send VM name, username, password, and WireGuard `.conf` attachment to the calendar event attendee

## Project Structure

```
├── api/
│   ├── cmd/server/main.go           # Wiring: constructs services, runs orchestrator
│   ├── internal/
│   │   ├── config/                   # Env-based infra config (ESXi URL/creds)
│   │   ├── logger/                   # Structured JSON logger
│   │   ├── models/                   # VM, VMSnapshot, VMListResponse
│   │   ├── orchestrator/             # All business logic
│   │   └── service/
│   │       ├── interfaces.go         # VMwareClient, CalendarClient, EmailSender, WireGuardManager
│   │       ├── calendar_config.go    # FeatureConfig + TOML loading
│   │       ├── calendar.go           # Google Calendar API client
│   │       ├── vmware.go             # govmomi: snapshots, power, password rotation
│   │       ├── gmail.go              # SMTP + MIME email with attachments
│   │       ├── password.go           # Crypto-random password generation
│   │       └── wireguard.go          # Key generation, client config, OPNsense API
│   ├── .env.example
│   └── go.mod
├── infra/terraform/
│   ├── gcloud/                       # GCP service account + API enablement
│   ├── esxi-users/                   # ESXi local accounts + per-VM permissions
│   ├── opnsense-wireguard/           # WireGuard peers on OPNsense
│   └── app-config/                   # Generates user_config.toml from above states
│       └── templates/user_config.toml.tftpl
├── Taskfile.yml
└── .devcontainer/
```

## Development

### Tests

```bash
task test               # all tests
task test:verbose        # verbose
task test:coverage       # coverage gate — config, logger, orchestrator must be 100%
```

Test files (`*_test.go`) are the specification and must not be modified. Hand-written mocks with function fields are used (no codegen).

### Pre-commit

```bash
pre-commit install
pre-commit run --all-files
```

## License

MIT
