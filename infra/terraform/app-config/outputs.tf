output "user_config_path" {
  description = "Path to the generated user_config.toml"
  value       = local_file.user_config.filename
}
