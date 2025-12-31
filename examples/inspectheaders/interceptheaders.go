package inspectheaders

import (
	"context"
	"log"
	"os"

	"github.com/hashicorp/cli"
	"github.com/hashicorp/go-tfe"
	"github.com/hashicorp/go-tfe/api/account"

	abstractions "github.com/microsoft/kiota-abstractions-go"
	khttp "github.com/microsoft/kiota-http-go"
)

type inspectHeadersCommand struct{}

var _ cli.Command = inspectHeadersCommand{}

func InspectHeadersCommandFactory() (cli.Command, error) {
	return &inspectHeadersCommand{}, nil
}

func (inspectHeadersCommand) Help() string {
	return "Intercept response headers from the TFE API example"
}

func (inspectHeadersCommand) Synopsis() string {
	return "Intercept response headers from the TFE API example"
}

func (inspectHeadersCommand) Run(args []string) int {
	client, err := tfe.NewClient(&tfe.Config{
		Token:   os.Getenv("TFE_TOKEN"),
		Address: os.Getenv("TFE_ADDRESS"),
	})

	if err != nil {
		log.Fatalf("Error creating TFE client: %s", err)
		return 1
	}

	ctx := context.Background()

	// 1. Create the HeadersInspectionRequestOption
	inspectionOptions := khttp.NewHeadersInspectionOptions()
	inspectionOptions.InspectResponseHeaders = true

	// 2. Create/add the option to the RequestInformation object for the request
	req := account.DetailsRequestBuilderGetRequestConfiguration{
		Options: []abstractions.RequestOption{inspectionOptions},
	}
	if err != nil {
		log.Fatalf("Error creating request: %s", err)
		return 1
	}

	// 3. Execute the request
	_, err = client.API.Account().Details().GetAsDetailsGetResponse(ctx, &req)
	if err != nil {
		log.Fatalf("Error getting account details: %s", err)
		return 1
	}

	// 4. Access the response headers from the HeadersInspectionRequestOption
	headers := inspectionOptions.GetResponseHeaders()
	for _, key := range headers.ListKeys() {
		log.Printf("%s: %v", key, headers.Get(key))
	}

	return 0
}
