<!--
Thank you for contributing to hashicorp/go-tfe!

Here's what to expect after the pull request is opened:

The test suite contains many acceptance tests that are run against a test version of Terraform Cloud, and additional testing is done against Terraform Enterprise. You can read more about running the tests against your own Terraform Enterprise environment in TESTS.md. Our CI system (CircleCI) will not test your fork unless you are an authorized employee, so a HashiCorp maintainer will initiate the tests for you and report any missing tests or simple problems. In order to speed up this process, it's not uncommon for your commits to be incorporated into another PR that we can commit test changes to.

Your change, depending on its impact, may be released in the following ways:

  1. For impactful bug fixes, it can be released fairly quickly as a patch release.
  2. For noncritical bug fixes and new features, it will be incorporated into the next minor version release.
  3. For breaking changes (those changes that alter the public method signatures), more consideration must be made and alternatives may be considered, depending on upgrade difficulty and release schedule.

Please fill out the remaining template to assist code reviewers and testers with incorporating your change. If a section does not apply, feel free to delete it.
-->

## Description

<!-- Describe why you're making this change. -->

## Testing plan

<!--
1.  _Describe how to replicate_
1.  _the conditions under which your code performs its purpose,_
1.  _including example code to run where necessary._
-->

## External links

<!--
_Include any links here that might be helpful for people reviewing your PR. If there are none, feel free to delete this section._

- [API documentation](https://www.terraform.io/docs/cloud/api/xxxx.html)
- [Related PR](https://github.com/terraform-providers/terraform-provider-tfe/pull/xxxx)

-->

## Output from tests (This may require access to a TFE instance, and may be ignored if you have no environment to test against.)

<!--
_Please run the tests locally for any files you changes and include the output here._
-->
```
$ TFE_ADDRESS="https://example" TFE_TOKEN="example" TF_ACC="1" go test ./... -v -tags=integration -run TestFunctionsAffectedByChange

...
```
