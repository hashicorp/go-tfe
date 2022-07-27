output "aws_instance_id" {
  value       = module.tflocal.aws_instance_id
  description = "The AWS identifier for the setup's EC2 instance"
}

output "ngrok_domain" {
  value       = module.tflocal.ngrok_domain
  description = "The ngrok domain name reserved for this instance"
}

output "tfe_address" {
  value       = module.tflocal.tfe_address
  description = "The full URL including scheme for the ngrok domain reserved for this instance"
}

output "tfe_password" {
  value       = module.tflocal.tfe_password
  sensitive   = true
  description = "The seeded site-admin password for this instance that can be used to log into the UI"
}

output "tfe_token" {
  value       = module.tflocal.tfe_token
  sensitive   = true
  description = "The seeded site-admin token for this instance that can be used to authenticate API actions"
}
