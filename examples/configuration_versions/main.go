// Copyright IBM Corp. 2018, 2025
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"bytes"
	"context"
	"log"

	"github.com/hashicorp/go-slug"
	tfe "github.com/hashicorp/go-tfe"
)

func main() {
	ctx := context.Background()
	client, err := tfe.NewClient(&tfe.Config{
		RetryServerErrors: true,
	})
	if err != nil {
		log.Fatal(err)
	}

	packer, err := slug.NewPacker(
		slug.DereferenceSymlinks(),  // dereferences symlinks
		slug.ApplyTerraformIgnore(), // ignores paths specified in .terraformignore
	)
	if err != nil {
		log.Fatal(err)
	}

	rawConfig := bytes.NewBuffer(nil)
	// Pass in a path
	_, err = packer.Pack("test-fixtures/config", rawConfig)
	if err != nil {
		log.Fatal(err)
	}

	// Create a configuration version
	cv, err := client.ConfigurationVersions.Create(ctx, "ws-12345678", tfe.ConfigurationVersionCreateOptions{
		AutoQueueRuns: tfe.Bool(false),
	})
	if err != nil {
		log.Fatal(err)
	}

	// Upload the configuration
	err = client.ConfigurationVersions.UploadTarGzip(ctx, cv.UploadURL, rawConfig)
	if err != nil {
		log.Fatal(err)
	}
}
