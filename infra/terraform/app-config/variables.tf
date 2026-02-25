# ──────────────────────────────────────────────
# Calendar
# ──────────────────────────────────────────────

variable "calendar_id" {
  description = "Google Calendar ID"
  type        = string
  default     = "c_c013620c0bccde816a0f74cdc7304179ed5b804b670ca28a7fe5b10cf3d69b48@group.calendar.google.com"
}

variable "service_account_path" {
  description = "Path to the service account JSON inside the container"
  type        = string
  default     = "/app/config/service-account.json"
}

# ──────────────────────────────────────────────
# ESXi overrides
# ──────────────────────────────────────────────

variable "esxi_snapshot_name" {
  description = "Snapshot name to restore VMs to (empty = use latest)"
  type        = string
  default     = ""
}

# ──────────────────────────────────────────────
# WireGuard client settings (not from state)
# ──────────────────────────────────────────────

variable "wg_server_tunnel_network" {
  description = "WireGuard tunnel network CIDR"
  type        = string
  default     = "172.17.18.0/24"
}

variable "wg_allowed_ips" {
  description = "Additional networks accessible through the WireGuard tunnel"
  type        = list(string)
  default     = ["172.17.17.0/24"]
}

variable "wg_client_mtu" {
  description = "MTU setting for WireGuard client interfaces"
  type        = number
  default     = 1380
}

variable "wg_keepalive" {
  description = "Persistent keepalive interval in seconds (0 to disable)"
  type        = number
  default     = 0
}

variable "wg_opnsense_insecure" {
  description = "Skip TLS certificate verification for the OPNsense API (required for self-signed certs on IP addresses)"
  type        = bool
  default     = true
}
