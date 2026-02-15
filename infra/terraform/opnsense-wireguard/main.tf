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
  uri            = var.opnsense_url
  api_key        = var.opnsense_api_key
  api_secret     = var.opnsense_api_secret
  allow_insecure = true
}

locals {
  # Unique placeholder keys used when provisioning peer slots.
  # OPNsense requires each peer to have a unique public key.
  # The Go application replaces these with real keys during rotation.
  placeholder_public_keys = {
    peer1 = "AQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
    peer2 = "AgAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
    peer3 = "AwAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
    peer4 = "BAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
  }
}

# ──────────────────────────────────────────────
# WireGuard Peers (provisioned as empty slots)
# ──────────────────────────────────────────────
# Terraform creates peer resources with a placeholder key.
# The Go application manages actual keys via OPNsense API rotation.

resource "opnsense_wireguard_client" "peer1" {
  enabled        = true
  name           = "lab-user-1"
  public_key     = local.placeholder_public_keys["peer1"]
  tunnel_address = [var.peer1_tunnel_address]

  lifecycle {
    ignore_changes = [public_key]
  }
}

resource "opnsense_wireguard_client" "peer2" {
  enabled        = true
  name           = "lab-user-2"
  public_key     = local.placeholder_public_keys["peer2"]
  tunnel_address = [var.peer2_tunnel_address]

  lifecycle {
    ignore_changes = [public_key]
  }
}

resource "opnsense_wireguard_client" "peer3" {
  enabled        = true
  name           = "lab-user-3"
  public_key     = local.placeholder_public_keys["peer3"]
  tunnel_address = [var.peer3_tunnel_address]

  lifecycle {
    ignore_changes = [public_key]
  }
}

resource "opnsense_wireguard_client" "peer4" {
  enabled        = true
  name           = "lab-user-4"
  public_key     = local.placeholder_public_keys["peer4"]
  tunnel_address = [var.peer4_tunnel_address]

  lifecycle {
    ignore_changes = [public_key]
  }
}

# ──────────────────────────────────────────────
# WireGuard Server
# ──────────────────────────────────────────────

resource "opnsense_wireguard_server" "wg" {
  enabled     = true
  name        = var.wireguard_server_name
  private_key = var.wireguard_server_private_key
  public_key  = var.wireguard_server_public_key
  port        = var.wireguard_server_port
  mtu         = var.wireguard_server_mtu
  dns         = var.wireguard_server_dns

  tunnel_address = [var.wireguard_server_tunnel_address]

  peers = concat(
    var.wireguard_existing_peer_ids,
    [
      opnsense_wireguard_client.peer1.id,
      opnsense_wireguard_client.peer2.id,
      opnsense_wireguard_client.peer3.id,
      opnsense_wireguard_client.peer4.id,
    ]
  )
}
