module "lab" {
  source = "../modules/lab"

  gcp_project = var.gcp_project
  secret_id   = var.lab_env_secret_id

  keys = ["ESXI_PASSWORD"]
  required_keys = [
    "ESXI_PASSWORD",
  ]
}

locals {
  esxi_url            = var.esxi_url
  esxi_admin_username = var.esxi_admin_username
  esxi_admin_password = module.lab.secrets["ESXI_PASSWORD"]
}
