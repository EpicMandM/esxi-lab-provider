check "lab_site_config" {
  assert {
    condition     = !can(regex("(?i)example\\.com", var.esxi_url))
    error_message = <<-EOT
      esxi_url still uses a template host (example.com).
      Run task infra:init to create a local override, or fix remote lab state.
      Devcontainer via WireGuard tunnel: esxi_url = "https://127.0.0.1:10443" (run task tunnel first).
    EOT
  }

  assert {
    condition     = local.wireguard_server_public_key != "" && can(regex("^[A-Za-z0-9+/]{43}=$", local.wireguard_server_public_key))
    error_message = "WireGuard server public key could not be resolved. Ensure WIREGUARD_SERVER_PRIVATE_KEY is in Secret Manager, or set wireguard_server_public_key via task infra:init."
  }
}
