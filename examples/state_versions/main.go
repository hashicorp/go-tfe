// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"context"
	"crypto/md5"
	"fmt"
	"log"
	"os"

	tfe "github.com/optable/go-tfe"
)

func main() {
	ctx := context.Background()
	client, err := tfe.NewClient(&tfe.Config{
		RetryServerErrors: true,
	})
	if err != nil {
		log.Fatal(err)
	}

	// Lock the workspace
	if _, err = client.Workspaces.Lock(ctx, "ws-12345678", tfe.WorkspaceLockOptions{}); err != nil {
		log.Fatal(err)
	}

	state, err := os.ReadFile("state.json")
	if err != nil {
		log.Fatal(err)
	}

	// Create upload options that does not contain a State attribute within the create options
	options := tfe.StateVersionUploadOptions{
		StateVersionCreateOptions: tfe.StateVersionCreateOptions{
			Lineage: tfe.String("493f7758-da5e-229e-7872-ea1f78ebe50a"),
			Serial:  tfe.Int64(int64(2)),
			MD5:     tfe.String(fmt.Sprintf("%x", md5.Sum(state))),
			Force:   tfe.Bool(false),
		},
		RawState: state,
	}

	// Upload a state version
	if _, err = client.StateVersions.Upload(ctx, "ws-12345678", options); err != nil {
		log.Fatal(err)
	}

	// Unlock the workspace
	if _, err = client.Workspaces.Unlock(ctx, "ws-12345678"); err != nil {
		log.Fatal(err)
	}
}
