output "bucket_name" {
  description = "Name of the workspace snapshots bucket."
  value       = google_storage_bucket.workspace_snapshots.name
}

output "bucket_url" {
  description = "gs:// URL for the workspace snapshots bucket."
  value       = "gs://${google_storage_bucket.workspace_snapshots.name}"
}
