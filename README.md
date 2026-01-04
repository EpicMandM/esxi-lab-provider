# ESXi Lab Provider

> Automated VMware ESXi lab session scheduler with Google Calendar integration

An automated system that syncs Google Calendar events with VMware ESXi lab environments, provisioning and managing virtual machines for scheduled lab sessions. The scheduler runs hourly, automatically powering on VMs, restoring snapshots, and resetting passwords for upcoming sessions.

## Features

- 🗓️ Google Calendar integration for lab session scheduling
- 🖥️ Automated VM provisioning and snapshot management
- 🔐 Automatic password generation and rotation
- ⏱️ Hourly sync with systemd timer
- 📊 Structured logging with journald integration
- 🚀 Remote deployment via SSH

## Prerequisites

- **VS Code** with Dev Containers extension
- **Docker** (for dev containers)
- **Google Cloud Project** with billing enabled
- **VMware ESXi/vSphere** environment access
- **Remote Linux server** (for deployment, optional)

## Setup

### 1. Open in Dev Container

This project uses VS Code Dev Containers with all required tools pre-installed:
- Go 1.25.3+
- OpenTofu (Terraform alternative)
- Google Cloud CLI
- Task (taskfile.dev)
- Ansible
- Pre-commit hooks

```bash
# Open in VS Code
code .

# VS Code will prompt to "Reopen in Container"
# Or use Command Palette: "Dev Containers: Reopen in Container"
```

The dev container includes:
- Go toolchain
- OpenTofu for infrastructure provisioning
- gcloud CLI for Google Cloud authentication
- Task for running common commands
- WireGuard tools (for VPN if needed)

### 2. Authenticate with Google Cloud

```bash
# Login to Google Cloud (opens browser)
gcloud auth login

# Set the project
gcloud config set project exsi-chat-app-478319

# Enable application default credentials for OpenTofu
gcloud auth application-default login
```

### 3. Provision Google Calendar Service Account

Navigate to the Terraform directory and initialize:

```bash
cd infra/terraform

# Initialize OpenTofu/Terraform
tofu init

# Review the plan
tofu plan

# Apply the configuration
tofu apply
```

**If service account already exists**, import it into Terraform state:

```bash
# Import existing service account
tofu import google_service_account.calendar_sa projects/exsi-chat-app-478319/serviceAccounts/calendar-service-account@exsi-chat-app-478319.iam.gserviceaccount.com

# Then apply to create the key
tofu apply
```

This will:
- Enable Google Calendar API
- Create (or import existing) service account
- Generate a new service account key

**Extract the service account JSON:**

```bash
# Save the service account key to a file
tofu output -raw service_account_key > service-account.json

# Verify the file was created
cat service-account.json | jq .
```

⚠️ **Important:** Keep `service-account.json` secure and never commit it to version control (already in `.gitignore`).

### 4. Share Google Calendar with Service Account

1. Go to [Google Calendar](https://calendar.google.com)
2. Find or create the calendar you want to use for lab bookings
3. Click the three dots next to the calendar → **Settings and sharing**
4. Under "Share with specific people", add the service account email:
   ```bash
   # Get the service account email
   tofu output -raw service_account_key | jq -r .client_email
   ```
5. Grant **"Make changes to events"** permission
6. Copy the **Calendar ID** from calendar settings

### 5. Configure Application

#### Create `.env` file in project root:

```bash
# VMware vSphere/ESXi Configuration
VCENTER_URL=https://your-vcenter.example.com
VCENTER_USERNAME=administrator@vsphere.local
VCENTER_PASSWORD=your-password
VCENTER_INSECURE=true  # Set to false for production with valid certificates

# SQLite Database Path
DB_PATH=./scheduler.db

# Optional: Logging Configuration
LOG_LEVEL=info
```

#### Edit `api/data/user_config.toml`:

```toml
[calendar]
# Your Google Calendar ID from step 4
calendar_id = "c_xxxxxxxxxxxxx@group.calendar.google.com"

# Path to service account JSON (for local dev)
service_account_path = "./infra/terraform/service-account.json"

[vsphere]
# List of VM names available for lab sessions
vms = [
    "lab-vm-01",
    "lab-vm-02",
    "lab-vm-03"
]

# List of VM users for password generation
users = [
    "lab-user1",
    "lab-user2",
    "lab-user3"
]

# Optional: Specific snapshot to restore
# snapshot_name = "clean-state"
```

### 6. Build and Run Locally

```bash
# Build the scheduler binary
task build

# Run once for testing
./esxi-lab-scheduler

# Check logs
tail -f scheduler.log
```

## Deployment

The scheduler is designed to run on a remote Linux server via systemd timer (hourly execution).

### Initial Server Setup (One-Time)

```bash
# Set deployment credentials
export DEPLOY_USER=your-username
export DEPLOY_HOST=your-server-ip
export DEPLOY_PORT=22

# Generate and upload SSH key
task ssh-keygen

# Configure passwordless sudo (will ask for password once)
task setup-sudo

# Configure log retention (7 days)
task setup-logging
```

### Deploy Application

```bash
# Build and deploy to remote server
task deploy
```

This will:
1. Build the binary for Linux
2. Copy binary, `.env`, `user_config.toml`, and `service-account.json` to server
3. Create systemd service and timer
4. Enable hourly execution

### Manage Deployment

```bash
# Check scheduler status
task status

# View logs (last 50 lines)
task logs

# View more logs
task logs LINES=200

# Test run scheduler once
task test-run

# Stop scheduler
task stop

# Clean up deployment
task clean
```

## How It Works

1. **Hourly Timer**: Systemd timer triggers the scheduler at the start of every hour
2. **Fetch Events**: Queries Google Calendar for events starting in the next hour
3. **VM Matching**: Matches calendar events with available VMs
4. **Provisioning**:
   - Powers on VM if off
   - Restores to specified snapshot (or latest)
   - Generates random password for VM user
   - Updates calendar event description with VM info and credentials
5. **Database**: Tracks bookings in SQLite to avoid duplicate processing

## Project Structure

```
.
├── api/                          # Go application
│   ├── cmd/server/              # Main entry point
│   ├── internal/
│   │   ├── config/              # Configuration management
│   │   ├── logger/              # Structured logging
│   │   ├── models/              # Data models (VM, Booking)
│   │   ├── service/             # Business logic
│   │   │   ├── booking.go       # Booking management
│   │   │   ├── calendar.go     # Google Calendar integration
│   │   │   ├── vmware.go        # VMware vSphere integration
│   │   │   └── password.go     # Password generation
│   │   └── store/               # SQLite persistence
│   ├── data/
│   │   └── user_config.toml     # User configuration
│   └── go.mod                   # Go dependencies
│
├── infra/
│   └── terraform/               # Infrastructure as Code
│       └── main.tf              # Google Cloud resources
│
├── .devcontainer/               # VS Code dev container config
├── Taskfile.yml                 # Task automation
└── README.md                    # This file
```

## Development

### Running Tests

```bash
cd api
go test ./...

# With coverage
go test -cover ./...

# Specific package
go test ./internal/config -v
```

### Available Tasks

```bash
# List all available tasks
task --list

# Common tasks:
task build          # Build the binary
task deploy         # Deploy to remote server
task status         # Check deployment status
task logs           # View remote logs
task test-run       # Run scheduler once on server
```

### Pre-commit Hooks

This project uses pre-commit hooks for code quality:

```bash
# Install hooks
pre-commit install

# Run manually
pre-commit run --all-files
```

## Configuration Reference

### Environment Variables (`.env`)

| Variable | Description | Required | Default |
|----------|-------------|----------|---------|
| `VCENTER_URL` | vSphere/ESXi server URL | Yes | - |
| `VCENTER_USERNAME` | vSphere username | Yes | - |
| `VCENTER_PASSWORD` | vSphere password | Yes | - |
| `VCENTER_INSECURE` | Skip TLS verification | No | `false` |
| `DB_PATH` | SQLite database path | No | `./scheduler.db` |
| `LOG_LEVEL` | Logging level | No | `info` |
| `CONFIG_PATH` | Path to user_config.toml | No | `./api/data/user_config.toml` |

### User Config (`user_config.toml`)

| Section | Key | Description |
|---------|-----|-------------|
| `[calendar]` | `calendar_id` | Google Calendar ID |
| | `service_account_path` | Path to service account JSON |
| `[vsphere]` | `vms` | List of VM names available for labs |
| | `users` | List of VM usernames for password resets |
| | `snapshot_name` | Optional: specific snapshot to restore |

## Troubleshooting

### Service Account Already Exists Error

If you get `Error 409: Service account calendar-service-account already exists`:

```bash
cd infra/terraform

# Import the existing service account
tofu import google_service_account.calendar_sa projects/exsi-chat-app-478319/serviceAccounts/calendar-service-account@exsi-chat-app-478319.iam.gserviceaccount.com

# Then apply to create the key
tofu apply
```

### Service Account Authentication Fails

```bash
# Verify service account key format
cat service-account.json | jq .

# Re-authenticate and regenerate
cd infra/terraform
tofu destroy -target=google_service_account_key.calendar_sa_key
tofu apply
tofu output -raw service_account_key > service-account.json
```

### Calendar Events Not Syncing

1. Verify service account has calendar access:
   - Check calendar sharing settings
   - Ensure "Make changes to events" permission
2. Check calendar ID in `user_config.toml` matches
3. Review logs: `task logs`

### VM Operations Failing

```bash
# Test vCenter connectivity
cd api
go run cmd/server/main.go
```

Check logs for authentication or permission errors.

### Deployment Issues

```bash
# Test SSH connection
ssh -p $DEPLOY_PORT $DEPLOY_USER@$DEPLOY_HOST

# Check systemd status
task status

# View detailed logs
task logs LINES=100

# Manual service restart
ssh $DEPLOY_USER@$DEPLOY_HOST "sudo systemctl restart esxi-lab-scheduler.timer"
```

## Security Considerations

- ⚠️ **Never commit** `service-account.json`, `.env`, or credentials
- 🔒 Use `VCENTER_INSECURE=false` in production
- 🔑 Rotate service account keys periodically
- 📝 Use limited vSphere user with only required permissions
- 🔐 Store `.env` and credentials securely on deployment server

## License

MIT License - feel free to use, modify, and distribute this project for any purpose.

## Contributing

Contributions are welcome! Feel free to:

- Open issues for bugs or feature requests
- Submit pull requests with improvements
- Fork and modify for your own use
- Share feedback and suggestions

No formal process required - just submit a PR or open an issue.
