// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

type ResourceTemplate struct {
	// Lower cased name of a resource, not plural
	Name string

	// Lower cased name of a resource, plural
	PluralName string

	// Name of resource model
	Resource string

	// Name of resource interface
	ResourceInterface string

	// Name of resource struct that implements resource interface
	ResourceStruct string

	// The resource ID
	ResourceID string

	// Struct tag name for (un)marshalling the JSON+API resource.
	PrimaryTag string

	ListOptions   string
	ReadOptions   string
	CreateOptions string
	UpdateOptions string
}

const helpTemplate = `
This script is used to quickly scaffold a resource in go-tfe. Simply provide a
resource name as the first argument and it will generate standard boilerplate.

Note: A resource name can only contain letters and underscores.

Allowed: policy_set, Run_task, orGanizatIon
Not Allowed: policy123, #user, my cool resource

If your resource contains multiple terms, e.g policy set, you must use an
underscore delimiter for each term in order to generate proper casing in your
code. For example, if you wanted to generate the policy set resource as
PolicySet you would pass policy_set as your argument.

Example usage: go run ./scripts/generate_resource/main.go policy_set`

const sourceTemplate = `
package tfe

import (
  "context"
)

var _ {{ .ResourceInterface }} = (*{{ .ResourceStruct }})(nil)

// {{ .ResourceInterface }} describes all the {{ .Name }} related methods that the Terraform
// Enterprise API supports
//
// TFE API docs: (TODO: ADD DOCS URL)
type {{ .ResourceInterface }} interface {
  // List all {{ .PluralName }}.
  List(ctx context.Context, options *{{ .ListOptions }}) (*{{ .Resource }}List, error)

  // Create a {{ .Name }}.
  Create(ctx context.Context, options {{ .CreateOptions }}) (*{{ .Resource }}, error)

  // Read a {{ .Name }} by its ID.
  Read(ctx context.Context, {{ .ResourceID }} string) (*{{ .Resource }}, error)

  // Read a {{ .Name }} by its ID with options.
  ReadWithOptions(ctx context.Context, {{ .ResourceID }} string, options *{{ .ReadOptions }}) (*{{ .Resource }}, error)

  // Update a {{ .Name }}.
  Update(ctx context.Context, {{ .ResourceID }} string, options {{ .UpdateOptions }}) (*{{ .Resource }}, error)

  // Delete a {{ .Name }}.
  Delete(ctx context.Context, {{ .ResourceID }} string) error
}

// {{ .ResourceStruct }} implements {{ .ResourceInterface }}
type {{ .ResourceStruct }} struct {
  client *Client
}

// {{ .Resource }}List represents a list of {{ .PluralName }}
type {{ .Resource }}List struct {
  *Pagination
  Items []*{{ .Resource }}
}

// {{ .Resource }} represents a Terraform Enterprise $resource
type {{ .Resource }} struct {
  ID string ` + "`jsonapi:\"primary," + `{{ .PrimaryTag }}` + "\"`" + `
  // Add more fields here
}

// {{ .ListOptions }} represents the options for listing {{ .PluralName }}
type {{ .ListOptions }} struct {
  ListOptions

  // Add more list options here
}

// {{ .CreateOptions }} represents the options for creating a {{ .Name }}
type {{ .CreateOptions }} struct {
  Type string ` + "`jsonapi:\"primary," + `{{ .PrimaryTag }}` + "\"`" + `
  // Add more create options here
}

// {{ .ReadOptions }} represents the options for reading a {{ .Name }}
type {{ .ReadOptions }} struct {
  // Add more read options here
}

// {{ .UpdateOptions }} represents the options for updating a {{ .Name }}
type {{ .UpdateOptions }} struct {
  ID string ` + "`jsonapi:\"primary," + `{{ .PrimaryTag }}` + "\"`" + `

  // Add more update options here
}

// List all {{ .PluralName }}.
func List(ctx context.Context, options *{{ .ListOptions }}) (*{{ .Resource }}List, error) {
    panic("not yet implemented")
}

// Create a {{ .Name }}.
func Create(ctx context.Context, options {{ .CreateOptions }}) (*{{ .Resource }}, error) {
    panic("not yet implemented")
}

// Read a {{ .Name }} by its ID.
func Read(ctx context.Context, {{ .ResourceID }} string) (*{{ .Resource }}, error) {
    panic("not yet implemented")
}

// Read a {{ .Name }} by its ID with options.
func ReadWithOptions(ctx context.Context, {{ .ResourceID }} string, options *{{ .ReadOptions }}) (*{{ .Resource }}, error) {
    panic("not yet implemented")
}

// Update a {{ .Name }}.
func Update(ctx context.Context, {{ .ResourceID }} string, options {{ .UpdateOptions }}) (*{{ .Resource }}, error) {
    panic("not yet implemented")
}

// Delete a {{ .Name }}.
func Delete(ctx context.Context, {{ .ResourceID }} string) error {
    panic("not yet implemented")
}`

const testTemplate = `package tfe

import (
  "context"
  "testing"

  "github.com/stretchr/testify/assert"
  "github.com/stretchr/testify/require"
)

func Test{{ .ResourceInterface }}List(t *testing.T) {
   client := testClient(t)
  ctx := context.Background()

  // Create your test helper resources here
  t.Run("test not yet implemented", func(t *testing.T) {
    require.NotNil(t, nil)
  })
}

func Test{{ .ResourceInterface }}Read(t *testing.T) {
   client := testClient(t)
  ctx := context.Background()

  // Create your test helper resources here
  t.Run("test not yet implemented", func(t *testing.T) {
    require.NotNil(t, nil)
  })
}

func Test{{ .ResourceInterface }}Create(t *testing.T) {
   client := testClient(t)
  ctx := context.Background()

  // Create your test helper resources here
  t.Run("test not yet implemented", func(t *testing.T) {
    require.NotNil(t, nil)
  })
}

func Test{{ .ResourceInterface }}Update(t *testing.T) {
   client := testClient(t)
  ctx := context.Background()

  // Create your test helper resources here
  t.Run("test not yet implemented", func(t *testing.T) {
    require.NotNil(t, nil)
  })
}`
