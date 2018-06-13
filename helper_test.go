package tfe

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/go-uuid"
)

const badIdentifier = "! / nope"

func testClient(t *testing.T, fn ...func(*Config)) *Client {
	config := DefaultConfig()

	for _, f := range fn {
		f(config)
	}

	if config.Token == "" {
		config.Token = os.Getenv("TFE_TOKEN")
		if config.Token == "" {
			t.Fatal("TFE_TOKEN must be set")
		}
	}

	if v := os.Getenv("TFE_ADDRESS"); v != "" {
		config.Address = v
	}

	client, err := NewClient(config)
	if err != nil {
		t.Fatal(err)
	}

	return client
}

func createOrganization(t *testing.T, client *Client) (*Organization, func()) {
	org, err := client.Organizations.Create(OrganizationCreateOptions{
		Name:  String(randomString(t)),
		Email: String(fmt.Sprintf("%s@tfe.local", randomString(t))),
	})
	if err != nil {
		t.Fatal(err)
	}

	return org, func() {
		if err := client.Organizations.Delete(org.Name); err != nil {
			t.Errorf("Error destroying organization! WARNING: Dangling resources\n"+
				"may exist! The full error is shown below.\n\n"+
				"Organization: %s\nError: %s", org.Name, err)
		}
	}
}

func createWorkspace(t *testing.T, client *Client, org *Organization) (*Workspace, func()) {
	var orgCleanup func()

	if org == nil {
		org, orgCleanup = createOrganization(t, client)
	}

	w, err := client.Workspaces.Create(org.Name, WorkspaceCreateOptions{
		Name: String(randomString(t)),
	})
	if err != nil {
		t.Fatal(err)
	}

	return w, func() {
		if err := client.Workspaces.Delete(org.Name, w.Name); err != nil {
			t.Errorf("Error destroying workspace! WARNING: Dangling resources\n"+
				"may exist! The full error is shown below.\n\n"+
				"Workspace: %s\nError: %s", w.Name, err)
		}

		if orgCleanup != nil {
			orgCleanup()
		}
	}
}

func createConfigurationVersion(t *testing.T, client *Client, w *Workspace) (*ConfigurationVersion, func()) {
	var wCleanup func()

	if w == nil {
		w, wCleanup = createWorkspace(t, client, nil)
	}

	cv, err := client.ConfigurationVersions.Create(
		w.ID,
		ConfigurationVersionCreateOptions{},
	)
	if err != nil {
		t.Fatal(err)
	}

	return cv, func() {
		if wCleanup != nil {
			wCleanup()
		}
	}
}

func createUploadedConfigurationVersion(t *testing.T, client *Client, w *Workspace) (*ConfigurationVersion, func()) {
	cv, cvCleanup := createConfigurationVersion(t, client, w)

	fh, err := os.Open("test-fixtures/configuration-version.tar.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer fh.Close()

	if err := client.upload(cv.UploadURL, fh); err != nil {
		t.Fatal(err)
	}

	// This is a bit nasty, but if you try to use the configuration version before
	// its fully processed server side, you will get an error when trying to use it.
	time.Sleep(3 * time.Second)

	return cv, cvCleanup
}

func createRun(t *testing.T, client *Client, w *Workspace) (*Run, func()) {
	var wCleanup func()

	if w == nil {
		w, wCleanup = createWorkspace(t, client, nil)
	}

	cv, cvCleanup := createUploadedConfigurationVersion(t, client, w)

	r, err := client.Runs.Create(RunCreateOptions{
		ConfigurationVersion: cv,
		Workspace:            w,
	})
	if err != nil {
		t.Fatal(err)
	}

	return r, func() {
		if wCleanup != nil {
			wCleanup()
		} else {
			cvCleanup()
		}
	}
}

func randomString(t *testing.T) string {
	v, err := uuid.GenerateUUID()
	if err != nil {
		t.Fatal(err)
	}
	return v
}
