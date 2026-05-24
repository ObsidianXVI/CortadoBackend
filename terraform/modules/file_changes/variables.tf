variable "ack_deadline_seconds" {
  description = "Pub/Sub acknowledgement deadline for the indexer-updater push subscription."
  type        = number
  default     = 60
}

variable "env" {
  description = "Deployment environment name."
  type        = string
}

variable "indexer_updater_service_uri" {
  description = "Public URI of the indexer-updater Cloud Run service."
  type        = string
}

variable "labels" {
  description = "Common labels applied to supported resources."
  type        = map(string)
}

variable "project_id" {
  description = "Google Cloud project ID."
  type        = string
}
