// Copyright IBM Corp. 2018, 2025
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"context"
	"log"

	tfe "github.com/hashicorp/go-tfe"

	"github.com/hashicorp/jsonapi"
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

	// Create a new project
	p, err := client.Projects.Create(ctx, "org-test", tfe.ProjectCreateOptions{
		Name: "my-app-tst",
	})
	if err != nil {
		log.Fatal(err)
	}

	// Update the project auto destroy activity duration
	p, err = client.Projects.Update(ctx, p.ID, tfe.ProjectUpdateOptions{
		AutoDestroyActivityDuration: jsonapi.NewNullableAttrWithValue("3d"),
	})
	if err != nil {
		log.Fatal(err)
	}

	// Disable auto destroy
	p, err = client.Projects.Update(ctx, p.ID, tfe.ProjectUpdateOptions{
		AutoDestroyActivityDuration: jsonapi.NewNullNullableAttr[string](),
	})
	if err != nil {
		log.Fatal(err)
	}

	err = client.Projects.Delete(ctx, p.ID)
	if err != nil {
		log.Fatal(err)
	}
}
