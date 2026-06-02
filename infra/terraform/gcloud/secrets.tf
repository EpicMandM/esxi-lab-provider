# Upload: secrets.env → task secrets:push

resource "google_project_service" "secretmanager_api" {
  service            = "secretmanager.googleapis.com"
  disable_on_destroy = false
}

resource "google_secret_manager_secret" "esxi_lab_env" {
  project   = var.gcp_project
  secret_id = var.lab_env_secret_id

  replication {
    auto {}
  }

  depends_on = [google_project_service.secretmanager_api]
}
