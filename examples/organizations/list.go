package organizations

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/hashicorp/cli"
	"github.com/hashicorp/go-tfe"
	"github.com/hashicorp/go-tfe/api/organizations"
	"github.com/microsoft/kiota-abstractions-go/serialization"
)

type organizationListCommand struct{}

var _ cli.Command = organizationListCommand{}

func OrganizationListCommandFactory() (cli.Command, error) {
	return &organizationListCommand{}, nil
}

func (organizationListCommand) Help() string {
	return "List Organizations"
}

func (organizationListCommand) Synopsis() string {
	return "List Organizations"
}

func (organizationListCommand) Run(args []string) int {
	client, err := tfe.NewClient(&tfe.Config{
		Token:   os.Getenv("TFE_TOKEN"),
		Address: os.Getenv("TFE_ADDRESS"),
	})

	if err != nil {
		log.Fatalf("Error creating TFE client: %s", err)
		return 1
	}

	ctx := context.Background()

	// Include subscriptions in the response by setting the include query parameter
	includeSubscriptions := organizations.SUBSCRIPTION_GETINCLUDEQUERYPARAMETERTYPE
	c := organizations.OrganizationsRequestBuilderGetRequestConfiguration{
		QueryParameters: &organizations.OrganizationsRequestBuilderGetQueryParameters{
			IncludeAsGetIncludeQueryParameterType: &includeSubscriptions,
		},
	}

	response, err := client.API.Organizations().GetAsOrganizationsGetResponse(ctx, &c)
	if err != nil {
		log.Fatalf("API returned an error status: %s", tfe.SummarizeAPIErrors(err))
		return 1
	}

	// Serialize the response to JSON for display
	buffer, err := serialization.SerializeToJson(response)
	if err != nil {
		log.Fatalf("Error serializing response: %s", err)
		return 1
	}

	fmt.Println(string(buffer))
	return 0
}
