package main

import (
	"context"
	"log"

	tfe "github.com/hashicorp/go-tfe"
)

func main() {
	config := &tfe.Config{
		Token: "insert-your-token-here",
	}

	client, err := tfe.NewClient(config)
	if err != nil {
		log.Fatal(err)
	}

	// Create a context
	ctx := context.Background()

	otl, _ := client.OAuthTokens.List(ctx, "org-name", tfe.OAuthTokenListOptions{})

	// Publish a module
	options := tfe.RegistryModulePublishOptions{
		VCSRepo: &tfe.VCSRepo{Identifier: "vcs-identifier", OAuthTokenID: otl.Items[0].ID},
	}
	rm, err := client.RegistryModules.Publish(ctx, options)
	if err != nil {
		log.Fatal(err)
	}

	//Delete module
	err = client.RegistryModules.Delete(ctx, "org-name", rm.Name, "", "")
	if err != nil {
		log.Fatal(err)
	}
}
