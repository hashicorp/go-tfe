Terraform Enterprise Go Client
==============================

[![Build Status](https://travis-ci.org/hashicorp/go-tfe.svg?branch=master)](https://travis-ci.org/hashicorp/go-tfe)
[![GitHub license](https://img.shields.io/github/license/hashicorp/go-tfe.svg)](https://github.com/hashicorp/go-tfe/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/hashicorp/go-tfe?status.svg)](https://godoc.org/github.com/hashicorp/go-tfe)
[![Go Report Card](https://goreportcard.com/badge/github.com/hashicorp/go-tfe)](https://goreportcard.com/report/github.com/hashicorp/go-tfe)
[![GitHub issues](https://img.shields.io/github/issues/hashicorp/go-tfe.svg)](https://github.com/hashicorp/go-tfe/issues)

This is an API client for [Terraform Enterprise](https://www.hashicorp.com/products/terraform).

## NOTE

The Terraform Enterprise API endpoints are in beta and are subject to change!
So that means this API client is also in beta and is also subject to change. We
will indicate any breaking changes by releasing new versions. Until the release
of v1.0, any minor version changes will indicate possible breaking changes. Patch
version changes will be used for both bugfixes and non-breaking changes.

## Coverage

Currently the following endpoints are supported:

- [x] [Accounts](https://www.terraform.io/docs/enterprise/api/account.html)
- [x] [Configuration Versions](https://www.terraform.io/docs/enterprise/api/configuration-versions.html)
- [x] [OAuth Clients](https://www.terraform.io/docs/enterprise/api/oauth-clients.html)
- [x] [OAuth Tokens](https://www.terraform.io/docs/enterprise/api/oauth-tokens.html)
- [x] [Organizations](https://www.terraform.io/docs/enterprise/api/organizations.html)
- [x] [Organization Tokens](https://www.terraform.io/docs/enterprise/api/organization-tokens.html)
- [x] [Policies](https://www.terraform.io/docs/enterprise/api/policies.html)
- [x] [Policy Sets](https://www.terraform.io/docs/enterprise/api/policy-sets.html)
- [x] [Policy Checks](https://www.terraform.io/docs/enterprise/api/policy-checks.html)
- [ ] [Registry Modules](https://www.terraform.io/docs/enterprise/api/modules.html)
- [x] [Runs](https://www.terraform.io/docs/enterprise/api/run.html)
- [x] [SSH Keys](https://www.terraform.io/docs/enterprise/api/ssh-keys.html)
- [x] [State Versions](https://www.terraform.io/docs/enterprise/api/state-versions.html)
- [x] [Team Access](https://www.terraform.io/docs/enterprise/api/team-access.html)
- [x] [Team Memberships](https://www.terraform.io/docs/enterprise/api/team-members.html)
- [x] [Team Tokens](https://www.terraform.io/docs/enterprise/api/team-tokens.html)
- [x] [Teams](https://www.terraform.io/docs/enterprise/api/teams.html)
- [x] [Variables](https://www.terraform.io/docs/enterprise/api/variables.html)
- [x] [Workspaces](https://www.terraform.io/docs/enterprise/api/workspaces.html)
- [ ] [Admin](https://www.terraform.io/docs/enterprise/api/admin/index.html)

## Installation

Installation can be done with a normal `go get`:

```
go get -u github.com/hashicorp/go-tfe
```

## Documentation

For complete usage of the API client, see the full [package docs](https://godoc.org/github.com/hashicorp/go-tfe).

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

orgs, err := client.Organizations.List(context.Background(), OrganizationListOptions{})
if err != nil {
	log.Fatal(err)
}
```

## Examples

The [examples](https://github.com/hashicorp/go-tfe/tree/master/examples) directory
contains a couple of examples. One of which is listed here as well:

```go
package main

import (
	"log"

	tfe "github.com/hashicorp/go-tfe"
)

func main() {
	config := &tfe.Config{
		Token: "insert-your-token-here",
	}

	client, err := tfe.NewClient(config)
	if err != nil {
		log.Fatal(err)
	}

	// Create a context
	ctx := context.Background()

	// Create a new organization
	options := tfe.OrganizationCreateOptions{
		Name:  tfe.String("example"),
		Email: tfe.String("info@example.com"),
	}

	org, err := client.Organizations.Create(ctx, options)
	if err != nil {
		log.Fatal(err)
	}

	// Delete an organization
	err = client.Organizations.Delete(ctx, org.Name)
	if err != nil {
		log.Fatal(err)
	}
}
```

## Running tests

### 1. (Optional) Create a policy sets repo

If you are planning to run the full suite of tests or work on policy sets, you'll need to set up a policy set repository in GitHub.

Your policy set repository will need the following: 
1. A policy set stored in a subdirectory `policy-sets/foo`
1. A branch other than master named `policies`
   
### 2. Set up environment variables

##### Required:
Tests are run against an actual backend so they require a valid backend address
and token.
1. `TFE_ADDRESS` - URL of a Terraform Cloud or Terraform Enterprise instance to be used for testing, including scheme. Example: `https://tfe.local`
1. `TFE_TOKEN` - A [user API token](https://www.terraform.io/docs/cloud/users-teams-organizations/users.html#api-tokens) for the Terraform Cloud or Terraform Enterprise instance being used for testing.

##### Optional:
1. `GITHUB_TOKEN` - [GitHub personal access token](https://help.github.com/en/github/authenticating-to-github/creating-a-personal-access-token-for-the-command-line). Required for running OAuth client tests.
1. `GITHUB_POLICY_SET_IDENTIFIER` - GitHub policy set repository identifier in the format `username/repository`. Required for running policy set tests.

You can set your environment variables up however you prefer. The following are instructions for setting up environment variables using [envchain](https://github.com/sorah/envchain).
   1. Make sure you have envchain installed. [Instructions for this can be found in the envchain README](https://github.com/sorah/envchain#installation).
   1. Pick a namespace for storing your environment variables. I suggest `go-tfe` or something similar.
   1. For each environment variable you need to set, run the following command:
      ```sh
      envchain --set YOUR_NAMESPACE_HERE ENVIRONMENT_VARIABLE_HERE
      ```
      **OR**
    
      Set all of the environment variables at once with the following command:
      ```sh
      envchain --set YOUR_NAMESPACE_HERE TFE_ADDRESS TFE_TOKEN GITHUB_TOKEN GITHUB_POLICY_SET_IDENTIFIER
      ```

### 3. Make sure run queue settings are correct

In order for the tests relating to queuing and capacity to pass, FRQ (fair run queuing) should be
enabled with a limit of 2 concurrent runs per organization on the Terraform Cloud or Terraform Enterprise instance you are using for testing.

### 4. Run the tests

#### Running all the tests
As running the all of the tests takes about ~20 minutes, make sure to add a timeout to your
command (as the default timeout is 10m).

##### With envchain:
```sh
$ envchain YOUR_NAMESPACE_HERE go test ./... -timeout=30m
```

##### Without envchain:
```sh
$ go test ./... -timeout=30m
```
#### Running specific tests

The commands below use notification configurations as an example.

##### With envchain:
```sh
$ envchain YOUR_NAMESPACE_HERE go test -run TestNotificationConfiguration -v ./...
```

##### Without envchain:
```sh
$ go test -run TestNotificationConfiguration -v ./...
```   

## Issues and Contributing

If you find an issue with this package, please report an issue. If you'd like,
we welcome any contributions. Fork this repository and submit a pull request.
