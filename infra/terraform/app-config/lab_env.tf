module "lab" {
  source = "../modules/lab"

  gcp_project = data.terraform_remote_state.gcloud.outputs.gcp_project

  keys = [
    "ESXI_PASSWORD",
    "OPNSENSE_API_KEY",
    "OPNSENSE_API_SECRET",
    "SMTP_PASSWORD",
  ]
  required_keys = [
    "ESXI_PASSWORD",
    "OPNSENSE_API_KEY",
    "OPNSENSE_API_SECRET",
  ]
}
