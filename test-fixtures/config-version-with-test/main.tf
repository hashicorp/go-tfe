# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

variable "wait_time" {
  type = string
  default = "0s"
}

resource "null_resource" "foo" {}

resource "time_sleep" "wait_5_seconds" {
  depends_on = [null_resource.foo]

  create_duration = var.wait_time
}

resource "null_resource" "bar" {
  depends_on = [time_sleep.wait_5_seconds]
}
