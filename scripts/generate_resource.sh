#!/bin/bash

set -euo pipefail

info() {
  printf "\r\033[00;35m$1\033[0m\n"
}

success() {
  printf "\r\033[00;32m$1\033[0m\n"
}

fail() {
  printf "\r\033[0;31m$1\033[0m\n"
}

__help="This script is used to quickly scaffold a resource in go-tfe. Simply provide a resource name as the first argument and it will generate standard boilerplate.
\n- The resource name cannot be plural.
\n- The resource name cannot contain numbers or special characters.
\nNote: This script cannot generate appropriate camelcasing for resource names.\n
Example usage: ./generate_resource.sh Organization
"

# Ensure a resource name is passed as an argument
if [[ "$#" -ne 1 ]]; then
    fail "Invalid number of arguments passed. Must specify a single resource name."
    exit 1
fi

arg="$1"
if [[ $arg = "-h" ]]; then
    echo -e $__help
    exit 1
fi


# Ensure the resource name isn't plural
if [[ "${arg: -1}" = "s" ]]; then
    fail "Error: Resource name cannot be plural."
    exit 1
fi

if [[ "${arg}" =~ [^a-zA-Z] ]]; then
    fail "Error: Resource name can only contain letters."
    exit 1
fi

resource=$(echo "$arg" | tr '[:upper:]' '[:lower:]')
plural_resource="${resource}s"

# Capitalized names
c_resource="$(tr '[:lower:]' '[:upper:]' <<< ${resource:0:1})${resource:1}"
c_plural_resource="$(tr '[:lower:]' '[:upper:]' <<< ${plural_resource:0:1})${plural_resource:1}"

# Option struct names
list_opts="${c_resource}ListOptions"
read_opts="${c_resource}ReadOptions"
create_opts="${c_resource}CreateOptions"
update_opts="${c_resource}UpdateOptions"

resource_list="${c_resource}List"
resource_id="${resource}ID"

info "Creating $resource.go"

cat <<EOF > $resource.go
package tfe

import (
  "context"
)

var _ $c_plural_resource = (*$plural_resource)(nil)

// $c_plural_resource describes all the $resource related methods that the Terraform
// Enterprise API supports
//
// TFE API docs: (TODO: ADD DOCS URL)
type $c_plural_resource interface {
  // List all $plural_resource.
  List(ctx context.Context, options *$list_opts) (*${resource_list}, error)

  // Create a $resource.
  Create(ctx context.Context, options $create_opts) (*${c_resource}, error)

  // Read a $resource by its ID.
  Read(ctx context.Context, $resource_id string) (*${c_resource}, error)

  // Read a $resource by its ID with options.
  ReadWithOptions(ctx context.Context, $resource_id string, options *$read_opts) (*${c_resource}, error)

  // Update a $resource.
  Update(ctx context.Context, $resource_id string, options $update_opts) (*${c_resource}, error)

  // Delete a $resource.
  Delete(ctx context.Context, $resource_id string) error
}

// $plural_resource implements $c_plural_resource
type $plural_resource struct {
  client *Client
}

// $resource_list represents a list of $plural_resource
type $resource_list struct {
  *Pagination
  Items []*${c_resource}
}

// $c_resource represents a Terraform Enterprise $resource
type $c_resource struct {
  ID string \`jsonapi:"primary,$plural_resource"\`
  // Add more fields here
}

// $list_opts represents the options for listing $plural_resource
type $list_opts struct {
  ListOptions

  // Add more list options here
}

// $create_opts represents the options for creating $plural_resource
type $create_opts struct {
  Type string \`jsonapi:"primary,$plural_resource"\`
  // Add more create options here
}

// $read_opts represents the options for reading a $resource
type $read_opts struct {
  // Add more read options here
}

// $update_opts represents the options for updating a $resource
type $update_opts struct {
  ID string \`jsonapi:"primary,$plural_resource"\`

  // Add more update options here
}

// List all $plural_resource
func (s *${plural_resource}) List(ctx context.Context, options *$list_opts) (*${resource_list}, error) {
  panic("not implemented")
}

// Create a new $resource with the given options.
func (s *${plural_resource}) Create(ctx context.Context, options $create_opts) (*${c_resource}, error) {
  panic("not implemented")
}

// Read a $resource by its ID.
func (s *${plural_resource}) Read(ctx context.Context, $resource_id string) (*${c_resource}, error) {
  panic("not implemented")
}


// Read a $resource by its ID with the given options.
func (s *${plural_resource}) ReadWithOptions(ctx context.Context, $resource_id string, options *$read_opts) (*${c_resource}, error) {
  panic("not implemented")
}

// Update a $resource by its ID.
func (s *${plural_resource}) Update(ctx context.Context, $resource_id string, options $update_opts) (*${c_resource}, error) {
  panic("not implemented")
}

// Delete a $resource by its ID.
func (s *${plural_resource}) Delete(ctx context.Context, $resource_id string) error {
  panic("not implemented")
}
EOF


info "Creating ${resource}_integration_test.go"

cat <<EOF > ${resource}_integration_test.go
//go:build integration
// +build integration

package tfe

import (
  "context"
  "testing"

  "github.com/stretchr/testify/assert"
  "github.com/stretchr/testify/require"
)

func Test${c_plural_resource}List(t *testing.T) {
  client := testClient(t)
  ctx := context.Background()

  // Create your test helper resources here
  t.Run("test not yet implemented", func(t *testing.T) {
    require.NotNil(t, nil)
  })
}

func Test${c_plural_resource}Read(t *testing.T) {
  client := testClient(t)
  ctx := context.Background()

  // Create your test helper resources here
  t.Run("test not yet implemented", func(t *testing.T) {
    require.NotNil(t, nil)
  })
}

func Test${c_plural_resource}Create(t *testing.T) {
  client := testClient(t)
  ctx := context.Background()

  // Create your test helper resources here
  t.Run("test not yet implemented", func(t *testing.T) {
    require.NotNil(t, nil)
  })
}

func Test${c_plural_resource}Update(t *testing.T) {
  client := testClient(t)
  ctx := context.Background()

  // Create your test helper resources here
  t.Run("test not yet implemented", func(t *testing.T) {
    require.NotNil(t, nil)
  })
}
EOF


# Ensure the file is properly formatted and our code is A-OK.
info "Formatting $resource.go and ${resource}_integration_test.go"
gofmt -w $resource.go ${resource}_integration_test.go
info "Vetting newly created files"
go vet ./...

success "✅ Created new resource: $c_resource"
