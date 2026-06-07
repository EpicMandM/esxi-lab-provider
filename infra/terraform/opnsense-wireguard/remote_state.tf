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

data "terraform_remote_state" "lab" {
  backend = "gcs"
  config = {
    bucket = local.terraform_state_bucket
    prefix = "lab"
  }
}
