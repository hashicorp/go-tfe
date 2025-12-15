# Copyright IBM Corp. 2018, 2025
# SPDX-License-Identifier: MPL-2.0

policy "enforce-mandatory-tags" {
  source = "./enforce-mandatory-tags.sentinel"
  enforcement_level = "hard-mandatory"
}
