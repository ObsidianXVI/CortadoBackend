locals {
  control_plane_image_url = format(
    "%s-docker.pkg.dev/%s/%s/cortado-control-plane:%s",
    var.region,
    var.project_id,
    var.repository_id,
    var.image_tag,
  )
  indexer_updater_image_url = format(
    "%s-docker.pkg.dev/%s/%s/cortado-indexer:%s",
    var.region,
    var.project_id,
    var.repository_id,
    var.indexer_updater_image_tag,
  )
  indexer_updater_qdrant_url_template = format(
    "http://{workspace_id}.%s.svc.%s:6333",
    var.workspace_namespace,
    var.cluster_dns_domain,
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
      image = local.control_plane_image_url

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

      env {
        name  = "CORTADO_CLUSTER_DNS_DOMAIN"
        value = var.cluster_dns_domain
      }

      env {
        name  = "CORTADO_AUTH_API_KEYS_COLLECTION"
        value = "api_keys"
      }

      env {
        name = "CORTADO_AI_API_KEY"
        value_source {
          secret_key_ref {
            secret  = var.ai_api_key_secret_id
            version = "latest"
          }
        }
      }

      env {
        name  = "CORTADO_AI_MODEL"
        value = "gemini-2.5-flash"
      }

      env {
        name  = "CORTADO_AUTH_CACHE_ADDR"
        value = var.auth_cache_addr
      }

      env {
        name  = "CORTADO_AUTH_REFRESH_TOKENS_COLLECTION"
        value = "refresh_tokens"
      }

      env {
        name  = "CORTADO_GKE_CLUSTER_LOCATION"
        value = var.region
      }

      env {
        name  = "CORTADO_GKE_CLUSTER_NAME"
        value = var.cluster_name
      }

      env {
        name  = "CORTADO_SNAPSHOT_BUCKET"
        value = var.snapshot_bucket_name
      }

      env {
        name = "CORTADO_SNAPSHOT_PASSWORD"
        value_source {
          secret_key_ref {
            secret  = var.snapshot_password_secret_id
            version = "latest"
          }
        }
      }

      env {
        name  = "CORTADO_WORKSPACE_STORAGE_CLASS"
        value = "cortado-workspace"
      }

      env {
        name  = "CORTADO_WORKSPACE_NAMESPACE"
        value = var.workspace_namespace
      }

      env {
        name  = "CORTADO_VERTEX_PROJECT_ID"
        value = var.project_id
      }

      env {
        name  = "CORTADO_USAGE_EVENTS_TOPIC"
        value = var.usage_events_topic_name
      }

      env {
        name = "CORTADO_JWT_PRIVATE_KEY_PEM"
        value_source {
          secret_key_ref {
            secret  = var.jwt_private_key_secret_id
            version = "latest"
          }
        }
      }
    }

    vpc_access {
      egress = "PRIVATE_RANGES_ONLY"

      network_interfaces {
        network    = var.network_name
        subnetwork = var.subnetwork_name
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

resource "google_cloud_run_v2_service" "indexer_updater" {
  name                = "cortado-indexer-updater-${var.env}"
  location            = var.region
  deletion_protection = false
  ingress             = "INGRESS_TRAFFIC_ALL"
  labels              = var.labels

  template {
    labels          = var.labels
    service_account = var.indexer_updater_service_account_email

    containers {
      image   = local.indexer_updater_image_url
      command = ["python"]
      args    = ["-m", "cortado_indexer.updater_server"]

      ports {
        container_port = 8080
      }

      env {
        name  = "CORTADO_CLUSTER_DNS_DOMAIN"
        value = var.cluster_dns_domain
      }

      env {
        name  = "CORTADO_ENV"
        value = var.env == "dev" ? "development" : "production"
      }

      env {
        name  = "CORTADO_QDRANT_URL_TEMPLATE"
        value = local.indexer_updater_qdrant_url_template
      }

      env {
        name  = "CORTADO_UPDATER_BATCH_WINDOW_SECONDS"
        value = "5"
      }

      env {
        name  = "CORTADO_VERTEX_PROJECT_ID"
        value = var.project_id
      }

      env {
        name  = "CORTADO_WORKSPACE_NAMESPACE"
        value = var.workspace_namespace
      }

      env {
        name  = "GCP_PROJECT"
        value = var.project_id
      }
    }

    vpc_access {
      egress = "PRIVATE_RANGES_ONLY"

      network_interfaces {
        network    = var.network_name
        subnetwork = var.subnetwork_name
      }
    }
  }
}

resource "google_cloud_run_v2_service_iam_member" "indexer_updater_public" {
  location = google_cloud_run_v2_service.indexer_updater.location
  name     = google_cloud_run_v2_service.indexer_updater.name
  project  = var.project_id
  role     = "roles/run.invoker"
  member   = "allUsers"
}
