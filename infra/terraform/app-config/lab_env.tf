module "lab" {
  source = "../modules/lab"

  gcp_project = var.gcp_project
  secret_id   = var.lab_env_secret_id

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
