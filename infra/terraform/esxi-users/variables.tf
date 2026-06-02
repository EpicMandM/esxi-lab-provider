variable "gcp_project" {
  type = string
}

variable "lab_env_secret_id" {
  type = string
}

variable "esxi_url" {
  type = string
}

variable "esxi_admin_username" {
  type = string
}

variable "user_count" {
  type    = number
  default = 4
}

variable "fortigate_pod_indices" {
  type    = list(number)
  default = [1, 2, 3]
}

variable "role_name" {
  type    = string
  default = "lab-console"
}
