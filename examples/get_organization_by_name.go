package main

import (
	"context"
	"log"

	"github.com/hashicorp/go-tfe"
)

func main() {
	client, err := tfe.NewClient(&tfe.Config{
		Token: "API TOKEN",
	})

	if err != nil {
		log.Fatalf("Error creating TFE client: %s", err)
	}

	response, err := client.API.Organizations().ByOrganization_Id("hashicorp").GetAsOrganization_GetResponse(context.Background(), nil)
	if err != nil {
		log.Fatalf("API returned an error status: %s", tfe.SummarizeAPIErrors(err))
	}

	hcpID := response.GetData().GetAttributes().GetHcpId()
	if hcpID == nil {
		log.Printf("[INFO] Organization is not an HCP Organization")
	}
}
