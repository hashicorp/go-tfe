// Copyright IBM Corp. 2018, 2026

package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/hashicorp/go-tfe/v2"
	"github.com/hashicorp/go-tfe/v2/api/organizations"
	abstractions "github.com/microsoft/kiota-abstractions-go"
)

func main() {
	client, err := tfe.NewClient(&tfe.Config{
		Token:   os.Getenv("TFE_TOKEN"),
		Address: os.Getenv("TFE_ADDRESS"),
	})
	if err != nil {
		log.Fatalf("Error creating TFE client: %s", err)
	}

	includeSubscriptions := organizations.SUBSCRIPTION_GETINCLUDEQUERYPARAMETERTYPE
	includeDefaultProject := organizations.DEFAULT_PROJECT_GETINCLUDEQUERYPARAMETERTYPE
	configuration := abstractions.RequestConfiguration[organizations.OrganizationsRequestBuilderGetQueryParameters]{
		QueryParameters: &organizations.OrganizationsRequestBuilderGetQueryParameters{
			Include: []organizations.GetIncludeQueryParameterType{
				includeSubscriptions,
				includeDefaultProject,
			},
		},
	}

	response, err := client.API.Organizations().Get(context.Background(), &configuration)
	if err != nil {
		log.Fatalf("Error listing organizations: %s", err)
	}

	for _, record := range response.GetIncluded() {
		if record.GetProjects() != nil {
			project := record.GetProjects()
			fmt.Printf("Included project: %s\n", *project.GetId())
		} else if record.GetSubscriptions() != nil {
			subscription := record.GetSubscriptions()
			fmt.Printf("Included subscription: %s\n", *subscription.GetId())
		} else {
			panic("This shouldn't happen")
		}
	}
}
