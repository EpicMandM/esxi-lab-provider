terraform {
  required_version = ">= 1.6.0"
  required_providers {
    opnsense = {
      source  = "browningluke/opnsense"
      version = "0.16.1"
    }
  }
}

provider "opnsense" {
  uri        = var.opnsense_url
  api_key    = var.opnsense_api_key
  api_secret = var.opnsense_api_secret
}

# Peer 1 - Development Client
resource "opnsense_wireguard_client" "peer1" {
  enabled        = true
  name           = "dev-client-1"
  public_key     = var.peer1_public_key
  psk            = var.peer1_preshared_key
  tunnel_address = [var.peer1_allowed_ips]
  server_address = var.peer1_server_address
  server_port    = var.peer1_server_port
  keep_alive     = 25
}

# Peer 2 - Development Client
resource "opnsense_wireguard_client" "peer2" {
  enabled        = true
  name           = "dev-client-2"
  public_key     = var.peer2_public_key
  psk            = var.peer2_preshared_key
  tunnel_address = [var.peer2_allowed_ips]
  server_address = var.peer2_server_address
  server_port    = var.peer2_server_port
  keep_alive     = 25
}

# Peer 3 - Development Client
resource "opnsense_wireguard_client" "peer3" {
  enabled        = true
  name           = "dev-client-3"
  public_key     = var.peer3_public_key
  psk            = var.peer3_preshared_key
  tunnel_address = [var.peer3_allowed_ips]
  server_address = var.peer3_server_address
  server_port    = var.peer3_server_port
  keep_alive     = 25
}

# Create WireGuard server instance
resource "opnsense_wireguard_server" "dev_server" {
  enabled = true
  name    = "wg-dev-server"

  private_key = var.server_private_key
  public_key  = var.server_public_key

  port = var.wireguard_port
  mtu  = var.wireguard_mtu

  dns = var.wireguard_dns

  tunnel_address = [var.server_tunnel_address]

  peers = [
    opnsense_wireguard_client.peer1.id,
    opnsense_wireguard_client.peer2.id,
    opnsense_wireguard_client.peer3.id,
  ]
}


# Allow WireGuard traffic on WAN
resource "opnsense_firewall_filter" "allow_wireguard" {
  enabled     = true
  description = "Allow WireGuard Inbound"

  interface = {
    interface = ["wan", "lan"]
  }

  filter = {
    action    = "pass"
    direction = "in"
    protocol  = "UDP"

    source = {
      net = "any"
    }

    destination = {
      port = tostring(var.wireguard_port)
    }
  }
}
