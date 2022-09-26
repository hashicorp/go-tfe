package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	tfe "github.com/hashicorp/go-tfe"
)

// instanceAddr contains the state target that will be forcibly replaced every run
const instanceAddr = "module.tflocal.module.tfbox.aws_instance.tfbox"

// tokenAddr contains the target token that will be forcibly replaced every run
const tokenAddr = "module.tflocal.var.tflocal_cloud_admin_token"

var workspace string
var organization string
var isDestroy bool

func init() {
	flag.StringVar(&organization, "o", "hashicorp-v2", "the TFC organization that owns the specified workspace.")
	flag.StringVar(&workspace, "w", "tflocal-go-tfe", "the TFC workspace to create a run in.")
	flag.BoolVar(&isDestroy, "d", false, "trigger a destroy run.")
	flag.Parse()
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	config := &tfe.Config{Token: os.Getenv("TFE_TOKEN")}

	client, err := tfe.NewClient(config)
	if err != nil {
		log.Fatalf("client initialization error: %v", err)
	}

	var runID string
	if runID, err = createRun(ctx, client); err != nil {
		log.Fatal(err)
	}

	if err := waitForRun(ctx, client, runID); err != nil {
		log.Fatal(err)
	}
}

func createRun(ctx context.Context, client *tfe.Client) (string, error) {
	wk, err := client.Workspaces.Read(ctx, organization, workspace)
	if err != nil {
		return "", fmt.Errorf("failed to read workspace: %w", err)
	}

	opts := tfe.RunCreateOptions{
		IsDestroy: tfe.Bool(isDestroy),
		Message:   tfe.String("Queued nightly from GH Actions via go-tfe"),
		Workspace: wk,
		AutoApply: tfe.Bool(true),
	}

	if !isDestroy {
		opts.ReplaceAddrs = []string{instanceAddr, tokenAddr}
	}

	run, err := client.Runs.Create(ctx, opts)
	if err != nil {
		return "", fmt.Errorf("failed to trigger run: %w", err)
	}

	log.Printf("Created run: %s\n", run.ID)
	return run.ID, nil
}

func waitForRun(ctx context.Context, client *tfe.Client, runID string) error {
	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()

	start := time.Now()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case tick := <-ticker.C:
			run, err := client.Runs.Read(ctx, runID)
			if err != nil {
				return err
			}

			duration := tick.Sub(start)

			if run.Status == tfe.RunErrored || run.Status == tfe.RunCanceled {
				return fmt.Errorf("run %s has errored or been canceled", runID)
			}

			if run.Status == tfe.RunApplied {
				log.Printf("Run completed (%s elapsed)", duration.Round(time.Second))
				return nil
			}

			log.Printf("Waiting run to complete (%s elapsed)", duration.Round(time.Second))
		}
	}
}
