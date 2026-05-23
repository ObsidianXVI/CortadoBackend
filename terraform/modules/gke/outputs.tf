output "cluster_id" {
  description = "ID of the GKE cluster."
  value       = google_container_cluster.main.id
}

output "cluster_name" {
  description = "Name of the GKE cluster."
  value       = google_container_cluster.main.name
}
