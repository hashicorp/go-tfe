package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"

	tfe "github.com/hashicorp/go-tfe"
)

type WorkflowRunnerConfiguration map[string]string

// fetchOutputs reads the current state version for the specified workspace and returns the outputs
func fetchOutputs(ctx context.Context, client *tfe.Client, organization string, workspace string) ([]*tfe.StateVersionOutput, error) {
	ws, err := client.Workspaces.Read(ctx, organization, workspace)
	if err != nil {
		return nil, fmt.Errorf("failed reading workspace (%s/%s): %v", organization, workspace, err)
	}

	sv, err := client.StateVersions.ReadCurrentWithOptions(ctx, ws.ID, &tfe.StateVersionCurrentOptions{
		Include: []tfe.StateVersionIncludeOpt{tfe.SVoutputs},
	})
	if err != nil {
		return nil, fmt.Errorf("failed reading current state version (%s): %v", ws.ID, err)
	} else if len(sv.Outputs) == 0 {
		return nil, fmt.Errorf("state version (%s) has no available outputs", sv.ID)
	}

	return sv.Outputs, nil
}

// newRunnerConfiguration creates a workflow runner configuration
func newRunnerConfiguration(ctx context.Context, outputs []*tfe.StateVersionOutput) (WorkflowRunnerConfiguration, error) {
	config := make(WorkflowRunnerConfiguration)
	for _, output := range outputs {
		if output.Name == "tfe_token" || output.Name == "tfe_address" {
			if val, ok := output.Value.(string); ok {
				config[strings.ToUpper(output.Name)] = val
			}
		}
	}

	if _, ok := config["TFE_TOKEN"]; !ok {
		return nil, fmt.Errorf("tfe_token output variable is not set")
	}
	// temporary debugging lines to troubleshoot for an issue around tfe_token
	tk := config["TFE_TOKEN"]
	fmt.Println(len(tk))
	lastfive := tk[len(tk)-5:]
 	fmt.Println(lastfive)

	if _, ok := config["TFE_ADDRESS"]; !ok {
		return nil, fmt.Errorf("tfe_address output variable is not set")
	}

	return config, nil
}

// writeToEnv writes the WorkflowRunnerConfiguration to $GITHUB_ENV
func writeToEnv(ctx context.Context, config WorkflowRunnerConfiguration) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("unable to get the current home directory: %w", err)
	}

	name := filepath.Join(homeDir, ".env")
	f, err := os.OpenFile(name, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("unable to open the file: %w", err)
	}
	defer f.Close()

	for k := range config {
		envVar := fmt.Sprintf("export %s=%s\n", k, config[k])
		if _, err := f.WriteString(envVar); err != nil {
			return fmt.Errorf("unable to write to the file: %w", err)
		}
	}

	return nil
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	if len(os.Args) < 3 {
		log.Fatal("usage: <organization-name> <workspace-name>")
	}

	organization := os.Args[1]
	workspace := os.Args[2]

	client, err := tfe.NewClient(tfe.DefaultConfig())
	if err != nil {
		log.Fatalf("client initialization error: %v", err)
	}

	outputs, err := fetchOutputs(ctx, client, organization, workspace)
	if err != nil {
		log.Fatal(err)
	}

	config, err := newRunnerConfiguration(ctx, outputs)
	if err != nil {
		log.Fatal(err)
	}

	err = writeToEnv(ctx, config)
	if err != nil {
		log.Fatal(err)
	}
}
