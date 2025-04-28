package main

import (
	"context"
	"log"
	"os"

	"github.com/hashicorp/go-tfe"
)

func main() {
	client, err := tfe.NewClient(&tfe.Config{
		Token:   os.Getenv("EXAMPLE_TOKEN"),
		Address: os.Getenv("EXAMPLE_ADDRESS"),
	})

	if err != nil {
		log.Fatalf("Error creating TFE client: %s", err)
	}

	response, err := client.API.Organizations().ByOrganization_Id("hashicorp").GetAsOrganization_GetResponse(context.Background(), nil)
	if err != nil {
		log.Fatalf("API returned an error status: %s", tfe.SummarizeAPIErrors(err))
	}

	log.Printf("Just fetched organization: %s, which was created at %s", *response.GetData().GetAttributes().GetName(), *response.GetData().GetAttributes().GetCreatedAt())
}
