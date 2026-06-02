terraform {
  # renovate: datasource=github-releases depName=opentofu/opentofu
  required_version = ">= 1.6.0"
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 7.34.0"
    }
    opnsense = {
      source  = "browningluke/opnsense"
      version = "0.19.0"
    }
  }
}

provider "google" {
  project = var.gcp_project
}

provider "opnsense" {
  uri            = local.opnsense_url
  api_key        = local.opnsense_api_key
  api_secret     = local.opnsense_api_secret
  allow_insecure = true
}

locals {
  placeholder_public_keys = {
    peer1 = "AQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
    peer2 = "AgAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
    peer3 = "AwAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
    peer4 = "BAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
  }
}

resource "opnsense_wireguard_client" "peer1" {
  enabled        = true
  name           = "lab-user-1"
  public_key     = local.placeholder_public_keys["peer1"]
  tunnel_address = [local.peer1_tunnel_address]

  lifecycle {
    ignore_changes = [public_key]
  }
}

resource "opnsense_wireguard_client" "peer2" {
  enabled        = true
  name           = "lab-user-2"
  public_key     = local.placeholder_public_keys["peer2"]
  tunnel_address = [local.peer2_tunnel_address]

  lifecycle {
    ignore_changes = [public_key]
  }
}

resource "opnsense_wireguard_client" "peer3" {
  enabled        = true
  name           = "lab-user-3"
  public_key     = local.placeholder_public_keys["peer3"]
  tunnel_address = [local.peer3_tunnel_address]

  lifecycle {
    ignore_changes = [public_key]
  }
}

resource "opnsense_wireguard_client" "peer4" {
  enabled        = true
  name           = "lab-user-4"
  public_key     = local.placeholder_public_keys["peer4"]
  tunnel_address = [local.peer4_tunnel_address]

  lifecycle {
    ignore_changes = [public_key]
  }
}

resource "opnsense_wireguard_server" "wg" {
  enabled     = true
  name        = local.wireguard_server_name
  private_key = local.wireguard_server_private_key
  public_key  = local.wireguard_server_public_key
  port        = local.wireguard_server_port
  mtu         = local.wireguard_server_mtu
  dns         = local.wireguard_server_dns

  tunnel_address = [local.wireguard_server_tunnel_address]

  peers = concat(
    local.wireguard_existing_peer_ids,
    [
      opnsense_wireguard_client.peer1.id,
      opnsense_wireguard_client.peer2.id,
      opnsense_wireguard_client.peer3.id,
      opnsense_wireguard_client.peer4.id,
    ]
  )
}
