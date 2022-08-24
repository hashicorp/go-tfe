variable "aws_region" {
  type = string
}

variable "artifactory_username" {
  description = "Username for Artifactory repository access"
  type = string
}

variable "artifactory_token" {
  description = "Token for Artifactory repository access"
  type = string
  sensitive = true
}

variable "tflocal_cloud_admin_password" {
  description = "The password for logging into this tfe:local's TFC instance; stored in 1Password under Engineering Service's 'TFLOCAL_CLOUD tfe instance login and ngrok'"
  type        = string
}

variable "env" {
  description = "Environment variables that will be present during startup. These may be propagated into services through the Docker Compose definitions"
  type        = map(string)
  default     = {}
}

variable "git_branch" {
  description = "Git branch of the `atlas` repo to build tfe:local from"
  type        = string
}

variable "tfe_ref" {
  description = "Git ref for the TFE tags to use for the stack's Docker images. If left blank will use the most recently published ref"
  type        = string
  default     = ""
}

variable "ngrok_domain" {
  description = "Public-facing ngrok url; stored in 1Password under Engineering Service's 'TFLOCAL_CLOUD tfe instance login and ngrok'"
  type        = string
}

variable "ngrok_tunnel_token" {
  description = "The NGROK Tunnel token"
  type        = string
  sensitive   = true
}

variable "private_github_token" {
  description = "Private token to give access to `atlas` repo via GitHub API"
  type        = string
  sensitive   = true
}

variable "private_github_user" {
  description = "User to give access to `atlas` repo via remote git calls"
  type        = string
}

variable "gem_contribsys_key" {
  description = "Sidekiq license key necessary for tfe:local's bundle install"
  type        = string
  sensitive   = true
}

variable "ejson_file_name" {
  description = "Name for EJSON secrets file"
  type        = string
}

variable "ejson_file_content" {
  description = "EJSON secrets key that is the sole content of EJSON secrets file"
  type        = string
  sensitive   = true
}

variable "quay_username" {
  description = "Username for Quay.io Docker container access"
  type        = string
}

variable "quay_password" {
  description = "Password for Quay.io Docker container access"
  type        = string
  sensitive   = true
}

variable "run_cleanup_script" {
  description = "On/off toggle for running the delete script to allow for easier troubleshooting from TFE UI for this job; set to 'true' or 'false'"
  type        = bool
  default     = true
}
