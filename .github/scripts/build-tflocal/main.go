package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"

	tfe "github.com/hashicorp/go-tfe"
)

// instanceAddr contains the state target that will be forcibly replaced every run
const instanceAddr = "module.tflocal.module.tfbox.aws_instance.tfbox"

// tokenAddr contains the target token that will be forcibly replaced every run
const tokenAddr = "module.tflocal.var.tflocal_cloud_admin_token"

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	if len(os.Args) < 3 {
		log.Fatal("usage: <organization-name> <workspace-name>")
	}
	organizationName := os.Args[1]
	workspaceName := os.Args[2]

	if err := triggerRun(ctx, organizationName, workspaceName); err != nil {
		log.Fatal(err)
	}
}

func triggerRun(ctx context.Context, organizationName, workspaceName string) error {
	config := &tfe.Config{Token: os.Getenv("TFE_TOKEN")}

	client, err := tfe.NewClient(config)
	if err != nil {
		return fmt.Errorf("client initialization error: %w", err)
	}

	wk, err := client.Workspaces.Read(ctx, organizationName, workspaceName)
	if err != nil {
		return fmt.Errorf("failed to read workspace: %w", err)
	}

	run, err := client.Runs.Create(ctx, tfe.RunCreateOptions{
		IsDestroy:    tfe.Bool(false),
		Message:      tfe.String("Queued nightly from tflocal-cloud GH Actions via go-tfe"),
		Workspace:    wk,
		ReplaceAddrs: []string{instanceAddr, tokenAddr},
	})
	if err != nil {
		return fmt.Errorf("failed to trigger run: %w", err)
	}

	fmt.Println("Created run: " + run.ID)
	return nil
}
