package main

import (
	"log"
	"os"

	"github.com/hashicorp/cli"
	"github.com/hashicorp/go-tfe/examples/account"
	"github.com/hashicorp/go-tfe/examples/inspectheaders"
	"github.com/hashicorp/go-tfe/examples/organizations"
	"github.com/hashicorp/go-tfe/examples/workspaces"
)

func main() {
	c := cli.NewCLI("app", "1.0.0")
	c.Args = os.Args[1:]
	c.Commands = map[string]cli.CommandFactory{
		"api headers":        inspectheaders.InspectHeadersCommandFactory,
		"account details":    account.AccountDetailsCommandFactory,
		"account password":   account.AccountChangePasswordCommandFactory,
		"organizations list": organizations.OrganizationListCommandFactory,
		"workspaces create":  workspaces.WorkspacesCreateCommandFactory,
	}

	exitStatus, err := c.Run()
	if err != nil {
		log.Println(err)
	}

	os.Exit(exitStatus)
}
