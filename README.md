HCP Terraform and Terraform Enterprise Go SDK Client 2.0
==============================

[![Tests](https://github.com/hashicorp/go-tfe/actions/workflows/ci.yml/badge.svg)](https://github.com/hashicorp/go-tfe/actions/workflows/ci.yml)
[![GitHub license](https://img.shields.io/github/license/hashicorp/go-tfe.svg)](https://github.com/hashicorp/go-tfe/blob/main/LICENSE)
[![GoDoc](https://godoc.org/github.com/hashicorp/go-tfe?status.svg)](https://godoc.org/github.com/hashicorp/go-tfe)
[![Go Report Card](https://goreportcard.com/badge/github.com/hashicorp/go-tfe)](https://goreportcard.com/report/github.com/hashicorp/go-tfe)
[![GitHub issues](https://img.shields.io/github/issues/hashicorp/go-tfe.svg)](https://github.com/hashicorp/go-tfe/issues)

The official Go API client for [HCP Terraform and Terraform Enterprise](https://www.hashicorp.com/products/terraform).

This client supports the [HCP Terraform V2 API](https://developer.hashicorp.com/terraform/cloud-docs/api-docs).
As Terraform Enterprise is a self-hosted distribution of HCP Terraform, this
client supports both HCP Terraform and Terraform Enterprise use cases. In all package
documentation and API, the platform will always be stated as 'Terraform
Enterprise' - but a feature will be explicitly noted as only supported in one or
the other, if applicable (rare).

## Version Information

Almost always, minor version changes will indicate backwards-compatible features and enhancements. Occasionally, function signature changes that reflect a bug fix may appear as a minor version change. Patch version changes will be used for bug fixes, performance improvements, and otherwise unimpactful changes.

## Basic Usage

Construct a new TFE client, then use the various endpoints on the client to
access different parts of the Terraform Enterprise API. The following example lists
all organizations.

```go
import (
	"context"
	"log"

	"github.com/hashicorp/go-tfe"
)

config := &tfe.Config{
	Token: "insert-your-token-here",   // Required
	Address: "https://tfe.local",      // Defaults to app.terraform.io
	RetryServerErrors: true,           // Defaults to false
}

client, err := tfe.NewClient(config)
if err != nil {
	log.Fatal(err)
}

org, err := client.API.Organizations().GetAsOrganizationsGetResponse(ctx, nil)
if err != nil {
	log.Fatalf("API returned an error status: %s", tfe.SummarizeAPIErrors(err))
	return 1
}
```

## Configuration Options Reference

All configuration fields defined by `tfe.Config`

| Option              | Description                                                      | Default            |
|---------------------|------------------------------------------------------------------|--------------------|
| `Address`           | The address URI of the TFE/HCPT service                          | `api.terraform.io` |
| `BasePath`          | The base endpoint path                                           | `/api/v2`          |
| `Token`             | The API token used for authentication.                           |                    |
| `Headers`           | `net/http` Header values to send with every request.             |                    |
| `RetryServerErrors` | Whether or not to retry 5XX errors automatically, up to 5 times. | `false`            |
| `RetryHook`         | A callback invoked _before_ the next retry after a server error. |                    |

## Reference Examples

See the [examples/ directory](https://github.com/hashicorp/go-tfe/tree/main/examples) for runnable
example code.

## Documentation

For complete usage of the API client, see the [full package docs](https://pkg.go.dev/github.com/hashicorp/go-tfe).

## Issues and Contributing

This API client is a wrapper around a client generated from an OpenAPI specification so it may be
likely there is an issue in the upstream API definition. To contribute to the wrapper portion of the
API client, see [CONTRIBUTING.md](docs/CONTRIBUTING.md)

## Releases

See [RELEASES.md](docs/RELEASES.md)
