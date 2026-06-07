module "lab" {
  source = "../modules/lab"

  gcp_project = data.terraform_remote_state.gcloud.outputs.gcp_project

  keys = [
    "WIREGUARD_SERVER_PRIVATE_KEY",
    "OPNSENSE_API_KEY",
    "OPNSENSE_API_SECRET",
  ]
  required_keys = [
    "WIREGUARD_SERVER_PRIVATE_KEY",
  ]
}

data "external" "wireguard_server_public_key" {
  program = ["python3", "${path.module}/../../../scripts/wireguard-pubkey.py"]

  query = {
    private_key         = module.lab.secrets["WIREGUARD_SERVER_PRIVATE_KEY"]
    opnsense_url        = var.opnsense_url
    opnsense_api_key    = module.lab.secrets["OPNSENSE_API_KEY"]
    opnsense_api_secret = module.lab.secrets["OPNSENSE_API_SECRET"]
    server_name         = var.wireguard_server_name
  }
}

locals {
  wireguard_server_public_key_override = (
    var.wireguard_server_public_key != "" &&
    var.wireguard_server_public_key != "server-public-key-here"
  ) ? var.wireguard_server_public_key : ""

  wireguard_server_public_key = local.wireguard_server_public_key_override != "" ? local.wireguard_server_public_key_override : data.external.wireguard_server_public_key.result.public_key
}
