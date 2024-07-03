// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strings"

	tfe "github.com/optable/go-tfe"
)

func main() {
	action := flag.String("action", "", "Action (soft-delete|restore|permanently-delete")
	externalId := flag.String("external-id", "", "External ID of StateVersion or ConfigurationVersion")

	flag.Parse()

	if action == nil || *action == "" {
		log.Fatal("No Action provided")
	}

	if externalId == nil || *externalId == "" {
		log.Fatal("No external ID provided")
	}

	ctx := context.Background()
	client, err := tfe.NewClient(&tfe.Config{
		RetryServerErrors: true,
	})
	if err != nil {
		log.Fatal(err)
	}

	err = performAction(ctx, client, *action, *externalId)
	if err != nil {
		log.Fatalf("Error performing action: %v", err)
	}
}

func performAction(ctx context.Context, client *tfe.Client, action string, id string) error {
	externalIdParts := strings.Split(id, "-")
	switch externalIdParts[0] {
	case "cv":
		switch action {
		case "soft-delete":
			return client.ConfigurationVersions.SoftDeleteBackingData(ctx, id)
		case "restore":
			return client.ConfigurationVersions.RestoreBackingData(ctx, id)
		case "permanently-delete":
			return client.ConfigurationVersions.PermanentlyDeleteBackingData(ctx, id)
		default:
			return fmt.Errorf("unsupported action: %s", action)
		}
	case "sv":
		switch action {
		case "soft-delete":
			return client.StateVersions.SoftDeleteBackingData(ctx, id)
		case "restore":
			return client.StateVersions.RestoreBackingData(ctx, id)
		case "permanently-delete":
			return client.StateVersions.PermanentlyDeleteBackingData(ctx, id)
		default:
			return fmt.Errorf("unsupported action: %s", action)
		}
	default:
		return fmt.Errorf("unsupported external ID: %s", id)
	}
	return nil
}
