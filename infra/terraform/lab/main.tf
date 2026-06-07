terraform {
  # renovate: datasource=github-releases depName=opentofu/opentofu
  required_version = ">= 1.6.0"
  required_providers {
    external = {
      source  = "hashicorp/external"
      version = "~> 2.3"
    }
  }

  backend "gcs" {
    bucket = "exsi-chat-app-478319-terraform-state"
    prefix = "lab"
  }
}

locals {
  terraform_state_bucket = "exsi-chat-app-478319-terraform-state"
}

data "terraform_remote_state" "gcloud" {
  backend = "gcs"
  config = {
    bucket = local.terraform_state_bucket
    prefix = "gcloud"
  }
}
