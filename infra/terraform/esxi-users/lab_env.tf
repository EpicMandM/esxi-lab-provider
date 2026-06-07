module "lab" {
  source = "../modules/lab"

  gcp_project = data.terraform_remote_state.gcloud.outputs.gcp_project

  keys = ["ESXI_PASSWORD"]
  required_keys = [
    "ESXI_PASSWORD",
  ]
}

locals {
  esxi_url            = data.terraform_remote_state.lab.outputs.esxi_url
  esxi_admin_username = data.terraform_remote_state.lab.outputs.esxi_admin_username
  esxi_admin_password = module.lab.secrets["ESXI_PASSWORD"]
}
