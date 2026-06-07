output "gcp_project" {
  description = "GCP project (from gcloud stack remote state)"
  value       = data.terraform_remote_state.gcloud.outputs.gcp_project
}

output "esxi_url" {
  value = var.esxi_url
}

output "esxi_admin_username" {
  value = var.esxi_admin_username
}

output "opnsense_url" {
  value = var.opnsense_url
}

output "smtp_username" {
  value = var.smtp_username
}

output "smtp_from" {
  value = var.smtp_from
}

output "wireguard_server_name" {
  value = var.wireguard_server_name
}

output "wireguard_server_public_key" {
  value       = local.wireguard_server_public_key
  description = "Derived from Secret Manager private key, OPNsense API, or lab.auto.tfvars override."
}

output "wireguard_server_port" {
  value = var.wireguard_server_port
}

output "wireguard_public_endpoint" {
  value = var.wireguard_public_endpoint
}

output "wireguard_server_tunnel_address" {
  value = var.wireguard_server_tunnel_address
}

output "wireguard_server_mtu" {
  value = var.wireguard_server_mtu
}

output "wireguard_server_dns" {
  value = var.wireguard_server_dns
}

output "wireguard_existing_peer_ids" {
  value = var.wireguard_existing_peer_ids
}

output "peer1_tunnel_address" {
  value = var.peer1_tunnel_address
}

output "peer2_tunnel_address" {
  value = var.peer2_tunnel_address
}

output "peer3_tunnel_address" {
  value = var.peer3_tunnel_address
}

output "peer4_tunnel_address" {
  value = var.peer4_tunnel_address
}
