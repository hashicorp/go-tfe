# HCP Terraform and Terraform Enterprise Go SDK Client 2.0

[![Tests](https://github.com/hashicorp/go-tfe/actions/workflows/ci.yml/badge.svg)](https://github.com/hashicorp/go-tfe/actions/workflows/ci.yml)
[![GitHub license](https://img.shields.io/github/license/hashicorp/go-tfe.svg)](https://github.com/hashicorp/go-tfe/blob/main/LICENSE)
[![GoDoc](https://godoc.org/github.com/hashicorp/go-tfe?status.svg)](https://godoc.org/github.com/hashicorp/go-tfe)
[![Go Report Card](https://goreportcard.com/badge/github.com/hashicorp/go-tfe)](https://goreportcard.com/report/github.com/hashicorp/go-tfe)
[![GitHub issues](https://img.shields.io/github/issues/hashicorp/go-tfe.svg)](https://github.com/hashicorp/go-tfe/issues)

The official Go API client for [HCP Terraform and Terraform Enterprise](https://www.hashicorp.com/products/terraform).

This client supports the [HCP Terraform V2 API](https://developer.hashicorp.com/terraform/cloud-docs/api-docs).
As Terraform Enterprise is a self-hosted distribution of HCP Terraform, this
client supports both HCP Terraform and Terraform Enterprise use cases.

## Quick Start

### Installation

To install the client, use `go get`:

```bash
go get github.com/hashicorp/go-tfe/v2
```

### Basic Usage

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/hashicorp/go-tfe/v2"
	"github.com/microsoft/kiota-abstractions-go/serialization"
)

func main() {
	client, err := tfe.NewClient(&tfe.Config{
		Token:   os.Getenv("TFE_TOKEN"),
		Address: os.Getenv("TFE_ADDRESS"),
	})
	if err != nil {
		log.Fatalf("Error creating client: %v", err)
	}

	ctx := context.Background()

	// Get a list of organizations
	response, err := client.API.Organizations().Get(ctx, nil)
	if err != nil {
		log.Fatalf("Error getting organizations: %v", err)
	}

	// Serialize the response to JSON for display
	buffer, err := serialization.SerializeToJson(response)
	if err != nil {
		log.Fatalf("Error serializing response: %s", err)
	}

	fmt.Println(string(buffer))
}
```

## Version Information

Almost always, minor version changes will indicate backwards-compatible features and enhancements. Occasionally, function signature changes that reflect a bug fix may appear as a minor version change. Patch version changes will be used for bug fixes, performance improvements, and otherwise unimpactful changes.

## Reference Documentation

### Client Configuration

All configuration is done using the `NewClient` function. See [Configuration Options Reference](#configuration-options-reference) for all options and defaults.

```go
client, err := tfe.NewClient(&tfe.Config{
  Token:   os.Getenv("TFE_TOKEN"),
  Address: os.Getenv("TFE_ADDRESS"),
})
```

### Path-based Interface

Every client interface starting with `API` uses a path-based naming convention followed by the
method of the operation on that path. Let's take a look at some examples:

```go
// Simple, unparameterized path GET /account/details
response, err := client.API.Account().Details().Get(ctx, nil)

// Parameterized path POST /organizations/{organization_name}/projects
response, err := client.API.Organizations().ByOrganization_name("foo").Projects().Post(ctx, newProjectRequestBody(), nil)
```

Use the [API reference](https://developer.hashicorp.com/terraform/cloud-docs/api-docs) to explore
all the available paths and operations.

### Request Bodies

For all POST/PATCH endpoints that accept a request body, it will be necessary to construct the appropriate
data envelope value for the operation and parse values for type enums. All operations follow a similar pattern.

```go
func mustParseCategory(category string) *models.Vars_attributes_category {
	cat, err := models.ParseVars_attributes_category(category)
	if err != nil {
		panic("cannot parse category \"" + category + "\"")
	}
	result := cat.(*models.Vars_attributes_category)
	return result
}

// NewVar creates a new models.VarsEnvelope for creating a variable from parameters.
func NewVar(key, value, category string, sensitive bool) *models.VarsEnvelope {
	hcl := false
	attrib := &models.Vars_attributes{}
	attrib.SetKey(&key)
	attrib.SetValue(&value)
	attrib.SetSensitive(&sensitive)
	attrib.SetHcl(&hcl)
	attrib.SetCategory(mustParseCategory(category))

	data := &models.Vars{}
	data.SetAttributes(attrib)

	body := &models.VarsEnvelope{}
	body.SetData(data)

	return body
}
```

### Query Parameters

Sometimes, you'll want to add query parameters to a GET request, such as `include=subscription` when
fetching organizations. Each operation defines their available parameters in a package based on
its path:

```go
import (
	"github.com/hashicorp/go-tfe/v2/api/organizations"

	abstractions "github.com/microsoft/kiota-abstractions-go"
)

// Include subscriptions in the response by setting the include query parameter
includeSubscriptions := organizations.SUBSCRIPTION_GETINCLUDEQUERYPARAMETERTYPE
req := abstractions.RequestConfiguration[organizations.OrganizationsRequestBuilderGetQueryParameters]{
	QueryParameters: &organizations.OrganizationsRequestBuilderGetQueryParameters{
		Include: &includeSubscriptions,
	},
}

response, err := client.API.Organizations().Get(ctx, &req)
```

### Inspecting Response Headers

```go
import (
	"github.com/microsoft/kiota-abstractions-go"
	khttp "github.com/microsoft/kiota-http-go"
)

// 1. Create the HeadersInspectionRequestOption
inspectionOptions := khttp.NewHeadersInspectionOptions()
inspectionOptions.InspectResponseHeaders = true

// 2. Create/add the option to the RequestInformation object for the request
req := abstractions.RequestConfiguration[abstractions.DefaultQueryParameters]{
	Options: []abstractions.RequestOption{inspectionOptions},
}

// 3. Execute the request
_, err = client.API.Account().Details().Get(ctx, &req)
if err != nil {
	log.Fatalf("Error getting account details: %s", err)
	return 1
}

// 4. Access the response headers from the HeadersInspectionRequestOption
headers := inspectionOptions.GetResponseHeaders()
for _, key := range headers.ListKeys() {
	log.Printf("%s: %v", key, headers.Get(key))
}
```

## Configuration Options Reference

All configuration fields defined by `tfe.Config`

| Option              | Description                                                                               | Default                    |
|---------------------|-------------------------------------------------------------------------------------------|----------------------------|
| `Token`             | (Required) The API token used for authentication                                          |                            |
| `Address`           | The address URI of the TFE/HCPT service                                                   | `https://app.terraform.io` |
| `BasePath`          | The base endpoint path                                                                    | `/api/v2`                  |
| `Headers`           | `net/http` Header values to send with every request.                                      |                            |
| `RetryServerErrors` | Whether or not to retry 5XX errors automatically, up to RetryMaxRetries times.            | `false`                    |
| `RetryMaxRetries`   | The number of times to retry server errors.                                               | `5`                        |
| `RetryRateLimited`  | Whether or not to retry 429 errors automatically, at the interval specified by the server | `false`                    |
| `RetryHook`         | A callback invoked _before_ the next retry after a server error.                          |                            |

## Examples

See the [examples/ directory](https://github.com/hashicorp/go-tfe/tree/main/v2/examples) for runnable
example code.

## Documentation

For complete usage of the API client, see the [full package docs](https://pkg.go.dev/github.com/hashicorp/go-tfe/v2).

## Issues and Contributing

This API client is a wrapper around a client generated from an OpenAPI specification so it may be
likely there is an issue in the upstream API definition. To contribute to the wrapper portion of the
API client, see [CONTRIBUTING.md](docs/CONTRIBUTING.md)

## Updating the SDK Client from Spec

Run `make api`

## Releases

Releases are automated and are updated around once per week, pending platform API changes.
