variable "gcp_project" {
  type = string
}

variable "lab_env_secret_id" {
  type = string
}

variable "opnsense_url" {
  type = string
}

variable "wireguard_server_name" {
  type = string
}

variable "wireguard_server_public_key" {
  type = string
}

variable "wireguard_server_port" {
  type = number
}

variable "wireguard_server_mtu" {
  type = number
}

variable "wireguard_server_dns" {
  type = list(string)
}

variable "wireguard_server_tunnel_address" {
  type = string
}

variable "wireguard_existing_peer_ids" {
  type = list(string)
}

variable "wireguard_public_endpoint" {
  type = string
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
