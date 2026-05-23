output "jwt_private_key_secret_id" {
  description = "Secret ID for the control-plane JWT private key."
  value       = google_secret_manager_secret.jwt_private_key.secret_id
}
