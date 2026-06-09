// Copyright IBM Corp. 2018, 2026
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	tfe "github.com/hashicorp/go-tfe"
	"github.com/hashicorp/go-tfe/testserver"
)

func main() {
	srv := testserver.New()
	defer srv.Close()

	srv.SeedOrganization("demo-org", "demo@example.com")
	if _, err := srv.SeedWorkspace("demo-org", "demo-workspace"); err != nil {
		log.Fatal(err)
	}

	client, err := tfe.NewClient(srv.ClientConfig())
	if err != nil {
		log.Fatal(err)
	}

	orgs, err := client.Organizations.List(context.Background(), nil)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("go-tfe testserver running\n\n")
	fmt.Printf("URL:   %s\n", srv.URL())
	fmt.Printf("Token: %s\n\n", srv.Token())
	fmt.Printf("Seeded organizations: %d\n\n", len(orgs.Items))

	fmt.Println("Try these commands:")
	fmt.Printf("  curl -H 'Authorization: Bearer %s' %s\n", srv.Token(), srv.URL())
	fmt.Printf("  curl -H 'Authorization: Bearer %s' %s/api/v2/organizations\n", srv.Token(), srv.URL())
	fmt.Printf("  curl -H 'Authorization: Bearer %s' %s/api/v2/organizations/demo-org/workspaces\n", srv.Token(), srv.URL())

	fmt.Println("\nPress Ctrl+C to stop.")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh
}
