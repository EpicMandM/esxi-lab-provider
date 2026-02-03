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

# WireGuard Server Configuration
variable "wireguard_port" {
  description = "WireGuard server listening port"
  type        = number
  default     = 51820
}

variable "wireguard_mtu" {
  description = "WireGuard MTU size"
  type        = number
  default     = 1420
}

variable "wireguard_dns" {
  description = "DNS servers for WireGuard clients"
  type        = list(string)
  default     = ["1.1.1.1", "8.8.8.8"]
}

variable "server_tunnel_address" {
  description = "WireGuard server tunnel address (e.g., 10.10.10.1/24)"
  type        = string
  default     = "10.10.10.1/24"
}

variable "server_public_key" {
  description = "WireGuard server public key"
  type        = string
}

variable "server_private_key" {
  description = "WireGuard server private key"
  type        = string
  sensitive   = true
}

# Peer 1 Configuration
variable "peer1_public_key" {
  description = "Public key for peer 1"
  type        = string
}

variable "peer1_preshared_key" {
  description = "Pre-shared key for peer 1 (optional)"
  type        = string
  default     = ""
  sensitive   = true
}

variable "peer1_allowed_ips" {
  description = "Allowed IPs for peer 1 (tunnel address)"
  type        = string
  default     = "10.10.10.2/32"
}

variable "peer1_server_address" {
  description = "Server endpoint address for peer 1"
  type        = string
  default     = ""
}

variable "peer1_server_port" {
  description = "Server endpoint port for peer 1"
  type        = number
  default     = -1
}

# Peer 2 Configuration
variable "peer2_public_key" {
  description = "Public key for peer 2"
  type        = string
}

variable "peer2_preshared_key" {
  description = "Pre-shared key for peer 2 (optional)"
  type        = string
  default     = ""
  sensitive   = true
}

variable "peer2_allowed_ips" {
  description = "Allowed IPs for peer 2 (tunnel address)"
  type        = string
  default     = "10.10.10.3/32"
}

variable "peer2_server_address" {
  description = "Server endpoint address for peer 2"
  type        = string
  default     = ""
}

variable "peer2_server_port" {
  description = "Server endpoint port for peer 2"
  type        = number
  default     = -1
}

# Peer 3 Configuration
variable "peer3_public_key" {
  description = "Public key for peer 3"
  type        = string
}

variable "peer3_preshared_key" {
  description = "Pre-shared key for peer 3 (optional)"
  type        = string
  default     = ""
  sensitive   = true
}

variable "peer3_allowed_ips" {
  description = "Allowed IPs for peer 3 (tunnel address)"
  type        = string
  default     = "10.10.10.4/32"
}

variable "peer3_server_address" {
  description = "Server endpoint address for peer 3"
  type        = string
  default     = ""
}

variable "peer3_server_port" {
  description = "Server endpoint port for peer 3"
  type        = number
  default     = -1
}
