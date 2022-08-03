terraform {
  required_version = "~> 1.2"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 3.46"
    }

    cloudinit = {
      source  = "hashicorp/cloudinit"
      version = "~> 2.2"
    }

    random = {
      source  = "hashicorp/random"
      version = "~> 3.1"
    }

    ngrok = {
      source  = "ngrok/ngrok"
      version = "~> 0.1"
    }
  }

  cloud {}
}
