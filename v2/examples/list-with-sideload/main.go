// Copyright IBM Corp. 2018, 2026

package main

import (
	"context"
	"log"
	"os"

	"github.com/hashicorp/go-tfe/v2"
	"github.com/hashicorp/go-tfe/v2/api/models"
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

	for _, record := range response.GetData() {
		subscriptionID := record.GetRelationships().GetSubscription().GetData().GetId()

		subscription := tfe.FindSideloadedResource(subscriptionID, response.GetIncluded(), func(included organizations.OrganizationsGetResponse_OrganizationsGetResponse_includedable) models.Subscriptionsable {
			return included.GetSubscriptions()
		})

		if subscription != nil {
			log.Printf("Found subscription for %s: %s", *record.GetAttributes().GetName(), *subscription.GetId())
		} else {
			log.Fatalf("Subscription not found for ID: %s", *subscriptionID)
		}

		projectIDRel := record.GetRelationships().GetDefaultProject()
		if projectIDRel == nil {
			log.Printf("No default project for %s", *record.GetAttributes().GetName())
		} else {
			projectID := projectIDRel.GetData().GetId()
			project := tfe.FindSideloadedResource(projectID, response.GetIncluded(), func(included organizations.OrganizationsGetResponse_OrganizationsGetResponse_includedable) models.Projectsable {
				return included.GetProjects()
			})

			if project != nil {
				log.Printf("Found project for %s: %s", *record.GetAttributes().GetName(), *project.GetId())
			} else {
				log.Fatalf("Project not found for ID: %s", *projectID)
			}
		}
	}
}
