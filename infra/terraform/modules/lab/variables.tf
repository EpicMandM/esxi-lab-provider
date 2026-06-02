variable "gcp_project" {
  type = string
}

variable "secret_id" {
  type = string
}

variable "keys" {
  type = list(string)
}

variable "required_keys" {
  type = list(string)
}
