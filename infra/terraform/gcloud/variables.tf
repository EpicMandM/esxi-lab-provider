variable "gcp_region" {
  type    = string
  default = "europe-west1"
}

variable "terraform_state_bucket" {
  type    = string
  default = "exsi-chat-app-478319-terraform-state"
}

variable "cost_guard_schedule" {
  type        = string
  description = "Cron schedule for the free-tier cost guard Cloud Function (UTC)"
  default     = "0 5 * * *"
}

variable "cost_guard_keep_secret_versions" {
  type        = number
  description = "Non-destroyed Secret Manager versions to retain per secret"
  default     = 1
}
