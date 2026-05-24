locals {
  apis = [
    "artifactregistry.googleapis.com",
    "bigquery.googleapis.com",
    "cloudbuild.googleapis.com",
    "cloudresourcemanager.googleapis.com",
    "compute.googleapis.com",
    "container.googleapis.com",
    "dns.googleapis.com",
    "firestore.googleapis.com",
    "iam.googleapis.com",
    "pubsub.googleapis.com",
    "redis.googleapis.com",
    "run.googleapis.com",
    "secretmanager.googleapis.com",
    "storage.googleapis.com",
  ]

  common_labels = {
    env     = var.env
    project = "cortado"
  }

  workspace_gsa_email = module.iam.workspace_agent_service_account_email
  workspace_image = format(
    "%s-docker.pkg.dev/%s/%s/%s:%s",
    var.region,
    var.project_id,
    module.gke.artifact_registry_repository_id,
    var.workspace_image_name,
    var.workspace_image_tag,
  )
  workspace_bootstrap_manifest = templatefile("${path.module}/../../k8s/workspace-namespace.yaml", {
    workspace_gsa = local.workspace_gsa_email
  })
  workspace_test_pod_manifest = templatefile("${path.module}/../../k8s/workspace-pod-test.yaml", {
    workspace_image = local.workspace_image
  })
}

data "google_project" "current" {
  project_id = var.project_id
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

  # Grant Firestore data access to the control-plane service account.
  # Keep existing role(s) and add roles/datastore.user per feat-1-3 Task 1.3.2.
  control_plane_project_roles = ["roles/container.developer", "roles/datastore.user"]

  depends_on = [google_project_service.api]
}

module "gke" {
  source = "../../modules/gke"

  control_plane_sa_email  = module.iam.control_plane_service_account_email
  cluster_dns_domain      = var.cluster_dns_domain
  env                     = var.env
  labels                  = local.common_labels
  network_name            = var.network_name
  project_id              = var.project_id
  region                  = var.region
  subnetwork_name         = var.subnetwork_name
  workspace_agent_sa_name = module.iam.workspace_agent_service_account_name

  depends_on = [google_project_service.api]
}

module "secrets" {
  source = "../../modules/secrets"

  control_plane_service_account_member = module.iam.control_plane_service_account_member
  env                                  = var.env
  labels                               = local.common_labels
  project_id                           = var.project_id

  depends_on = [google_project_service.api]
}

module "redis" {
  source = "../../modules/redis"

  env          = var.env
  labels       = local.common_labels
  network_name = var.network_name
  project_id   = var.project_id
  region       = var.region

  depends_on = [google_project_service.api]
}

module "cloudrun" {
  source = "../../modules/cloudrun"

  ai_api_key_secret_id                  = module.secrets.ai_api_key_secret_id
  auth_cache_addr                       = module.redis.address
  cluster_dns_domain                    = var.cluster_dns_domain
  cluster_name                          = module.gke.cluster_name
  env                                   = var.env
  image_tag                             = var.control_plane_image_tag
  indexer_updater_image_tag             = var.indexer_updater_image_tag
  indexer_updater_service_account_email = module.iam.indexer_updater_service_account_email
  jwt_private_key_secret_id             = module.secrets.jwt_private_key_secret_id
  labels                                = local.common_labels
  network_name                          = var.network_name
  project_id                            = var.project_id
  region                                = var.region
  repository_id                         = module.gke.artifact_registry_repository_id
  service_account_email                 = module.iam.control_plane_service_account_email
  snapshot_bucket_name                  = module.workspace_snapshots.bucket_name
  snapshot_password_secret_id           = module.secrets.snapshot_password_secret_id
  subnetwork_name                       = var.subnetwork_name
  usage_events_topic_name               = module.billing_events.usage_events_topic_name
  workspace_namespace                   = var.workspace_namespace

  depends_on = [google_project_service.api]
}

module "file_changes" {
  source = "../../modules/file_changes"

  env                         = var.env
  indexer_updater_service_uri = module.cloudrun.indexer_updater_service_uri
  labels                      = local.common_labels
  project_id                  = var.project_id

  depends_on = [google_project_service.api, module.cloudrun]
}

module "billing_events" {
  source = "../../modules/billing_events"

  env                                   = var.env
  labels                                = local.common_labels
  project_id                            = var.project_id
  project_number                        = data.google_project.current.number
  region                                = var.region
  workspace_agent_service_account_email = module.iam.workspace_agent_service_account_email

  depends_on = [google_project_service.api]
}

module "workspace_snapshots" {
  source = "../../modules/workspace_snapshots"

  env                                   = var.env
  labels                                = local.common_labels
  project_id                            = var.project_id
  region                                = var.region
  workspace_agent_service_account_email = module.iam.workspace_agent_service_account_email

  depends_on = [google_project_service.api]
}

resource "null_resource" "k8s_bootstrap" {
  depends_on = [module.gke]

  triggers = {
    cluster_name          = module.gke.cluster_name
    manifest_hash         = filesha256("${path.module}/../../k8s/workspace-namespace.yaml")
    workspace_agent_email = local.workspace_gsa_email
  }

  provisioner "local-exec" {
    interpreter = ["/bin/bash", "-c"]
    command     = <<-EOT
      set -euo pipefail
      gcloud container clusters get-credentials ${module.gke.cluster_name} \
        --region ${var.region} \
        --project ${var.project_id}
      cat <<'EOF' >/tmp/cortado-workspace-bootstrap-${var.env}.yaml
${local.workspace_bootstrap_manifest}
EOF
      kubectl apply -f /tmp/cortado-workspace-bootstrap-${var.env}.yaml
    EOT
  }
}

resource "null_resource" "k8s_workspace_test_pod" {
  count = var.workspace_test_pod_enabled ? 1 : 0

  depends_on = [null_resource.k8s_bootstrap]

  triggers = {
    cluster_name  = module.gke.cluster_name
    enabled       = tostring(var.workspace_test_pod_enabled)
    manifest_hash = filesha256("${path.module}/../../k8s/workspace-pod-test.yaml")
    workspace_img = local.workspace_image
  }

  provisioner "local-exec" {
    interpreter = ["/bin/bash", "-c"]
    command     = <<-EOT
      set -euo pipefail
      gcloud container clusters get-credentials ${module.gke.cluster_name} \
        --region ${var.region} \
        --project ${var.project_id}
      cat <<'EOF' >/tmp/cortado-workspace-pod-test-${var.env}.yaml
${local.workspace_test_pod_manifest}
EOF
      kubectl -n cortado-workspaces delete pod/workspace-pod-test --ignore-not-found
      kubectl apply -f /tmp/cortado-workspace-pod-test-${var.env}.yaml
    EOT
  }
}

resource "null_resource" "k8s_storageclass" {
  depends_on = [module.gke]

  triggers = {
    cluster_name  = module.gke.cluster_name
    manifest_hash = filesha256("${path.module}/../../k8s/workspace-storageclass.yaml")
  }

  provisioner "local-exec" {
    interpreter = ["/bin/bash", "-c"]
    command     = <<-EOT
      set -euo pipefail
      gcloud container clusters get-credentials ${module.gke.cluster_name} \
        --region ${var.region} \
        --project ${var.project_id}
      cat <<'EOF' >/tmp/cortado-workspace-storageclass-${var.env}.yaml
${file("${path.module}/../../k8s/workspace-storageclass.yaml")}
EOF
      kubectl apply -f /tmp/cortado-workspace-storageclass-${var.env}.yaml
    EOT
  }
}
