module "lab" {
  source = "../modules/lab"

  gcp_project = data.terraform_remote_state.gcloud.outputs.gcp_project

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
  lab                             = data.terraform_remote_state.lab.outputs
  opnsense_url                    = local.lab.opnsense_url
  opnsense_api_key                = module.lab.secrets["OPNSENSE_API_KEY"]
  opnsense_api_secret             = module.lab.secrets["OPNSENSE_API_SECRET"]
  wireguard_server_name           = local.lab.wireguard_server_name
  wireguard_server_public_key     = local.lab.wireguard_server_public_key
  wireguard_server_private_key    = module.lab.secrets["WIREGUARD_SERVER_PRIVATE_KEY"]
  wireguard_server_port           = local.lab.wireguard_server_port
  wireguard_public_endpoint       = local.lab.wireguard_public_endpoint
  wireguard_server_tunnel_address = local.lab.wireguard_server_tunnel_address
  wireguard_server_mtu            = local.lab.wireguard_server_mtu
  wireguard_server_dns            = local.lab.wireguard_server_dns
  wireguard_existing_peer_ids     = local.lab.wireguard_existing_peer_ids
  peer1_tunnel_address            = local.lab.peer1_tunnel_address
  peer2_tunnel_address            = local.lab.peer2_tunnel_address
  peer3_tunnel_address            = local.lab.peer3_tunnel_address
  peer4_tunnel_address            = local.lab.peer4_tunnel_address
}
