// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"bytes"
	"context"
	"log"

	"github.com/hashicorp/go-slug"
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

	packer, err := slug.NewPacker(
		slug.DereferenceSymlinks(),            // dereferences symlinks
		slug.ApplyTerraformIgnore(),           // ignores paths specified in .terraformignore
		slug.AllowSymlinkTarget("/some/path"), // allow certain symlink target paths
	)
	if err != nil {
		log.Fatal(err)
	}

	rawConfig := bytes.NewBuffer(nil)
	// Pass in the configuration path
	_, err = packer.Pack("test-fixtures/config", rawConfig)
	if err != nil {
		log.Fatal(err)
	}

	// Create a registry module
	rm, err := client.RegistryModules.Create(ctx, "hashicorp", tfe.RegistryModuleCreateOptions{
		Name:         tfe.String("my-module"),
		Provider:     tfe.String("provider"),
		RegistryName: tfe.PrivateRegistry,
	})
	if err != nil {
		log.Fatal(err)
	}

	opts := tfe.RegistryModuleCreateVersionOptions{
		Version: tfe.String("1.1.0"),
	}

	// Create a registry module version
	rmv, err := client.RegistryModules.CreateVersion(ctx, tfe.RegistryModuleID{
		Organization: "hashicorp",
		Name:         rm.Name,
		Provider:     rm.Provider,
	}, opts)
	if err != nil {
		log.Fatal(err)
	}

	uploadURL, ok := rmv.Links["upload"].(string)
	if !ok {
		log.Fatal("upload url must be a valid string")
	}
	// Upload the buffer
	err = client.RegistryModules.UploadTarGzip(ctx, uploadURL, rawConfig)
	if err != nil {
		log.Fatal(err)
	}
}
