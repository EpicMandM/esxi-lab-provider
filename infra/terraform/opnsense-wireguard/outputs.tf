output "wireguard_server_id" {
  description = "The ID of the WireGuard server"
  value       = opnsense_wireguard_server.dev_server.id
}

output "wireguard_server_name" {
  description = "The name of the WireGuard server"
  value       = opnsense_wireguard_server.dev_server.name
}

output "wireguard_server_port" {
  description = "The port the WireGuard server is listening on"
  value       = opnsense_wireguard_server.dev_server.port
}

output "wireguard_server_tunnel_address" {
  description = "The tunnel address of the WireGuard server"
  value       = opnsense_wireguard_server.dev_server.tunnel_address
}

output "peer1_id" {
  description = "The ID of peer 1"
  value       = opnsense_wireguard_client.peer1.id
}

output "peer1_name" {
  description = "The name of peer 1"
  value       = opnsense_wireguard_client.peer1.name
}

output "peer1_tunnel_address" {
  description = "The tunnel address for peer 1"
  value       = opnsense_wireguard_client.peer1.tunnel_address
}

output "peer2_id" {
  description = "The ID of peer 2"
  value       = opnsense_wireguard_client.peer2.id
}

output "peer2_name" {
  description = "The name of peer 2"
  value       = opnsense_wireguard_client.peer2.name
}

output "peer2_tunnel_address" {
  description = "The tunnel address for peer 2"
  value       = opnsense_wireguard_client.peer2.tunnel_address
}

output "peer3_id" {
  description = "The ID of peer 3"
  value       = opnsense_wireguard_client.peer3.id
}

output "peer3_name" {
  description = "The name of peer 3"
  value       = opnsense_wireguard_client.peer3.name
}

output "peer3_tunnel_address" {
  description = "The tunnel address for peer 3"
  value       = opnsense_wireguard_client.peer3.tunnel_address
}
