<!--
Thank you for contributing to hashicorp/go-tfe! Please read docs/CONTRIBUTING.md for detailed information when preparing your change.

Here's what to expect after the pull request is opened:

The test suite contains many acceptance tests that are run against a test version of Terraform Cloud, and additional testing is done against Terraform Enterprise. You can read more about running the tests against your own Terraform Enterprise environment in TESTS.md. Our CI system (Github Actions) will not test your fork until a one-time approval takes place.

Your change, depending on its impact, may be released in the following ways:

  1. For impactful bug fixes, it can be released fairly quickly as a patch release.
  2. For noncritical bug fixes and new features, it will be incorporated into the next minor version release.
  3. For breaking changes (those changes that alter the public method signatures), more consideration must be made and alternatives may be considered, depending on upgrade difficulty and release schedule.

Please note that API features that are not generally available should not be merged/released without prior discussion with the maintainers. See docs/CONTRIBUTING Section "Adding API changes that are not generally available" for more information.

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

- [API documentation](https://developer.hashicorp.com/terraform/cloud-docs/api-docs/xxxx)
- [Related PR](https://github.com/terraform-providers/terraform-provider-tfe/pull/xxxx)

-->

## Output from tests
Including output from tests may require access to a TFE instance. Ignore this section if you have no environment to test against.

<!--
_Please run the tests locally for any files you changes and include the output here._
-->
```
$ TFE_ADDRESS="https://example" TFE_TOKEN="example" go test ./... -v -run TestFunctionsAffectedByChange

...
```
