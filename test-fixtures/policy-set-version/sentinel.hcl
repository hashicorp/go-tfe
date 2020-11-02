policy "enforce-mandatory-tags" {
  source = "./enforce-mandatory-tags.sentinel"
  enforcement_level = "hard-mandatory"
}

module "tfplan-functions" {
  source = "https://raw.githubusercontent.com/hashicorp/terraform-guides/master/governance/third-generation/common-functions/tfplan-functions/tfplan-functions.sentinel"
}
