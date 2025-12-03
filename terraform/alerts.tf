resource "google_monitoring_alert_policy" "high_error_rate" {
  display_name = "High Error Rate (HTTP 5xx)"
  combiner     = "OR"
  conditions {
    display_name = "HTTP 5xx > 1%"
    condition_threshold {
      filter     = "resource.type=\"k8s_container\" AND metric.type=\"prometheus.googleapis.com/http_requests_total/counter\" AND metric.label.code=~\"5..\""
      duration   = "300s"
      comparison = "COMPARISON_GT"
      aggregations {
        alignment_period   = "60s"
        per_series_aligner = "ALIGN_RATE"
        cross_series_reducer = "REDUCE_SUM"
      }
      threshold_value = 0.01
    }
  }
}

resource "google_monitoring_alert_policy" "high_latency" {
  display_name = "High Latency (> 2s)"
  combiner     = "OR"
  conditions {
    display_name = "99th Percentile Latency > 2s"
    condition_threshold {
      filter     = "resource.type=\"k8s_container\" AND metric.type=\"prometheus.googleapis.com/http_request_duration_seconds/histogram\""
      duration   = "300s"
      comparison = "COMPARISON_GT"
      aggregations {
        alignment_period   = "60s"
        per_series_aligner = "ALIGN_PERCENTILE_99"
      }
      threshold_value = 2
    }
  }
}

resource "google_monitoring_alert_policy" "container_restarts" {
  display_name = "Container Restarts"
  combiner     = "OR"
  conditions {
    display_name = "Container Restarting"
    condition_threshold {
      filter     = "resource.type=\"k8s_container\" AND metric.type=\"kubernetes.io/container/restart_count\""
      duration   = "300s"
      comparison = "COMPARISON_GT"
      aggregations {
        alignment_period   = "60s"
        per_series_aligner = "ALIGN_RATE"
      }
      threshold_value = 0
    }
  }
}
