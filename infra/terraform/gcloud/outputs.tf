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

output "gcp_project" {
  description = "GCP project ID"
  value       = local.gcp_project
}

output "lab_env_secret_id" {
  value = google_secret_manager_secret.esxi_lab_env.secret_id
}

output "wg0_secret_id" {
  value = google_secret_manager_secret.esxi_lab_wg0.secret_id
}

output "cost_guard_function_uri" {
  description = "HTTP URI of the scheduled free-tier cost guard function"
  value       = google_cloudfunctions2_function.cost_guard.service_config[0].uri
}

output "cost_guard_scheduler_job" {
  description = "Cloud Scheduler job name for the cost guard"
  value       = google_cloud_scheduler_job.cost_guard.name
}
