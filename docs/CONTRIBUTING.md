# Contributing to go-tfe

If you find an issue with this package, please create an issue in GitHub. If you'd like, we welcome any contributions. Fork this repository and submit a pull request.

## Running the Linters Locally

1. Ensure you have have [installed golangci-lint](https://golangci-lint.run/usage/install/#local-installation)
2. From the CLI, run `golangci-lint run`

## Writing Tests

The test suite contains many acceptance tests that are run against the latest version of Terraform Enterprise. You can read more about running the tests against your own Terraform Enterprise environment in [TESTS.md](TESTS.md). Our CI system (Circle) will not test your fork unless you are an authorized employee, so a HashiCorp maintainer will initiate the tests or you and report any missing tests or simple problems. In order to speed up this process, it's not uncommon for your commits to be incorporated into another PR that we can commit test changes to.

## Editor Settings

We've included VSCode settings to assist with configuring the go extension. For other editors that integrate with the [Go Language Server](https://github.com/golang/tools/tree/master/gopls), the main thing to do is to add the `integration` build tags so that the test files are found by the language server. See `.vscode/settings.json` for more details.

## Generating Mocks

You'll need to generate mocks if an existing endpoint method is modified or a new method is added. To generate mocks, simply run `./generate_mocks.sh` If you're adding a new API resource to go-tfe, you'll need to add the command to `generate_mocks.sh`. For example if someone creates `example_resource.go`, you'll add:

```
mockgen -source=example_resource.go -destination=mocks/example_resource_mocks.go -package=mocks
```

## Adding a New Endpoint

Here you will find a scaffold to get you started when building a json:api RESTful endpoint. The comments are meant to guide you but should be replaced with endpoint-specific and type-specific documentation. Additionally, you'll need to add an integration test that covers each method of the main interface.

In general, an interface should cover one RESTful resource, which sometimes involves two or more endpoints. Add all new modules to the tfe package.

```go
package tfe

import (
	"context"
	"errors"
	"fmt"
	"net/url"
)

var ErrInvalidExampleID = errors.New("invalid value for example ID") // move this line to errors.go

// Compile-time proof of interface implementation
var _ ExampleResource = (*example)(nil)

// Example represents all the example methods in the context of an organization
// that the Terraform Cloud/Enterprise API supports.
// If this API is in beta or pre-release state, include that warning here.
type ExampleResource interface {
	// Create an example for an organization
	Create(ctx context.Context, organization string, options ExampleCreateOptions) (*Example, error)

	// List all examples for an organization
	List(ctx context.Context, organization string, options *ExampleListOptions) (*ExampleList, error)

	// Read an organization's example by ID
	Read(ctx context.Context, exampleID string) (*Example, error)

	// Read an organization's example by ID with given options
	ReadWithOptions(ctx context.Context, exampleID string, options *ExampleReadOptions) (*Example, error)

	// Update an example for an organization
	Update(ctx context.Context, exampleID string, options ExampleUpdateOptions) (*Example, error)

	// Delete an organization's example
	Delete(ctx context.Context, exampleID string) error
}

// example implements Example
type example struct {
	client *Client
}

// Example represents a TFC/E example resource
type Example struct {
	ID            string  `jsonapi:"primary,tasks"`
	Name          string  `jsonapi:"attr,name"`
	URL           string  `jsonapi:"attr,url"`
	OptionalValue *string `jsonapi:"attr,optional-value,omitempty"`

	Organization *Organization `jsonapi:"relation,organization"`
}

// ExampleCreateOptions represents the set of options for creating an example
type ExampleCreateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,tasks"`

	// Required: The name of the example
	Name string `jsonapi:"attr,name"`

	// Required: The URL to send in the example
	URL string `jsonapi:"attr,url"`

	// Optional: An optional value that is omitted if empty
	OptionalValue *string `jsonapi:"attr,optional-value,omitempty"`
}

// ExampleIncludeOpt represents the available options for include query params.
// https://www.terraform.io/cloud-docs/api-docs/examples#list-examples (replace this URL with the actual documentation URL)
type ExampleIncludeOpt string

const (
	ExampleOrganization ExampleIncludeOpt = "organization"
	ExampleRun ExampleIncludeOpt = "run"
)

// ExampleListOptions represents the set of options for listing examples
type ExampleListOptions struct {
	ListOptions

	// Optional: A list of relations to include with an example. See available resources:
	// https://www.terraform.io/cloud-docs/api-docs/examples#list-examples (replace this URL with the actual documentation URL)
	Include []ExampleIncludeOpt `url:"include,omitempty"`
}

// ExampleList represents a list of examples
type ExampleList struct {
	*Pagination
	Items []*Example
}

// ExampleReadOptions represents the set of options for reading an example
type ExampleReadOptions struct {
	// Optional: A list of relations to include with an example. See available resources:
	// https://www.terraform.io/cloud-docs/api-docs/examples#list-examples (replace this URL with the actual documentation URL)
	Include []RunTaskIncludeOpt `url:"include,omitempty"`
}

// ExampleUpdateOptions represents the set of options for updating an organization's examples
type ExampleUpdateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,tasks"`

	// Optional: The name of the example, defaults to previous value
	Name *string `jsonapi:"attr,name,omitempty"`

	// Optional: The URL to send a example payload, defaults to previous value
	URL *string `jsonapi:"attr,url,omitempty"`

	// Optional: An optional value
	OptionalValue *string `jsonapi:"attr,optional-value,omitempty"`
}

// Create is used to create a new example for an organization
func (s *example) Create(ctx context.Context, organization string, options ExampleCreateOptions) (*Example, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}

	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("organizations/%s/tasks", url.QueryEscape(organization))
	req, err := s.client.newRequest("POST", u, &options)
	if err != nil {
		return nil, err
	}

	r := &Example{}
	err = s.client.do(ctx, req, r)
	if err != nil {
		return nil, err
	}

	return r, nil
}

// List all the examples for an organization
func (s *example) List(ctx context.Context, organization string, options *ExampleListOptions) (*ExampleList, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}
	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("organizations/%s/examples", url.QueryEscape(organization))
	req, err := s.client.newRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	el := &ExampleList{}
	err = s.client.do(ctx, req, el)
	if err != nil {
		return nil, err
	}

	return el, nil
}

// Read is used to read an organization's example by ID
func (s *example) Read(ctx context.Context, exampleID string) (*Example, error) {
	return s.ReadWithOptions(ctx, exampleID, nil)
}

// Read is used to read an organization's example by ID with options
func (s *example) ReadWithOptions(ctx context.Context, exampleID string, options *ExampleReadOptions) (*Example, error) {
	if !validStringID(&exampleID) {
		return nil, ErrInvalidExampleID
	}
	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("examples/%s", url.QueryEscape(exampleID))
	req, err := s.client.newRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	e := &Example{}
	err = s.client.do(ctx, req, e)
	if err != nil {
		return nil, err
	}

	return e, nil
}

// Update an existing example for an organization by ID
func (s *example) Update(ctx context.Context, exampleID string, options ExampleUpdateOptions) (*Example, error) {
	if !validStringID(&exampleID) {
		return nil, ErrInvalidExampleID
	}

	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("examples/%s", url.QueryEscape(exampleID))
	req, err := s.client.newRequest("PATCH", u, &options)
	if err != nil {
		return nil, err
	}

	r := &Example{}
	err = s.client.do(ctx, req, r)
	if err != nil {
		return nil, err
	}

	return r, nil
}

// Delete an existing example for an organization by ID
func (s *example) Delete(ctx context.Context, exampleID string) error {
	if !validStringID(&exampleID) {
		return ErrInvalidExampleID
	}

	u := fmt.Sprintf("examples/%s", exampleID)
	req, err := s.client.newRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	return s.client.do(ctx, req, nil)
}

func (o *ExampleUpdateOptions) valid() error {
	if o.Name != nil && !validString(o.Name) {
		return ErrRequiredName
	}

	if o.URL != nil && !validString(o.URL) {
		return ErrInvalidRunTaskURL
	}

	return nil
}

func (o *ExampleCreateOptions) valid() error {
	if !validString(&o.Name) {
		return ErrRequiredName
	}

	if !validString(&o.URL) {
		return ErrInvalidRunTaskURL
	}

	return nil
}

func (o *ExampleListOptions) valid() error {
	if o == nil {
		return nil // nothing to validate 
	}
	if err := validateExampleIncludeParams(o.Include); err != nil {
		return err
	}

	return nil
}

func (o *ExampleReadOptions) valid() error {
	if o == nil {
		return nil // nothing to validate 
	}
	if err := validateExampleIncludeParams(o.Include); err != nil {
		return err
	}

	return nil
}

func validateExampleIncludeParams(params []ExampleIncludeOpt) error {
	for _, p := range params {
		switch p {
		case ExampleOrganization, ExampleRun:
			// do nothing
		default:
			return ErrInvalidIncludeValue
		}
	}

	return nil
}
```

## Generating Mocks

To generate mocks, simply run `./generate_mocks.sh`. You'll need to do so if an existing endpoint method is modified or a new method is added. If you're adding a new API resource to go-tfe, you'll need to add the command to `generate_mocks.sh`. For example if someone creates `example_resource.go`, you'll add:

```
mockgen -source=example_resource.go -destination=mocks/example_resource_mocks.go -package=mocks
```

