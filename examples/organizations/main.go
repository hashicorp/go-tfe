// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"context"
	"log"

	tfe "github.com/optable/go-tfe"
)

func main() {
	config := &tfe.Config{
		Token:             "insert-your-token-here",
		RetryServerErrors: true,
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
