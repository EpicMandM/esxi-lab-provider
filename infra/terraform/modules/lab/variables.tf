variable "gcp_project" {
  type = string
}

variable "keys" {
  type = list(string)
}

variable "required_keys" {
  type = list(string)
}
