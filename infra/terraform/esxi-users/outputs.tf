output "user_credentials" {
  description = "Credentials for each lab user (use to distribute access)"
  sensitive   = true
  value = {
    for name, user in local.users : name => {
      username      = name
      password      = user.password
      vm_fortigate  = user.vm_fortigate
      vm_client_deb = user.vm_client_deb
    }
  }
}

output "esxi_url" {
  description = "ESXi host URL"
  value       = var.esxi_url
}

output "user_vm_mappings" {
  description = "Map of lab user names to their assigned VMs"
  value = {
    for name, user in local.users : name => {
      fortigate  = user.vm_fortigate
      client_deb = user.vm_client_deb
    }
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
