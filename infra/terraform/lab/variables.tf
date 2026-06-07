variable "esxi_url" {
  type = string
}

variable "esxi_admin_username" {
  type = string
}

variable "opnsense_url" {
  type = string
}

variable "smtp_username" {
  type    = string
  default = ""
}

variable "smtp_from" {
  type    = string
  default = ""
}

variable "wireguard_server_name" {
  type = string
}

variable "wireguard_server_public_key" {
  type        = string
  default     = ""
  description = "Optional override. When empty, derived from WIREGUARD_SERVER_PRIVATE_KEY or read from OPNsense."
}

variable "wireguard_server_port" {
  type = number
}

variable "wireguard_public_endpoint" {
  type = string
}

variable "wireguard_server_tunnel_address" {
  type = string
}

variable "wireguard_server_mtu" {
  type = number
}

variable "wireguard_server_dns" {
  type = list(string)
}

variable "wireguard_existing_peer_ids" {
  type    = list(string)
  default = []
}

variable "peer1_tunnel_address" {
  type = string
}

variable "peer2_tunnel_address" {
  type = string
}

variable "peer3_tunnel_address" {
  type = string
}

variable "peer4_tunnel_address" {
  type = string
}
