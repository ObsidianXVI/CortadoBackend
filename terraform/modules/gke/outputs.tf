output "artifact_registry_repository_id" {
  description = "Artifact Registry repository ID for Cortado images."
  value       = google_artifact_registry_repository.main.repository_id
}

output "cluster_id" {
  description = "ID of the GKE cluster."
  value       = google_container_cluster.main.id
}

output "cluster_name" {
  description = "Name of the GKE cluster."
  value       = google_container_cluster.main.name
}

