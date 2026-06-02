terraform {
  required_version = ">= 1.6.0"
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 7.34.0"
    }
    local = {
      source  = "hashicorp/local"
      version = "~> 2.5"
    }
  }
}

provider "google" {
  project = var.gcp_project
}

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

locals {
  esxi_user_vm_mappings_normalized = {
    for username, prefixes in data.terraform_remote_state.esxi_users.outputs.user_vm_mappings : username =>
    try(tolist(prefixes), [tostring(prefixes)])
  }

  esxi_url_raw = data.terraform_remote_state.esxi_users.outputs.esxi_url
  esxi_url     = endswith(local.esxi_url_raw, "/sdk") ? local.esxi_url_raw : "${trimsuffix(local.esxi_url_raw, "/")}/sdk"
}

resource "local_file" "user_config" {
  filename        = "${path.module}/../../../api/data/user_config.toml"
  file_permission = "0644"

  content = templatefile("${path.module}/templates/user_config.toml.tftpl", {
    calendar_id          = var.calendar_id
    service_account_path = var.service_account_path

    esxi_url              = data.terraform_remote_state.esxi_users.outputs.esxi_url
    esxi_user_vm_mappings = local.esxi_user_vm_mappings_normalized
    esxi_snapshot_name    = var.esxi_snapshot_name

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

resource "local_file" "env" {
  filename        = "${path.module}/../../../.env"
  file_permission = "0600"

  content = templatefile("${path.module}/templates/.env.tftpl", {
    esxi_url            = local.esxi_url
    esxi_username       = data.terraform_remote_state.esxi_users.outputs.esxi_admin_username
    esxi_password       = module.lab.secrets["ESXI_PASSWORD"]
    esxi_insecure       = var.esxi_insecure
    opnsense_api_key    = module.lab.secrets["OPNSENSE_API_KEY"]
    opnsense_api_secret = module.lab.secrets["OPNSENSE_API_SECRET"]
    smtp_host           = var.smtp_host
    smtp_port           = var.smtp_port
    smtp_username       = var.smtp_username
    smtp_password       = lookup(module.lab.secrets, "SMTP_PASSWORD", "")
    smtp_from           = var.smtp_from != "" ? var.smtp_from : var.smtp_username
  })

  lifecycle {
    precondition {
      condition     = var.smtp_username == "" || lookup(module.lab.secrets, "SMTP_PASSWORD", "") != ""
      error_message = "SMTP_USERNAME is set but SMTP_PASSWORD is missing from Secret Manager."
    }
  }
}
