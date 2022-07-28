
variable "TFC_WORKSPACE_NAME" {
  type        = string
  description = "Current TFE(C) workspace name. Used as part of the AWS provider's assume role session name argument. Set to a placeholder default to allow for non-TFE runs to occur when needed."
  default     = "TFC_WORKSPACE_NAME_DEFAULT"
}

variable "TFC_RUN_ID" {
  type        = string
  description = "Current TFE(C) run ID. Used as part of the AWS provider's assume role session name argument. Set to a placeholder default to allow for non-TFE runs to occur when needed."
  default     = "TFC_RUN_ID_DEFAULT"
}

variable "aws_assume_role_arn" {
  type        = string
  description = "AWS arn of the role the provider should assume role into."
}

variable "aws_assume_role_external_id" {
  type        = string
  description = "External ID of the role the provider should assume role into, prevents collisions."
}

provider "aws" {
  region = var.aws_region

  assume_role {
    # [Session name] is a string of characters consisting of upper- and lower-case alphanumeric characters
    # with no spaces. You can also include underscores or any of the following characters: =,.@-
    session_name = "${replace(var.TFC_WORKSPACE_NAME, "/", "-")}-${var.TFC_RUN_ID}"
    role_arn     = var.aws_assume_role_arn
    external_id  = var.aws_assume_role_external_id
  }
}

provider "random" {}

provider "cloudinit" {}

provider "ngrok" {}
