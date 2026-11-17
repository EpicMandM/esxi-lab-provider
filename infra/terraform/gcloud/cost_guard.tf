# ──────────────────────────────────────────────
# Cost guard — Cloud Function + daily Cloud Scheduler cron
# Prunes Secret Manager versions and disables unused APIs.
# ──────────────────────────────────────────────

resource "google_project_service" "cloudfunctions_api" {
  service            = "cloudfunctions.googleapis.com"
  disable_on_destroy = false
}

resource "google_project_service" "cloudscheduler_api" {
  service            = "cloudscheduler.googleapis.com"
  disable_on_destroy = false
}

resource "google_project_service" "run_api" {
  service            = "run.googleapis.com"
  disable_on_destroy = false
}

resource "google_project_service" "cloudbuild_api" {
  service            = "cloudbuild.googleapis.com"
  disable_on_destroy = false
}

resource "google_project_service" "artifactregistry_api" {
  service            = "artifactregistry.googleapis.com"
  disable_on_destroy = false
}

data "google_project" "current" {
  project_id = local.gcp_project
}

resource "google_service_account" "cost_guard" {
  account_id   = "esxi-lab-cost-guard"
  display_name = "ESXi Lab Cost Guard"
  description  = "Prunes Secret Manager versions and disables unused APIs on a schedule"
}

# Cloud Build (used by Functions Gen2 deploy) must be able to act as the runtime SA.
resource "google_service_account_iam_member" "cost_guard_cloudbuild_user" {
  service_account_id = google_service_account.cost_guard.name
  role               = "roles/iam.serviceAccountUser"
  member             = "serviceAccount:${data.google_project.current.number}@cloudbuild.gserviceaccount.com"

  depends_on = [google_project_service.cloudbuild_api]
}

resource "google_project_iam_member" "cost_guard_secret_versions" {
  project = local.gcp_project
  role    = "roles/secretmanager.secretVersionManager"
  member  = "serviceAccount:${google_service_account.cost_guard.email}"

  depends_on = [google_project_service.secretmanager_api]
}

resource "google_project_iam_member" "cost_guard_service_usage" {
  project = local.gcp_project
  role    = "roles/serviceusage.serviceUsageAdmin"
  member  = "serviceAccount:${google_service_account.cost_guard.email}"

  depends_on = [google_project_service.cloudfunctions_api]
}

resource "google_storage_bucket" "cost_guard_source" {
  name                        = "${local.gcp_project}-cost-guard-source"
  location                    = var.gcp_region
  project                     = local.gcp_project
  uniform_bucket_level_access = true
  force_destroy               = true

  lifecycle_rule {
    condition {
      age = 30
    }
    action {
      type = "Delete"
    }
  }

  depends_on = [google_project_service.storage_api]
}

data "archive_file" "cost_guard" {
  type        = "zip"
  source_dir  = "${path.module}/cost_guard"
  output_path = "${path.module}/.cost_guard.zip"
}

resource "google_storage_bucket_object" "cost_guard_source" {
  name   = "cost-guard-${data.archive_file.cost_guard.output_md5}.zip"
  bucket = google_storage_bucket.cost_guard_source.name
  source = data.archive_file.cost_guard.output_path
}

resource "google_cloudfunctions2_function" "cost_guard" {
  name        = "esxi-lab-cost-guard"
  location    = var.gcp_region
  description = "Daily free-tier guard: prune secrets, disable unused APIs"

  build_config {
    runtime     = "python312"
    entry_point = "cost_guard"
    source {
      storage_source {
        bucket = google_storage_bucket.cost_guard_source.name
        object = google_storage_bucket_object.cost_guard_source.name
      }
    }
  }

  service_config {
    max_instance_count    = 1
    min_instance_count    = 0
    available_memory      = "256Mi"
    timeout_seconds       = 120
    service_account_email = google_service_account.cost_guard.email
    # Scheduler reaches the function over the public Run URL; IAM still requires invoker.
    ingress_settings               = "ALLOW_ALL"
    all_traffic_on_latest_revision = true

    environment_variables = {
      GCP_PROJECT          = local.gcp_project
      KEEP_SECRET_VERSIONS = tostring(var.cost_guard_keep_secret_versions)
    }
  }

  depends_on = [
    google_project_service.cloudfunctions_api,
    google_project_service.run_api,
    google_project_service.cloudbuild_api,
    google_project_service.artifactregistry_api,
    google_project_iam_member.cost_guard_secret_versions,
    google_project_iam_member.cost_guard_service_usage,
  ]
}

resource "google_cloud_run_service_iam_member" "cost_guard_invoker" {
  project  = local.gcp_project
  location = google_cloudfunctions2_function.cost_guard.location
  service  = google_cloudfunctions2_function.cost_guard.name
  role     = "roles/run.invoker"
  member   = "serviceAccount:${google_service_account.cost_guard.email}"
}

# Cloud Scheduler must mint OIDC tokens for the cost-guard SA.
resource "google_project_iam_member" "scheduler_token_creator" {
  project = local.gcp_project
  role    = "roles/iam.serviceAccountTokenCreator"
  member  = "serviceAccount:service-${data.google_project.current.number}@gcp-sa-cloudscheduler.iam.gserviceaccount.com"

  depends_on = [google_project_service.cloudscheduler_api]
}

resource "google_service_account_iam_member" "scheduler_sa_user" {
  service_account_id = google_service_account.cost_guard.name
  role               = "roles/iam.serviceAccountUser"
  member             = "serviceAccount:service-${data.google_project.current.number}@gcp-sa-cloudscheduler.iam.gserviceaccount.com"

  depends_on = [google_project_service.cloudscheduler_api]
}

resource "google_project_service" "appengine_api" {
  service            = "appengine.googleapis.com"
  disable_on_destroy = false
}

# Cloud Scheduler still expects an App Engine app in the project (location is permanent).
resource "google_app_engine_application" "app" {
  project     = local.gcp_project
  location_id = "europe-west"

  depends_on = [
    google_project_service.cloudscheduler_api,
    google_project_service.appengine_api,
  ]
}

locals {
  cost_guard_uri = trimsuffix(google_cloudfunctions2_function.cost_guard.service_config[0].uri, "/")
}

resource "google_cloud_scheduler_job" "cost_guard" {
  name             = "esxi-lab-cost-guard"
  description      = "Run free-tier cost guard daily"
  schedule         = var.cost_guard_schedule
  time_zone        = "UTC"
  region           = var.gcp_region
  attempt_deadline = "320s"

  retry_config {
    retry_count = 2
  }

  http_target {
    http_method = "POST"
    uri         = local.cost_guard_uri

    oidc_token {
      service_account_email = google_service_account.cost_guard.email
      audience              = local.cost_guard_uri
    }
  }

  depends_on = [
    google_project_service.cloudscheduler_api,
    google_cloud_run_service_iam_member.cost_guard_invoker,
    google_project_iam_member.scheduler_token_creator,
    google_service_account_iam_member.scheduler_sa_user,
    google_app_engine_application.app,
  ]
}
