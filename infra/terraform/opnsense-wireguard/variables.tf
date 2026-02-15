# ──────────────────────────────────────────────
# OPNsense Connection
# ──────────────────────────────────────────────

variable "opnsense_url" {
  description = "OPNsense firewall URL (e.g., https://192.168.1.1)"
  type        = string
}

variable "opnsense_api_key" {
  description = "OPNsense API key"
  type        = string
  sensitive   = true
}

variable "opnsense_api_secret" {
  description = "OPNsense API secret"
  type        = string
  sensitive   = true
}

# ──────────────────────────────────────────────
# WireGuard Server
# ──────────────────────────────────────────────

variable "wireguard_server_name" {
  description = "Existing WireGuard server name"
  type        = string
}

variable "wireguard_server_public_key" {
  description = "WireGuard server public key"
  type        = string
}

variable "wireguard_server_private_key" {
  description = "WireGuard server private key"
  type        = string
  sensitive   = true
}

variable "wireguard_server_port" {
  description = "WireGuard server listening port"
  type        = number
}

variable "wireguard_server_mtu" {
  description = "WireGuard server MTU (0 for default, -1 for interface MTU)"
  type        = number
  default     = 0
}

variable "wireguard_server_dns" {
  description = "DNS servers for WireGuard clients"
  type        = list(string)
  default     = []
}

variable "wireguard_server_tunnel_address" {
  description = "WireGuard server tunnel address (e.g., 172.17.18.1/24)"
  type        = string
}

variable "wireguard_existing_peer_ids" {
  description = "Existing WireGuard peer UUIDs to keep attached to the server"
  type        = list(string)
  default     = []
}

# ──────────────────────────────────────────────
# WireGuard Peers (tunnel addresses only — keys managed by Go)
# ──────────────────────────────────────────────

variable "wireguard_public_endpoint" {
  description = "Public endpoint (ip:port) that WireGuard clients connect to (may differ from OPNsense management URL)"
  type        = string
}

variable "peer1_tunnel_address" {
  description = "Tunnel address for peer 1"
  type        = string
  default     = "172.17.18.101/32"
}

variable "peer2_tunnel_address" {
  description = "Tunnel address for peer 2"
  type        = string
  default     = "172.17.18.102/32"
}

variable "peer3_tunnel_address" {
  description = "Tunnel address for peer 3"
  type        = string
  default     = "172.17.18.103/32"
}

variable "peer4_tunnel_address" {
  description = "Tunnel address for peer 4"
  type        = string
  default     = "172.17.18.104/32"
}
