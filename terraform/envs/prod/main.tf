locals {
  apis = [
    "artifactregistry.googleapis.com",
    "bigquery.googleapis.com",
    "cloudbuild.googleapis.com",
    "cloudresourcemanager.googleapis.com",
    "container.googleapis.com",
    "firestore.googleapis.com",
    "iam.googleapis.com",
    "pubsub.googleapis.com",
    "run.googleapis.com",
    "secretmanager.googleapis.com",
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
  env                     = var.env
  labels                  = local.common_labels
  project_id              = var.project_id
  region                  = var.region
  workspace_agent_sa_name = module.iam.workspace_agent_service_account_name

  depends_on = [google_project_service.api]
}

module "cloudrun" {
  source = "../../modules/cloudrun"

  env                   = var.env
  image_tag             = var.control_plane_image_tag
  labels                = local.common_labels
  project_id            = var.project_id
  region                = var.region
  repository_id         = module.gke.artifact_registry_repository_id
  service_account_email = module.iam.control_plane_service_account_email

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