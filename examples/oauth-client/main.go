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
	// Create an oauth-client
	o, err := client.OAuthClients.Create(ctx, "org-name", tfe.OAuthClientCreateOptions{
		ServiceProvider: tfe.ServiceProvider(tfe.ServiceProviderGithubEE),
		HTTPURL:         tfe.String("http-url"),
		APIURL:          tfe.String("api-url"),
		Key:             tfe.String("key"),
		Secret:          tfe.String("secret"),
	})
	if err != nil {
		log.Fatal(err)
	}

	// Delete an oauth-client
	err = client.OAuthClients.Delete(ctx, o.ID)
	if err != nil {
		log.Fatal(err)
	}
}
