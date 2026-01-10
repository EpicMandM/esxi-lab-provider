terraform {
  required_version = ">= 1.6.0"
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 7.12.0"
    }
  }
}

provider "google" {
  project = "exsi-chat-app-478319"
  region  = "europe-west1"
}

# Enable the Google Calendar API
resource "google_project_service" "calendar_api" {
  service            = "calendar-json.googleapis.com"
  disable_on_destroy = false
}

# Enable the Gmail API
resource "google_project_service" "gmail_api" {
  service            = "gmail.googleapis.com"
  disable_on_destroy = false
}

# Create a Service Account for the application (Calendar + Gmail)
resource "google_service_account" "calendar_sa" {
  account_id   = "calendar-service-account"
  display_name = "Calendar and Gmail Service Account"
  description  = "Service account for accessing Google Calendar and Gmail APIs"
}

# Create a key for the Service Account
resource "google_service_account_key" "calendar_sa_key" {
  service_account_id = google_service_account.calendar_sa.name
}

# Output the JSON key (sensitive)
output "service_account_key" {
  value     = base64decode(google_service_account_key.calendar_sa_key.private_key)
  sensitive = true
}
