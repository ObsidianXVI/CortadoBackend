locals {
  image_url = format(
    "%s-docker.pkg.dev/%s/%s/cortado-control-plane:%s",
    var.region,
    var.project_id,
    var.repository_id,
    var.image_tag,
  )
}

resource "google_cloud_run_v2_service" "control_plane" {
  name                = "cortado-control-plane-${var.env}"
  location            = var.region
  deletion_protection = false
  ingress             = "INGRESS_TRAFFIC_ALL"
  labels              = var.labels

  template {
    labels          = var.labels
    service_account = var.service_account_email

    containers {
      image = local.image_url

      ports {
        container_port = 8080
      }

      env {
        name  = "CORTADO_ENV"
        value = var.env == "dev" ? "development" : "production"
      }

      env {
        name  = "GCP_PROJECT"
        value = var.project_id
      }
    }
  }
}

resource "google_cloud_run_v2_service_iam_member" "public" {
  location = google_cloud_run_v2_service.control_plane.location
  name     = google_cloud_run_v2_service.control_plane.name
  project  = var.project_id
  role     = "roles/run.invoker"
  member   = "allUsers"
}
