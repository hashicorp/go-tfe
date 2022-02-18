Terraform Cloud/Enterprise Go Client
==============================

[![Build Status](https://circleci.com/gh/hashicorp/go-tfe.svg?style=shield)](https://circleci.com/gh/hashicorp/go-tfe)
[![GitHub license](https://img.shields.io/github/license/hashicorp/go-tfe.svg)](https://github.com/hashicorp/go-tfe/blob/main/LICENSE)
[![GoDoc](https://godoc.org/github.com/hashicorp/go-tfe?status.svg)](https://godoc.org/github.com/hashicorp/go-tfe)
[![Go Report Card](https://goreportcard.com/badge/github.com/hashicorp/go-tfe)](https://goreportcard.com/report/github.com/hashicorp/go-tfe)
[![GitHub issues](https://img.shields.io/github/issues/hashicorp/go-tfe.svg)](https://github.com/hashicorp/go-tfe/issues)

The official Go API client for [Terraform Cloud/Enterprise](https://www.hashicorp.com/products/terraform).

This client supports the [Terraform Cloud V2 API](https://www.terraform.io/docs/cloud/api/index.html).
As Terraform Enterprise is a self-hosted distribution of Terraform Cloud, this
client supports both Cloud and Enterprise use cases. In all package
documentation and API, the platform will always be stated as 'Terraform
Enterprise' - but a feature will be explicitly noted as only supported in one or
the other, if applicable (rare).

## Version Information

Almost always, minor version changes will indicate backwards-compatible features and enhancements. Occasionally, function signature changes that reflect a bug fix may appear as a minor version change. Patch version changes will be used for bug fixes, performance improvements, and otherwise unimpactful changes.

## Installation

Installation can be done with a normal `go get`:

```
go get -u github.com/hashicorp/go-tfe
```

## Usage

```go
import tfe "github.com/hashicorp/go-tfe"
```

Construct a new TFE client, then use the various endpoints on the client to
access different parts of the Terraform Enterprise API. For example, to list
all organizations:

```go
config := &tfe.Config{
	Token: "insert-your-token-here",
}

client, err := tfe.NewClient(config)
if err != nil {
	log.Fatal(err)
}

orgs, err := client.Organizations.List(context.Background(), nil)
if err != nil {
	log.Fatal(err)
}
```

## Documentation

For complete usage of the API client, see the full [package docs](https://pkg.go.dev/github.com/hashicorp/go-tfe).

## Examples

See the [examples directory](https://github.com/hashicorp/go-tfe/tree/main/examples).

## Running tests

See [TESTS.md](https://github.com/hashicorp/go-tfe/tree/main/TESTS.md).

## Issues and Contributing

If you find an issue with this package, please report an issue. If you'd like,
we welcome any contributions. Fork this repository and submit a pull request.

## Releases

Documentation updates and test fixes that only touch test files don't require a release or tag. You can just merge these changes into `main` once they have been approved.

### Creating a release

1. [Create a new release in GitHub](https://help.github.com/en/github/administering-a-repository/creating-releases) by clicking on "Releases" and then "Draft a new release"
1. Set the `Tag version` to a new tag, using [Semantic Versioning](https://semver.org/) as a guideline. 
1. Set the `Target` as `main`.
1. Set the `Release title` to the tag you created, `vX.Y.Z`
1. Use the description section to describe why you're releasing and what changes you've made. You should include links to merged PRs. Use the following headers in the description of your release:
   - BREAKING CHANGES: Use this for any changes that aren't backwards compatible. Include details on how to handle these changes.
   - FEATURES: Use this for any large new features added, 
   - ENHANCEMENTS: Use this for smaller new features added
   - BUG FIXES: Use this for any bugs that were fixed.
   - NOTES: Use this section if you need to include any additional notes on things like upgrading, upcoming deprecations, or any other information you might want to highlight.

   Markdown example:

   ```markdown
   ENHANCEMENTS
   * Add description of new small feature (#3)[link-to-pull-request]

   BUG FIXES
   * Fix description of a bug (#2)[link-to-pull-request]
   * Fix description of another bug (#1)[link-to-pull-request]
   ```

1. Don't attach any binaries. The zip and tar.gz assets are automatically created and attached after you publish your release.
1. Click "Publish release" to save and publish your release.

