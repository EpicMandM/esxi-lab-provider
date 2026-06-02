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

output "lab_env_secret_id" {
  description = "Secret Manager secret ID for lab credentials — consumed by esxi-users and opnsense-wireguard"
  value       = google_secret_manager_secret.esxi_lab_env.secret_id
}

output "gcp_project" {
  description = "GCP project ID"
  value       = var.gcp_project
}
