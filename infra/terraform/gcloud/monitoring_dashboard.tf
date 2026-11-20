# Cloud Monitoring dashboard for lab.* workload metrics.
# Metric types use the OTel GCP exporter default prefix:
#   workload.googleapis.com/<otel metric name>
#
# Aligners:
#   CUMULATIVE counters / distributions → ALIGN_DELTA
#   GAUGE (UpDownCounter)             → ALIGN_MEAN

resource "google_monitoring_dashboard" "lab_metrics" {
  dashboard_json = file("${path.module}/dashboards/lab_metrics.json")

  depends_on = [google_project_service.monitoring_api]
}
