package workspaces

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/hashicorp/cli"
	"github.com/hashicorp/go-tfe"
	"github.com/hashicorp/go-tfe/api/organizations"
	"github.com/hashicorp/go-tfe/helpers"
	"github.com/microsoft/kiota-abstractions-go/serialization"
)

type workspacesCreateCommand struct{}

var _ cli.Command = workspacesCreateCommand{}

func WorkspacesCreateCommandFactory() (cli.Command, error) {
	return &workspacesCreateCommand{}, nil
}

func (workspacesCreateCommand) Help() string {
	return "Create a new workspace"
}

func (workspacesCreateCommand) Synopsis() string {
	return "Create a new workspace"
}

func (c workspacesCreateCommand) Run(args []string) int {
	client, err := tfe.NewClient(&tfe.Config{
		Token:   os.Getenv("TFE_TOKEN"),
		Address: os.Getenv("TFE_ADDRESS"),
	})

	if err != nil {
		log.Fatalf("Error creating TFE client: %s", err)
		return 1
	}

	passwordFlags := flag.NewFlagSet("create workspace", flag.ContinueOnError)
	workspaceName := passwordFlags.String("name", "", "Workspace name")
	organizationName := passwordFlags.String("organization-name", "", "Organization name")

	if err := passwordFlags.Parse(args); err != nil {
		log.Fatalf("Error parsing flags: %s", err)
		return 1
	}

	if workspaceName == nil || *workspaceName == "" || organizationName == nil || *organizationName == "" {
		fmt.Println("Both --name and --organization-name are required")
		passwordFlags.Usage()
		return 1
	}

	ctx := context.Background()

	// It can be helpful to use wrapper functions to construct models used as
	// request bodies because they require many local variables to build
	// api/v2/account/password
	workspacable := helpers.NewCreateWorkspaceBody(helpers.CreateWorkspaceParams{
		Name: *workspaceName,
	})
	pw := organizations.NewItemWorkspacesPostRequestBody()
	pw.SetData(workspacable)

	response, err := client.API.Organizations().ByOrganization_name(*organizationName).Workspaces().Post(ctx, pw, nil)
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
