output "billing_dataset_id" {
  description = "BigQuery dataset ID that stores billing usage events."
  value       = google_bigquery_dataset.billing.dataset_id
}

output "usage_events_subscription_name" {
  description = "Pub/Sub subscription name that exports usage events to BigQuery."
  value       = google_pubsub_subscription.usage_to_bigquery.name
}

output "usage_events_table_id" {
  description = "BigQuery table ID that stores usage events."
  value       = google_bigquery_table.usage_events.table_id
}

output "usage_events_topic_name" {
  description = "Pub/Sub topic name that receives usage events."
  value       = google_pubsub_topic.usage_events.name
}
