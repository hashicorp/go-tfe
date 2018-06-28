package main

import (
	"log"

	tfe "github.com/hashicorp/go-tfe"
)

func main() {
	config := &tfe.Config{
		Token: "UXsybZKSz07IEw.tfev2.FajRykbzcnG9ESrhBjBMLNUSsPp69qLyzclIskE",
	}

	client, err := tfe.NewClient(config)
	if err != nil {
		log.Fatal(err)
	}

	// Create a new workspace
	w, err := client.Workspaces.Create("org-name", tfe.WorkspaceCreateOptions{
		Name: tfe.String("my-app-tst"),
	})
	if err != nil {
		log.Fatal(err)
	}

	// Update the workspace
	w, err = client.Workspaces.Update("org-name", w.Name, tfe.WorkspaceUpdateOptions{
		AutoApply:        tfe.Bool(false),
		TerraformVersion: tfe.String("0.11.1"),
		WorkingDirectory: tfe.String("my-app/infra"),
	})
	if err != nil {
		log.Fatal(err)
	}
}
