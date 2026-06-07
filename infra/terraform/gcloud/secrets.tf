# Upload: secrets.env → task secrets:push

resource "google_project_service" "secretmanager_api" {
  service            = "secretmanager.googleapis.com"
  disable_on_destroy = false
}

resource "google_secret_manager_secret" "esxi_lab_env" {
  project   = local.gcp_project
  secret_id = "esxi-lab-env"

  replication {
    auto {}
  }

  depends_on = [google_project_service.secretmanager_api]
}
