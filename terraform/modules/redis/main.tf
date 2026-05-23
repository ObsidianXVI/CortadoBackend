resource "google_redis_instance" "auth_cache" {
  authorized_network = format("projects/%s/global/networks/%s", var.project_id, var.network_name)
  connect_mode       = "PRIVATE_SERVICE_ACCESS"
  labels             = var.labels
  memory_size_gb     = var.memory_size_gb
  name               = "cortado-auth-cache-${var.env}"
  project            = var.project_id
  redis_version      = "REDIS_7_0"
  region             = var.region
  tier               = "BASIC"
}
