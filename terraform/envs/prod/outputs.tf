output "artifact_registry_repository_id" {
  description = "Artifact Registry repository ID for the prod environment."
  value       = module.gke.artifact_registry_repository_id
}

output "billing_dataset_id" {
  description = "BigQuery dataset ID for billing usage events in the prod environment."
  value       = module.billing_events.billing_dataset_id
}

output "daemon_install_script_url" {
  description = "Public URL for the hosted daemon install script."
  value       = "https://storage.googleapis.com/${google_storage_bucket.daemon_install.name}/${google_storage_bucket_object.daemon_install_script.name}"
}

output "cluster_name" {
  description = "Name of the GKE cluster for the prod environment."
  value       = module.gke.cluster_name
}

output "control_plane_service_uri" {
  description = "Public URI of the control-plane Cloud Run service for the prod environment."
  value       = module.cloudrun.service_uri
}

output "control_plane_service_account_email" {
  description = "Control plane service account email for the prod environment."
  value       = module.iam.control_plane_service_account_email
}

output "file_changes_subscription_name" {
  description = "Pub/Sub subscription name for workspace file changes in the prod environment."
  value       = module.file_changes.indexer_updater_subscription_name
}

output "file_changes_topic_name" {
  description = "Pub/Sub topic name for workspace file changes in the prod environment."
  value       = module.file_changes.file_changes_topic_name
}

output "indexer_updater_service_account_email" {
  description = "Indexer-updater service account email for the prod environment."
  value       = module.iam.indexer_updater_service_account_email
}

output "indexer_updater_service_uri" {
  description = "Public URI of the indexer-updater Cloud Run service for the prod environment."
  value       = module.cloudrun.indexer_updater_service_uri
}

output "workspace_agent_service_account_email" {
  description = "Workspace agent service account email for the prod environment."
  value       = module.iam.workspace_agent_service_account_email
}

output "workspace_snapshots_bucket_name" {
  description = "Workspace snapshots bucket name for the prod environment."
  value       = module.workspace_snapshots.bucket_name
}

output "workspace_snapshots_bucket_url" {
  description = "Workspace snapshots bucket URL for the prod environment."
  value       = module.workspace_snapshots.bucket_url
}

output "usage_events_subscription_name" {
  description = "Pub/Sub subscription name for usage-event export in the prod environment."
  value       = module.billing_events.usage_events_subscription_name
}

output "usage_events_table_id" {
  description = "BigQuery table ID for usage events in the prod environment."
  value       = module.billing_events.usage_events_table_id
}

output "usage_events_topic_name" {
  description = "Pub/Sub topic name for usage events in the prod environment."
  value       = module.billing_events.usage_events_topic_name
}
