terraform {
  required_version = ">= 1.6.0"
  required_providers {
    null = {
      source  = "hashicorp/null"
      version = "~> 3.2"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.6"
    }
  }
}

locals {
  govc_env = {
    GOVC_URL      = var.esxi_url
    GOVC_USERNAME = var.esxi_admin_username
    GOVC_PASSWORD = var.esxi_admin_password
    GOVC_INSECURE = "true"
  }

  # Build user -> VM mapping: lab-user-1 -> Pod-1_FortiGate, etc.
  users = {
    for i in range(1, var.user_count + 1) : "lab-user-${i}" => {
      index         = i
      vm_fortigate  = "Pod-${i}_FortiGate"
      vm_client_deb = "Pod-${i}_Client_Deb"
      password      = random_password.user_passwords[i - 1].result
    }
  }

  # ESXi privilege IDs for console access + VM power operations
  role_privileges = [
    "VirtualMachine.Interact.ConsoleInteract",
    "VirtualMachine.Interact.DeviceConnection",
    "VirtualMachine.Interact.PowerOff",
    "VirtualMachine.Interact.PowerOn",
    "VirtualMachine.Interact.Reset",
    "System.Anonymous",
    "System.Read",
    "System.View",
  ]
}

# ──────────────────────────────────────────────
# Generate random passwords for each user
# ──────────────────────────────────────────────

resource "random_password" "user_passwords" {
  count   = var.user_count
  length  = 16
  special = true
  # ESXi password complexity: at least 1 uppercase, 1 lowercase, 1 digit, 1 special
  min_upper   = 1
  min_lower   = 1
  min_numeric = 1
  min_special = 1
}

# ──────────────────────────────────────────────
# Create the "lab-console" role on ESXi
# ──────────────────────────────────────────────
# Grants only console interaction and power management privileges.

resource "null_resource" "esxi_role" {
  triggers = {
    privileges    = join(",", local.role_privileges)
    role_name     = var.role_name
    esxi_url      = var.esxi_url
    esxi_username = var.esxi_admin_username
    esxi_password = var.esxi_admin_password
  }

  lifecycle {
    ignore_changes = [
      triggers["esxi_url"],
      triggers["esxi_username"],
      triggers["esxi_password"],
    ]
  }

  provisioner "local-exec" {
    command     = <<-EOT
      # Create role, or update if it already exists
      govc role.create "${var.role_name}" ${join(" ", local.role_privileges)} 2>/dev/null || \
      govc role.update "${var.role_name}" ${join(" ", local.role_privileges)}
    EOT
    environment = local.govc_env
  }

  provisioner "local-exec" {
    when    = destroy
    command = <<-EOT
      govc role.remove "${self.triggers.role_name}" 2>/dev/null || true
    EOT
    environment = {
      GOVC_URL      = self.triggers.esxi_url
      GOVC_USERNAME = self.triggers.esxi_username
      GOVC_PASSWORD = self.triggers.esxi_password
      GOVC_INSECURE = "true"
    }
  }
}

# ──────────────────────────────────────────────
# Create local ESXi users
# ──────────────────────────────────────────────

resource "null_resource" "esxi_users" {
  for_each = local.users

  triggers = {
    username      = each.key
    password      = each.value.password
    esxi_url      = var.esxi_url
    esxi_username = var.esxi_admin_username
    esxi_password = var.esxi_admin_password
  }

  lifecycle {
    ignore_changes = [
      triggers["esxi_url"],
      triggers["esxi_username"],
      triggers["esxi_password"],
    ]
  }

  provisioner "local-exec" {
    command     = <<-EOT
      # Create user, or update password if user already exists
      govc host.account.create \
        -id "${each.key}" \
        -password "${each.value.password}" \
        -description "Lab user ${each.value.index} - ${each.value.vm_fortigate} + ${each.value.vm_client_deb} console access" 2>/dev/null || \
      govc host.account.update \
        -id "${each.key}" \
        -password "${each.value.password}" \
        -description "Lab user ${each.value.index} - ${each.value.vm_fortigate} + ${each.value.vm_client_deb} console access"
    EOT
    environment = local.govc_env
  }

  provisioner "local-exec" {
    when    = destroy
    command = <<-EOT
      govc host.account.remove -id "${self.triggers.username}" 2>/dev/null || true
    EOT
    environment = {
      GOVC_URL      = self.triggers.esxi_url
      GOVC_USERNAME = self.triggers.esxi_username
      GOVC_PASSWORD = self.triggers.esxi_password
      GOVC_INSECURE = "true"
    }
  }

  depends_on = [null_resource.esxi_role]
}

# ──────────────────────────────────────────────
# Assign per-VM permissions (user -> VM with role)
# ──────────────────────────────────────────────
# Each user gets the "lab-console" role on their FortiGate VM and Client_Deb VM (non-propagated).

resource "null_resource" "fortigate_permissions" {
  for_each = local.users

  triggers = {
    username      = each.key
    vm_name       = each.value.vm_fortigate
    role_name     = var.role_name
    esxi_url      = var.esxi_url
    esxi_username = var.esxi_admin_username
    esxi_password = var.esxi_admin_password
  }

  lifecycle {
    ignore_changes = [
      triggers["esxi_url"],
      triggers["esxi_username"],
      triggers["esxi_password"],
    ]
  }

  provisioner "local-exec" {
    command     = <<-EOT
      govc permissions.set \
        -principal "${each.key}" \
        -role "${var.role_name}" \
        -propagate=false \
        "/ha-datacenter/vm/${each.value.vm_fortigate}"
    EOT
    environment = local.govc_env
  }

  provisioner "local-exec" {
    when    = destroy
    command = <<-EOT
      govc permissions.remove \
        -principal "${self.triggers.username}" \
        "/ha-datacenter/vm/${self.triggers.vm_name}" 2>/dev/null || true
    EOT
    environment = {
      GOVC_URL      = self.triggers.esxi_url
      GOVC_USERNAME = self.triggers.esxi_username
      GOVC_PASSWORD = self.triggers.esxi_password
      GOVC_INSECURE = "true"
    }
  }

  depends_on = [
    null_resource.esxi_role,
    null_resource.esxi_users,
  ]
}

resource "null_resource" "client_deb_permissions" {
  for_each = local.users

  triggers = {
    username      = each.key
    vm_name       = each.value.vm_client_deb
    role_name     = var.role_name
    esxi_url      = var.esxi_url
    esxi_username = var.esxi_admin_username
    esxi_password = var.esxi_admin_password
  }

  lifecycle {
    ignore_changes = [
      triggers["esxi_url"],
      triggers["esxi_username"],
      triggers["esxi_password"],
    ]
  }

  provisioner "local-exec" {
    command     = <<-EOT
      govc permissions.set \
        -principal "${each.key}" \
        -role "${var.role_name}" \
        -propagate=false \
        "/ha-datacenter/vm/${each.value.vm_client_deb}"
    EOT
    environment = local.govc_env
  }

  provisioner "local-exec" {
    when    = destroy
    command = <<-EOT
      govc permissions.remove \
        -principal "${self.triggers.username}" \
        "/ha-datacenter/vm/${self.triggers.vm_name}" 2>/dev/null || true
    EOT
    environment = {
      GOVC_URL      = self.triggers.esxi_url
      GOVC_USERNAME = self.triggers.esxi_username
      GOVC_PASSWORD = self.triggers.esxi_password
      GOVC_INSECURE = "true"
    }
  }

  depends_on = [
    null_resource.esxi_role,
    null_resource.esxi_users,
  ]
}
