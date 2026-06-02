output "user_config_path" {
  value = local_file.user_config.filename
}

output "env_path" {
  value = local_file.env.filename
}
