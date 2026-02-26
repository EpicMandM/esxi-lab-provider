terraform {
  # renovate: datasource=github-releases depName=opentofu/opentofu
  required_version = ">= 1.6.0"
  required_providers {
    local = {
      source  = "hashicorp/local"
      version = "~> 2.5"
    }
  }
}

# ──────────────────────────────────────────────
# Read outputs from sibling modules' state files
# ──────────────────────────────────────────────

data "terraform_remote_state" "esxi_users" {
  backend = "local"
  config = {
    path = "${path.module}/../esxi-users/terraform.tfstate"
  }
}

data "terraform_remote_state" "wireguard" {
  backend = "local"
  config = {
    path = "${path.module}/../opnsense-wireguard/terraform.tfstate"
  }
}

# ──────────────────────────────────────────────
# Generate user_config.toml from all infrastructure outputs
# ──────────────────────────────────────────────
# This module is the single source of truth for the application's
# feature configuration. It consumes outputs from:
#   - esxi-users:         ESXi URL + user/VM mappings
#   - opnsense-wireguard: WireGuard server config + peer addresses

resource "local_file" "user_config" {
  filename        = "${path.module}/../../../api/data/user_config.toml"
  file_permission = "0644"

  content = templatefile("${path.module}/templates/user_config.toml.tftpl", {
    # Calendar
    calendar_id          = var.calendar_id
    service_account_path = var.service_account_path

    # ESXi (from esxi-users state)
    esxi_url              = data.terraform_remote_state.esxi_users.outputs.esxi_url
    esxi_user_vm_mappings = data.terraform_remote_state.esxi_users.outputs.user_vm_mappings
    esxi_snapshot_name    = var.esxi_snapshot_name

    # WireGuard (from opnsense-wireguard state)
    wg_server_public_key     = data.terraform_remote_state.wireguard.outputs.server_public_key
    wg_server_endpoint       = data.terraform_remote_state.wireguard.outputs.server_endpoint
    wg_opnsense_url          = data.terraform_remote_state.wireguard.outputs.opnsense_url
    wg_server_tunnel_network = var.wg_server_tunnel_network
    wg_allowed_ips           = var.wg_allowed_ips
    wg_mtu                   = var.wg_client_mtu
    wg_client_addresses      = data.terraform_remote_state.wireguard.outputs.peer_tunnel_addresses
    wg_keepalive             = var.wg_keepalive
    wg_opnsense_insecure     = var.wg_opnsense_insecure
  })
}
