terraform {
  # renovate: datasource=github-releases depName=opentofu/opentofu
  required_version = ">= 1.6.0"
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 7.21.0"
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

# ──────────────────────────────────────────────
# Service Account for Calendar + Gmail
# ──────────────────────────────────────────────

resource "google_service_account" "calendar_sa" {
  account_id   = "calendar-service-account"
  display_name = "Calendar and Gmail Service Account"
  description  = "Service account for accessing Google Calendar and Gmail APIs"
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
