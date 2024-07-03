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
		Token:             "insert Your user token here",
		RetryServerErrors: true,
	}

	client, err := tfe.NewClient(config)
	if err != nil {
		log.Fatal(err)
	}

	// Create a context
	ctx := context.Background()

	// Read Current User Details
	user, err := client.Users.ReadCurrent(ctx)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("%v", user)
}
