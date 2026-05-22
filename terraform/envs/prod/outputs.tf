output "artifact_registry_repository_id" {
  description = "Artifact Registry repository ID for the prod environment."
  value       = module.gke.artifact_registry_repository_id
}

output "cluster_name" {
  description = "Name of the GKE cluster for the prod environment."
  value       = module.gke.cluster_name
}

output "control_plane_service_account_email" {
  description = "Control plane service account email for the prod environment."
  value       = module.iam.control_plane_service_account_email
}

output "workspace_agent_service_account_email" {
  description = "Workspace agent service account email for the prod environment."
  value       = module.iam.workspace_agent_service_account_email
}

