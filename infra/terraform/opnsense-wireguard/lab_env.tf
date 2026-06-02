module "lab" {
  source = "../modules/lab"

  gcp_project = var.gcp_project
  secret_id   = var.lab_env_secret_id

  keys = [
    "OPNSENSE_API_KEY",
    "OPNSENSE_API_SECRET",
    "WIREGUARD_SERVER_PRIVATE_KEY",
  ]
  required_keys = [
    "OPNSENSE_API_KEY",
    "OPNSENSE_API_SECRET",
    "WIREGUARD_SERVER_PRIVATE_KEY",
  ]
}

locals {
  opnsense_url                    = var.opnsense_url
  opnsense_api_key                = module.lab.secrets["OPNSENSE_API_KEY"]
  opnsense_api_secret             = module.lab.secrets["OPNSENSE_API_SECRET"]
  wireguard_server_name           = var.wireguard_server_name
  wireguard_server_public_key     = var.wireguard_server_public_key
  wireguard_server_private_key    = module.lab.secrets["WIREGUARD_SERVER_PRIVATE_KEY"]
  wireguard_server_port           = var.wireguard_server_port
  wireguard_public_endpoint       = var.wireguard_public_endpoint
  wireguard_server_tunnel_address = var.wireguard_server_tunnel_address
  wireguard_server_mtu            = var.wireguard_server_mtu
  wireguard_server_dns            = var.wireguard_server_dns
  wireguard_existing_peer_ids     = var.wireguard_existing_peer_ids
  peer1_tunnel_address            = var.peer1_tunnel_address
  peer2_tunnel_address            = var.peer2_tunnel_address
  peer3_tunnel_address            = var.peer3_tunnel_address
  peer4_tunnel_address            = var.peer4_tunnel_address
}
