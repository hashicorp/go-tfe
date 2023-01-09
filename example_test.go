package tfe

import (
	"bytes"
	"context"
	"log"

	slug "github.com/hashicorp/go-slug"
)

func ExampleOrganizations() {
	config := &Config{
		Token:             "insert-your-token-here",
		RetryServerErrors: true,
	}

	client, err := NewClient(config)
	if err != nil {
		log.Fatal(err)
	}

	// Create a context
	ctx := context.Background()

	// Create a new organization
	options := OrganizationCreateOptions{
		Name:  String("example"),
		Email: String("info@example.com"),
	}

	org, err := client.Organizations.Create(ctx, options)
	if err != nil {
		log.Fatal(err)
	}

	// Delete an organization
	err = client.Organizations.Delete(ctx, org.Name)
	if err != nil {
		log.Fatal(err)
	}
}

func ExampleWorkspaces() {
	config := &Config{
		Token:             "insert-your-token-here",
		RetryServerErrors: true,
	}

	client, err := NewClient(config)
	if err != nil {
		log.Fatal(err)
	}

	// Create a context
	ctx := context.Background()

	// Create a new workspace
	w, err := client.Workspaces.Create(ctx, "org-name", WorkspaceCreateOptions{
		Name: String("my-app-tst"),
	})
	if err != nil {
		log.Fatal(err)
	}

	// Update the workspace
	w, err = client.Workspaces.Update(ctx, "org-name", w.Name, WorkspaceUpdateOptions{
		AutoApply:        Bool(false),
		TerraformVersion: String("0.11.1"),
		WorkingDirectory: String("my-app/infra"),
	})
	if err != nil {
		log.Fatal(err)
	}
}

func ExampleConfigurationVersions_UploadTarGzip() {
	ctx := context.Background()
	client, err := NewClient(&Config{
		Token:             "insert-your-token-here",
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
		Token:             "insert-your-token-here",
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
