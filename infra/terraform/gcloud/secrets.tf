# Upload: secrets.env / wg0.conf → task secrets:push

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

resource "google_secret_manager_secret" "esxi_lab_wg0" {
  project   = local.gcp_project
  secret_id = "esxi-lab-wg0"

  replication {
    auto {}
  }

  depends_on = [google_project_service.secretmanager_api]
}
