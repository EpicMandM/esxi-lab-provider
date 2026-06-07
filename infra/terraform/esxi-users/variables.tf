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
