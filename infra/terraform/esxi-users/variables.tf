variable "esxi_url" {
  description = "ESXi host URL (e.g., https://192.168.1.10)"
  type        = string
}

variable "esxi_admin_username" {
  description = "ESXi admin username (must have permission to manage users/roles)"
  type        = string
  default     = "root"
}

variable "esxi_admin_password" {
  description = "ESXi admin password"
  type        = string
  sensitive   = true
}

variable "user_count" {
  description = "Number of lab users to provision (creates lab-user-1..N with Pod-1..N_FortiGate access)"
  type        = number
  default     = 4
}

variable "role_name" {
  description = "Name of the ESXi role for lab console access"
  type        = string
  default     = "lab-console"
}
