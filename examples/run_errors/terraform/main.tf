# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

terraform {
  cloud {
    workspaces {
      name = "go-tfe-examples-run_errors"
    }
  }
}

# The following example should return an error
data "http" "example_head" {
  url    = "https://this-shall-not-exist.hashicorp.com/example"
  method = "GET"
}
