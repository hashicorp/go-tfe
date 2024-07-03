// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"context"
	"log"
	"time"

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

	// Create a new workspace
	w, err := client.Workspaces.Create(ctx, "org-name", tfe.WorkspaceCreateOptions{
		Name:          tfe.String("my-app-tst"),
		AutoDestroyAt: tfe.NullableTime(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
	})
	if err != nil {
		log.Fatal(err)
	}

	// Update the workspace
	w, err = client.Workspaces.Update(ctx, "org-name", w.Name, tfe.WorkspaceUpdateOptions{
		AutoApply:        tfe.Bool(false),
		TerraformVersion: tfe.String("0.11.1"),
		WorkingDirectory: tfe.String("my-app/infra"),
		AutoDestroyAt:    tfe.NullableTime(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
	})
	if err != nil {
		log.Fatal(err)
	}

	// Disable auto destroy
	w, err = client.Workspaces.Update(ctx, "org-name", w.Name, tfe.WorkspaceUpdateOptions{
		AutoDestroyAt: tfe.NullTime(),
	})
	if err != nil {
		log.Fatal(err)
	}
}
