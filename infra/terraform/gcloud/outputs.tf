output "service_account_key_json" {
  description = "Service account key JSON content"
  value       = base64decode(google_service_account_key.calendar_sa_key.private_key)
  sensitive   = true
}

output "service_account_json_path" {
  description = "Path to the generated service-account.json file"
  value       = local_file.service_account_json.filename
}

output "service_account_email" {
  description = "Service account email address"
  value       = google_service_account.calendar_sa.email
}
