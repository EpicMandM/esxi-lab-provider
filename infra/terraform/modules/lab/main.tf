terraform {
  required_version = ">= 1.6.0"
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 7.35.0"
    }
  }
}

data "google_secret_manager_secret_version" "lab_env" {
  project = var.gcp_project
  secret  = var.secret_id
}

locals {
  env_lines = [
    for line in split("\n", data.google_secret_manager_secret_version.lab_env.secret_data) :
    trimspace(line)
    if trimspace(line) != "" && !startswith(trimspace(line), "#")
  ]

  env = {
    for line in local.env_lines :
    trimspace(regex("^([^=]+)=(.*)$", line)[0]) => trimspace(regex("^([^=]+)=(.*)$", line)[1])
    if can(regex("^[^=]+=", line))
  }

  secrets = { for key in var.keys : key => try(local.env[key], "") }

  missing_secrets = [
    for key in var.required_keys : key
    if !contains(keys(local.env), key) || trimspace(try(local.env[key], "")) == ""
  ]
}

check "required_secrets" {
  assert {
    condition     = length(local.missing_secrets) == 0
    error_message = "Missing Secret Manager credentials: ${join(", ", local.missing_secrets)}. Fill secrets.env and run task secrets:push."
  }
}
