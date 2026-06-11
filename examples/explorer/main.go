// Copyright IBM Corp. 2018, 2026
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"context"
	"fmt"
	"log"
	"os"

	tfe "github.com/hashicorp/go-tfe"
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

	ctx := context.Background()
	organization := "example-org"

	// Query workspaces across the organization, filtering by name and sorting
	// by the most recently updated. Field names and operators are passed as
	// strings and validated by the backend.
	result, err := client.Explorer.Query(ctx, organization, tfe.ExplorerQueryOptions{
		Type: tfe.ExplorerViewWorkspaces,
		Sort: "-workspace_updated_at",
		Filters: []tfe.ExplorerFilter{
			{
				Field:    "workspace_name",
				Operator: tfe.ExplorerOpContains,
				Values:   []string{"prod"},
			},
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	// Note: query parameters (filter/sort/fields) use snake_case field names,
	// but the response attributes are keyed in kebab-case.
	for _, record := range result.Items {
		if name, ok := record.Attributes["workspace-name"].(string); ok {
			fmt.Printf("%s\t%s\n", record.ID, name)
		}
	}

	// Export the same view as CSV.
	csv, err := client.Explorer.ExportCSV(ctx, organization, tfe.ExplorerQueryOptions{
		Type: tfe.ExplorerViewWorkspaces,
	})
	if err != nil {
		log.Fatal(err)
	}

	if err := os.WriteFile("workspaces.csv", csv, 0o644); err != nil {
		log.Fatal(err)
	}
}
