# Contributing to go-tfe

If you find an issue with this package, please create an issue in GitHub. If you'd like, we welcome any contributions. Fork this repository and submit a pull request.

## Adding new functionality or fixing relevant bugs

If you are adding a new endpoint, make sure to update the [coverage list in README.md](../README.md#API-Coverage) where we keep a list of the TFC APIs that this SDK supports.

If you are making relevant changes that is worth communicating to our users, please include a note about it in our CHANGELOG.md. You can include it as part of the PR where you are submitting your changes.

CHANGELOG.md should have the next minor version listed as `# v1.X.0 (Unreleased)` and any changes can go under there. But if you feel that your changes are better suited for a patch version (like a critical bug fix), you may list a new section for this version. You should repeat the same formatting style introduced by previous versions.

### Scoping pull requests that add new resources

There are instances where several new resources being added (i.e Workspace Run Tasks and Organization Run Tasks) are coalesced into one PR. In order to keep the review process as efficient and least error prone as possible, we ask that you please scope each PR to an individual resource even if the multiple resources you're adding share similarities. If joining multiple related PRs into one single PR makes more sense logistically, we'd ask that you organize your commit history by resource. A general convention for this repository is one commit for the implementation of the resource's methods, one for the integration test, and one for cleanup and housekeeping (e.g modifying the changelog/docs, generating mocks, etc).

**Note HashiCorp Employees Only:** When submitting a new set of endpoints please ensure that one of your respective team members approves the changes as well before merging.

## Running the Linters Locally

1. Ensure you have [installed golangci-lint](https://golangci-lint.run/usage/install/#local-installation)
2. From the CLI, run `make lint`

## Writing Tests

The test suite contains many acceptance tests that are run against the latest version of Terraform Enterprise. You can read more about running the tests against your own Terraform Enterprise environment in [TESTS.md](TESTS.md). Our CI system (Github Actions) will not test your fork until a one-time approval takes place.

## Test Splitting

Our CI workflow makes use of multiple nodes to run our tests in a more efficient manner. To prevent your test from running across all nodes, you **must** add `skipIfNotCINode(t)` to your top level test before any other helper or test logic.

## Editor Settings

We've included VSCode settings to assist with configuring the go extension. For other editors that integrate with the [Go Language Server](https://github.com/golang/tools/tree/master/gopls), the main thing to do is to add the `integration` build tags so that the test files are found by the language server. See `.vscode/settings.json` for more details.

## Generating Mocks
Ensure you have installed the [mockgen](https://github.com/golang/mock) tool.

You'll need to generate mocks if an existing endpoint method is modified or a new method is added. To generate mocks, simply run `./generate_mocks.sh` If you're adding a new API resource to go-tfe, you'll need to add the command to `generate_mocks.sh`. For example if someone creates `example_resource.go`, you'll add:

```
mockgen -source=example_resource.go -destination=mocks/example_resource_mocks.go -package=mocks
```

Alternatively, you can use the Makefile target `mocks` to automate the steps:

```
FILENAME=example_resource.go make mocks
```

## Adding API changes that are not generally available

In general, beta features should not be merged/released until generally available (GA). However, the maintainers recognize almost any reason to release beta features on a case-by-case basis. These could include: partial customer availability, software dependency, or any reason short of feature completeness.

Beta features, if released, should be clearly commented:

```
// **Note: This field is still in BETA and subject to change.**
ExampleNewField *bool `jsonapi:"attr,example-new-field,omitempty"`
```

When adding test cases, you can temporarily use the skipIfBeta() test helper to omit beta features from running in CI.

```
t.Run("with nested changes trigger", func (t *testing.T) {
  skipIfNotCINode(t)
  skipIfBeta(t)
  options := WorkspaceCreateOptions {
     // rest of required fields here
     ExampleNewField: Bool(true),
   }
  // the rest of your test logic here
})
```

**Note**: After your PR has been merged, and the feature either reaches general availability, you should remove the `skipIfBeta()` flag.

## Adding New Endpoints

### Scaffolding a Resource

When creating a new resource you can use the helper script `generate_resource` to quickly setup boilerplate code for adding a new set of endpoints related to that resource:

#### Running the script directly
```sh
cd ./scripts/generate_resource
go run . example_resource
```

#### Running the Makefile target `generate`
```sh
RESOURCE=example_resource make generate
```

### Guidelines for Adding New Endpoints

* An interface should cover one RESTful resource, which sometimes involves two or more endpoints.
* We require that each resource interface provides compile-time proof that it has been implemented.
* You'll need to add an integration test that covers each method of the resource's interface.
* Option structs serve as a proxy for either passing query params or request bodies:
    - `ListOptions` and `ReadOptions` are values passed as query parameters.
    - `CreateOptions` and `UpdateOptions` represent the request body.
* URL parameters should be defined as method parameters.
* Any resource specific errors must be defined in `errors.go`

Here is a more comprehensive example of what a resource looks like when implemented. The helper script `generate_resource` generates a subset of this example, focusing only on the core details that are required across all resources in go-tfe.

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
	ID            string  `jsonapi:"primary,examples"`
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
	Type string `jsonapi:"primary,examples"`

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
	Type string `jsonapi:"primary,examples"`

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
	req, err := s.client.NewRequest("POST", u, &options)
	if err != nil {
		return nil, err
	}

	r := &Example{}
	err = req.Do(ctx, r)
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
	req, err := s.client.NewRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	el := &ExampleList{}
	err = req.Do(ctx, el)
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
	req, err := s.client.NewRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	e := &Example{}
	err = req.Do(ctx, e)
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
	req, err := s.client.NewRequest("PATCH", u, &options)
	if err != nil {
		return nil, err
	}

	r := &Example{}
	err = req.Do(ctx, r)
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
	req, err := s.client.NewRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
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

## Rebasing a fork to trigger CI (Maintainers Only)

Pull requests that originate from a fork will not have access to this repository's secrets, thus resulting in the inability to test against our CI instance. In order to trigger the CI action workflow, there is a handy script `./scripts/rebase-fork.sh` that automates the steps for you. It will:

* Checkout the fork PR locally onto your machine and create a new branch prefixed as follows: `local/{name_of_fork_branch}`
* Push your newly created branch to Github, appending an empty commit stating the original branch that was rebased.
* Copy the contents of the fork's pull request (title and description) and create a new pull request, triggering the CI workflow.

**Important**: This script does not handle subsequent commits to the original PR and would require you to rebase them manually. Therefore, it is important that authors include test results in their description and changes are approved before this script is executed.

This script depends on `gh` and `jq`. It also requires you to `gh auth login`, providing a SSO-authorized personal access token with the following scopes enabled:

- repo
- read:org
- read:discussion

### Example Usage

```sh
./scripts/rebase-fork.sh 557
```

