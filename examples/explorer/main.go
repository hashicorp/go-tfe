// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"time"

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

	// Create a context
	ctx := context.Background()

	organization := "insert-your-organization-name"

	// Note: The following queries may not yield any results as the data available to query is dependent on your organization. Also results are paginated so the initial response may not reflect the full query result.

	// (#1) Workspaces Example: Give me all the workspace names that have a
	// current run status of "errored" AND are in project "foo"
	wql, err := client.Explorer.QueryWorkspaces(ctx, organization, tfe.ExplorerQueryOptions{
		Fields: []string{"workspace_name"},
		Filters: []*tfe.ExplorerQueryFilter{
			{
				Index:    0,
				Name:     "current_run_status",
				Operator: tfe.OpIs,
				Value:    "errored",
			},
			{
				Index:    1,
				Name:     "project_name",
				Operator: tfe.OpIs,
				Value:    "foo",
			},
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(wql.Items[0].WorkspaceName)

	// (#2) Modules Example: Give me all the modules that are being used by more than one workspace and sort them by number of workspaces DESC.
	mql, err := client.Explorer.QueryModules(ctx, organization, tfe.ExplorerQueryOptions{
		Sort: "-workspace_count",
		Filters: []*tfe.ExplorerQueryFilter{
			{
				Index:    0,
				Name:     "workspace_count",
				Operator: tfe.OpGreaterThan,
				Value:    "1",
			},
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(mql.Items[0].Name)
	fmt.Println(mql.Items[0].WorkspaceCount)

	// (#3) Providers Example: Give me all the providers that are being used by the workspace "staging-us-east-1"
	pql, err := client.Explorer.QueryProviders(ctx, organization, tfe.ExplorerQueryOptions{
		Filters: []*tfe.ExplorerQueryFilter{
			{
				Index:    0,
				Name:     "workspaces",
				Operator: tfe.OpContains,
				Value:    "staging-us-east-1",
			},
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(len(pql.Items))

	// (#4) Terraform Versions Example: Give me all of the workspaces
	// that are not using Terraform version 1.10.0
	tfql, err := client.Explorer.QueryTerraformVersions(ctx, organization, tfe.ExplorerQueryOptions{
		Filters: []*tfe.ExplorerQueryFilter{
			{
				Index:    0,
				Name:     "version",
				Operator: tfe.OpIsNot,
				Value:    "1.10.0",
			},
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(tfql.Items[0].Version)
	fmt.Println(tfql.Items[0].WorkspaceCount)

	// (#5) Export to CSV: Give me all the workspaces where health checks have
	// succeeded and have been updated since 2 days ago. Note: This method can also
	// be used for modules, providers and terraform version queries.
	since := time.Now().AddDate(0, 0, -2).Format(time.RFC3339)
	data, err := client.Explorer.ExportToCSV(ctx, organization, tfe.ExplorerQueryOptions{
		Filters: []*tfe.ExplorerQueryFilter{
			{
				Index:    0,
				Name:     "all_checks_succeeded",
				Operator: tfe.OpIs,
				Value:    "true",
			},
			{
				Index:    1,
				Name:     "updated_at",
				Operator: tfe.OpIsAfter,
				Value:    since,
			},
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	reader := csv.NewReader(bytes.NewReader(data))
	rows, err := reader.ReadAll()
	if err != nil {
		log.Fatal(err)
	}

	for _, r := range rows {
		// Do something with each row in the CSV
		fmt.Println(r)
	}
}
