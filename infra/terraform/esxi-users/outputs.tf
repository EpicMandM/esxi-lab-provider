output "user_credentials" {
  description = "Credentials for each lab user (use to distribute access)"
  sensitive   = true
  value = {
    for name, user in local.users : name => {
      username  = name
      password  = user.password
      vm_prefix = user.vm_prefix
    }
  }
}

output "esxi_url" {
  description = "ESXi host URL"
  value       = var.esxi_url
}

output "user_vm_mappings" {
  description = <<-EOT
    Map of lab user names to a list of VM name prefixes ([]string).
    Rendered into user_config.toml as TOML arrays (required by LoadFeatureConfig), e.g.
    "lab-user-1" = ["Pod-1_"]. All VMs whose names start with a prefix belong to that user.
  EOT
  value = {
    for name, user in local.users : name => [user.vm_prefix]
  }
}

output "user_count" {
  description = "Number of lab users provisioned"
  value       = var.user_count
}

output "role_name" {
  description = "The ESXi role created for lab users"
  value       = var.role_name
}

output "role_privileges" {
  description = "Privileges assigned to the lab-console role"
  value       = local.role_privileges
}
