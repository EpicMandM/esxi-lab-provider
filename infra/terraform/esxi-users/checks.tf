check "esxi_reachable_config" {
  assert {
    condition     = !can(regex("(?i)example\\.com", local.esxi_url))
    error_message = <<-EOT
      ESXi URL in lab remote state is still a template (example.com).
      Set esxi_url in infra/terraform/lab/lab.auto.tfvars and run: task infra:lab
    EOT
  }
}
