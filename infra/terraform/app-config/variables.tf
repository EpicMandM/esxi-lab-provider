variable "gcp_project" {
  type = string
}

variable "lab_env_secret_id" {
  type = string
}

variable "esxi_insecure" {
  type    = bool
  default = true
}

variable "calendar_id" {
  type    = string
  default = "c_c013620c0bccde816a0f74cdc7304179ed5b804b670ca28a7fe5b10cf3d69b48@group.calendar.google.com"
}

variable "service_account_path" {
  type    = string
  default = "/app/config/service-account.json"
}

variable "esxi_snapshot_name" {
  type    = string
  default = ""
}

variable "wg_server_tunnel_network" {
  type    = string
  default = "172.17.18.0/24"
}

variable "wg_allowed_ips" {
  type    = list(string)
  default = ["172.17.17.0/24"]
}

variable "wg_client_mtu" {
  type    = number
  default = 1380
}

variable "wg_keepalive" {
  type    = number
  default = 0
}

variable "wg_opnsense_insecure" {
  type    = bool
  default = true
}

variable "smtp_host" {
  type    = string
  default = "smtp.gmail.com"
}

variable "smtp_port" {
  type    = string
  default = "587"
}

variable "smtp_username" {
  type    = string
  default = ""
}

variable "smtp_from" {
  type    = string
  default = ""
}
