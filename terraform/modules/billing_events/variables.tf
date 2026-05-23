variable "env" {
  description = "Deployment environment name."
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

variable "project_number" {
  description = "Google Cloud project number."
  type        = string
}

variable "region" {
  description = "Google Cloud region for regional resources."
  type        = string
}
