package tfe

import (
	"bytes"
	"context"
	"log"

	slug "github.com/hashicorp/go-slug"
)

func ExampleConfigurationVersions_UploadTarGzip() {
	ctx := context.Background()
	client, err := NewClient(&Config{
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
	// Pass in a path
	_, err = packer.Pack("test-fixtures/config", rawConfig)
	if err != nil {
		log.Fatal(err)
	}

	// Create a configuration version
	cv, err := client.ConfigurationVersions.Create(ctx, "ws-12345678", ConfigurationVersionCreateOptions{
		AutoQueueRuns: Bool(false),
	})
	if err != nil {
		log.Fatal(err)
	}

	// Upload the buffer
	err = client.ConfigurationVersions.UploadTarGzip(ctx, cv.UploadURL, rawConfig)
	if err != nil {
		log.Fatal(err)
	}
}

func ExampleRegistryModules_UploadTarGzip() {
	ctx := context.Background()
	client, err := NewClient(&Config{
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
	// Pass in a path
	_, err = packer.Pack("test-fixtures/config", rawConfig)
	if err != nil {
		log.Fatal(err)
	}

	// Create a registry module
	rm, err := client.RegistryModules.Create(ctx, "hashicorp", RegistryModuleCreateOptions{
		Name:         String("my-module"),
		Provider:     String("provider"),
		RegistryName: PrivateRegistry,
	})
	if err != nil {
		log.Fatal(err)
	}

	opts := RegistryModuleCreateVersionOptions{
		Version: String("1.1.0"),
	}

	// Create a registry module version
	rmv, err := client.RegistryModules.CreateVersion(ctx, RegistryModuleID{
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
