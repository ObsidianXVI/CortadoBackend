output "address" {
  description = "host:port address for the auth cache."
  value       = format("%s:%d", google_redis_instance.auth_cache.host, google_redis_instance.auth_cache.port)
}
