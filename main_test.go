package tfe

import (
	"context"
	"log"
	"os"
	"strings"
	"testing"
	"time"
)

const (
	ghaResourceOrganizationName = "gha-test-resources"
	ghaRunMutexWorkspaceName    = "gha-run-mutex"
)

var (
	ghaRunMutexWorkspaceID string
)

func mustHaveGHAResourceOrganization(client *Client) {
	_, err := client.Organizations.Create(context.Background(), OrganizationCreateOptions{
		Name:  String(ghaResourceOrganizationName),
		Email: String("support@hashicorp.com"),
	})

	if err != nil {
		if strings.Contains(err.Error(), "Name has already been taken") {
			log.Printf("[DEBUG] Organization %q already exists", ghaResourceOrganizationName)
		} else {
			log.Fatalf("Error creating organization: %s", err)
		}
	}
}

func mustHaveGHARunMutexWorkspace(client *Client) {
	ws, err := client.Workspaces.Create(context.Background(), ghaResourceOrganizationName, WorkspaceCreateOptions{
		Name: String(ghaRunMutexWorkspaceName),
	})

	if err != nil {
		if strings.Contains(err.Error(), "Name has already been taken") {
			log.Printf("[DEBUG] Workspace %q already exists", ghaResourceOrganizationName)
			ws, err = client.Workspaces.Read(context.Background(), ghaResourceOrganizationName, ghaRunMutexWorkspaceName)

			if err != nil {
				log.Fatalf("Error reading %q workspace: %s", ghaRunMutexWorkspaceName, err)
			}
		} else {
			log.Fatalf("Error creating %q workspace: %s", ghaRunMutexWorkspaceName, err)
		}
	}

	ghaRunMutexWorkspaceID = ws.ID
}

func TestMain(m *testing.M) {
	// There is an org and a workspace that are created in the setup
	// that will be used to synchronize access to terraform in the testing
	// environment due to a bug with how tflocal runs terraform without acess
	// to a nomad agent.

	client, err := NewClient(&Config{
		RetryServerErrors: true,
	})
	if err != nil {
		log.Fatalf("Error creating client in TestMain: %s", err)
	}

	mustHaveGHAResourceOrganization(client)
	mustHaveGHARunMutexWorkspace(client)

	os.Exit(m.Run())
}

func acquireRunMutex(t *testing.T, client *Client) {
	t.Helper()

	if strings.Contains(t.Name(), "/") {
		t.Fatalf("Do not call acquireRunMutex within a t.Run call- use it at the top level of the Test function.")
	}

	ctx, cancelFunc := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancelFunc()

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	result := make(chan error)

	go func() {
		for {
			select {
			case <-ctx.Done():
				// Timeout has been reached
				result <- ctx.Err()
				close(result)
				return
			case <-ticker.C:
				log.Printf("[DEBUG] Attempting to acquire run mutex for %q", t.Name())
				_, err := client.Workspaces.Lock(ctx, ghaRunMutexWorkspaceID, WorkspaceLockOptions{
					Reason: String(t.Name()),
				})
				if err == nil {
					log.Printf("[DEBUG] Run mutex acquired for %q", t.Name())
					close(result)
					return
				}
				log.Printf("[DEBUG] Error acquiring run mutex for %q: %s", t.Name(), err)
			}
		}
	}()

	err := <-result
	if err != nil {
		t.Fatalf("Error acquiring run mutex: %s", err)
	}

	t.Cleanup(func() {
		_, err := client.Workspaces.Unlock(context.Background(), ghaRunMutexWorkspaceID)
		if err != nil {
			t.Fatalf("Error releasing run mutex-- future tests will fail until the workspace is manually unlocked: %s", err)
		}
		log.Printf("[DEBUG] Run mutex released for %q", t.Name())
	})
}
