output "service_name" {
  description = "Name of the Cloud Run control-plane service."
  value       = google_cloud_run_v2_service.control_plane.name
}

output "service_uri" {
  description = "Public URI of the Cloud Run control-plane service."
  value       = google_cloud_run_v2_service.control_plane.uri
}
