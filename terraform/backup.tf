resource "google_gke_backup_backup_plan" "todo_app_backup" {
  name     = "todo-app-backup"
  cluster  = google_container_cluster.primary.id
  location = var.region

  retention_policy {
    backup_retain_days = 7
  }

  backup_schedule {
    cron_schedule = "0 2 * * *" # 2 AM UTC
  }

  backup_config {
    include_secrets     = true
    include_volume_data = true
    selected_namespaces {
      namespaces = ["todo-app"]
    }
  }

  depends_on = [google_project_service.gkebackup_api]
}
