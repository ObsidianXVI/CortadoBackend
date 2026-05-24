output "file_changes_topic_name" {
  description = "Pub/Sub topic name that receives workspace file change events."
  value       = google_pubsub_topic.file_changes.name
}

output "indexer_updater_subscription_name" {
  description = "Pub/Sub subscription name that pushes file changes to the indexer-updater service."
  value       = google_pubsub_subscription.indexer_updater.name
}
