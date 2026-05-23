locals {
  apis = [
    "artifactregistry.googleapis.com",
    "bigquery.googleapis.com",
    "cloudresourcemanager.googleapis.com",
    "container.googleapis.com",
    "iam.googleapis.com",
    "pubsub.googleapis.com",
    "secretmanager.googleapis.com",
  ]

  common_labels = {
    env     = var.env
    project = "cortado"
  }
}

resource "google_project_service" "api" {
  for_each = toset(local.apis)

  service            = each.value
  disable_on_destroy = false
}

module "iam" {
  source = "../../modules/iam"

  env        = var.env
  project_id = var.project_id

  depends_on = [google_project_service.api]
}

module "gke" {
  source = "../../modules/gke"

  control_plane_sa_email  = module.iam.control_plane_service_account_email
  env                     = var.env
  labels                  = local.common_labels
  project_id              = var.project_id
  region                  = var.region
  workspace_agent_sa_name = module.iam.workspace_agent_service_account_name

  depends_on = [google_project_service.api]
}
