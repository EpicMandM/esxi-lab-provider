variable "gcp_project" {
  type = string
}

variable "gcp_region" {
  type    = string
  default = "europe-west1"
}

variable "terraform_state_bucket" {
  type    = string
  default = "exsi-chat-app-478319-terraform-state"
}

variable "lab_env_secret_id" {
  type = string
}
