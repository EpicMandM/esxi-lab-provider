output "service_account_key_json" {
  description = "Service account key JSON content (Calendar, Gmail, and Cloud Monitoring)"
  value       = base64decode(google_service_account_key.calendar_sa_key.private_key)
  sensitive   = true
}

output "service_account_json_path" {
  description = "Path to the generated service-account.json file — use as GOOGLE_APPLICATION_CREDENTIALS"
  value       = local_file.service_account_json.filename
}

output "service_account_email" {
  description = "Service account email address (Calendar, Gmail, and Cloud Monitoring)"
  value       = google_service_account.calendar_sa.email
}
