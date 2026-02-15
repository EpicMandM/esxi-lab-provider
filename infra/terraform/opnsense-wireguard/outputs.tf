output "peer_ids" {
  description = "UUIDs of the provisioned WireGuard peer slots"
  value = {
    "lab-user-1" = opnsense_wireguard_client.peer1.id
    "lab-user-2" = opnsense_wireguard_client.peer2.id
    "lab-user-3" = opnsense_wireguard_client.peer3.id
    "lab-user-4" = opnsense_wireguard_client.peer4.id
  }
}

output "peer_tunnel_addresses" {
  description = "Tunnel addresses assigned to each peer"
  value = [
    var.peer1_tunnel_address,
    var.peer2_tunnel_address,
    var.peer3_tunnel_address,
    var.peer4_tunnel_address,
  ]
}

output "server_id" {
  description = "The ID of the WireGuard server"
  value       = opnsense_wireguard_server.wg.id
}

output "server_public_key" {
  description = "WireGuard server public key"
  value       = var.wireguard_server_public_key
}

output "server_endpoint" {
  description = "WireGuard server endpoint (host:port) for client configs"
  value       = var.wireguard_public_endpoint
}

output "server_tunnel_address" {
  description = "WireGuard server tunnel address CIDR"
  value       = var.wireguard_server_tunnel_address
}

output "opnsense_url" {
  description = "OPNsense URL"
  value       = var.opnsense_url
}
