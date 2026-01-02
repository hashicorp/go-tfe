package account

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/hashicorp/cli"
	"github.com/hashicorp/go-tfe"
	"github.com/microsoft/kiota-abstractions-go/serialization"
)

type accountDetailsCommand struct{}

var _ cli.Command = accountDetailsCommand{}

func AccountDetailsCommandFactory() (cli.Command, error) {
	return &accountDetailsCommand{}, nil
}

func (accountDetailsCommand) Help() string {
	return "Get/Update account details or change account password"
}

func (accountDetailsCommand) Synopsis() string {
	return "Manage account details and password"
}

func (accountDetailsCommand) Run(args []string) int {
	client, err := tfe.NewClient(&tfe.Config{
		Token:   os.Getenv("TFE_TOKEN"),
		Address: os.Getenv("TFE_ADDRESS"),
	})

	if err != nil {
		log.Fatalf("Error creating TFE client: %s", err)
		return 1
	}

	ctx := context.Background()

	// nil request configuration is common and indicates no query parameters,
	// headers, or special request options
	response, err := client.API.Account().Details().GetAsDetailsGetResponse(ctx, nil)

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
