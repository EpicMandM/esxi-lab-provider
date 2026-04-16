terraform {
  # renovate: datasource=github-releases depName=opentofu/opentofu
  required_version = ">= 1.6.0"
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 7.28.0"
    }
    local = {
      source  = "hashicorp/local"
      version = "~> 2.5"
    }
  }
}

provider "google" {
  project = var.gcp_project
  region  = var.gcp_region
}

# ──────────────────────────────────────────────
# Enable required Google APIs
# ──────────────────────────────────────────────

resource "google_project_service" "calendar_api" {
  service            = "calendar-json.googleapis.com"
  disable_on_destroy = false
}

resource "google_project_service" "gmail_api" {
  service            = "gmail.googleapis.com"
  disable_on_destroy = false
}

resource "google_project_service" "monitoring_api" {
  service            = "monitoring.googleapis.com"
  disable_on_destroy = false
}

resource "google_project_service" "forms_api" {
  service            = "forms.googleapis.com"
  disable_on_destroy = false
}

resource "google_project_service" "sheets_api" {
  service            = "sheets.googleapis.com"
  disable_on_destroy = false
}

resource "google_project_service" "apps_script_api" {
  service            = "script.googleapis.com"
  disable_on_destroy = false
}

# ──────────────────────────────────────────────
# Service Account for Calendar, Gmail + Metrics
# ──────────────────────────────────────────────

resource "google_service_account" "calendar_sa" {
  account_id   = "calendar-service-account"
  display_name = "ESXi Lab Service Account"
  description  = "Service account for Google Calendar, Gmail, and Cloud Monitoring (metrics writer)"
}

# Grant metricWriter so the binary can push custom metrics directly
resource "google_project_iam_member" "calendar_sa_metric_writer" {
  project = var.gcp_project
  role    = "roles/monitoring.metricWriter"
  member  = "serviceAccount:${google_service_account.calendar_sa.email}"

  depends_on = [google_project_service.monitoring_api]
}

resource "google_service_account_key" "calendar_sa_key" {
  service_account_id = google_service_account.calendar_sa.name
}

# Write the service account key to a JSON file
resource "local_file" "service_account_json" {
  filename        = "${path.module}/../service-account.json"
  file_permission = "0600"
  content         = base64decode(google_service_account_key.calendar_sa_key.private_key)
}
