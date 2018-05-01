package tfe

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/go-uuid"
)

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
	resp, err := client.CreateOrganization(&CreateOrganizationInput{
		Name:  String(randomString(t)),
		Email: String(fmt.Sprintf("%s@tfe.local", randomString(t))),
	})
	if err != nil {
		t.Fatal(err)
	}
	return resp.Organization, func() {
		client.DeleteOrganization(&DeleteOrganizationInput{
			Name: resp.Organization.Name,
		})
	}
}

func createWorkspace(t *testing.T, client *Client, org *Organization) (
	*Workspace, func()) {

	var orgCleanup func()

	if org == nil {
		org, orgCleanup = createOrganization(t, client)
	}

	resp, err := client.CreateWorkspace(&CreateWorkspaceInput{
		OrganizationName: org.Name,
		Name:             String(randomString(t)),
	})
	if err != nil {
		t.Fatal(err)
	}
	return resp.Workspace, func() {
		client.DeleteWorkspace(&DeleteWorkspaceInput{
			OrganizationName: org.Name,
			Name:             resp.Workspace.Name,
		})

		if orgCleanup != nil {
			orgCleanup()
		}
	}
}

func createConfigurationVersion(t *testing.T, client *Client,
	ws *Workspace) (*ConfigurationVersion, func()) {

	var wsCleanup func()

	if ws == nil {
		ws, wsCleanup = createWorkspace(t, client, nil)
	}

	resp, err := client.CreateConfigurationVersion(
		&CreateConfigurationVersionInput{
			WorkspaceID: ws.ID,
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	return resp.ConfigurationVersion, func() {
		if wsCleanup != nil {
			wsCleanup()
		}
	}
}

func createUploadedConfigurationVersion(t *testing.T, client *Client,
	ws *Workspace) (*ConfigurationVersion, func()) {

	cv, cleanup := createConfigurationVersion(t, client, ws)

	fh, err := os.Open("test-fixtures/configuration-version.tar.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer fh.Close()

	if _, err := client.UploadConfigurationVersion(
		&UploadConfigurationVersionInput{
			ConfigurationVersion: cv,
			Data:                 fh,
		},
	); err != nil {
		t.Fatal(err)
	}

	return cv, cleanup
}

func createRun(t *testing.T, client *Client, ws *Workspace) (*Run, func()) {
	cv, cvCleanup := createUploadedConfigurationVersion(t, client, ws)

	resp, err := client.CreateRun(&CreateRunInput{
		WorkspaceID:            ws.ID,
		ConfigurationVersionID: cv.ID,
	})
	if err != nil {
		t.Fatal(err)
	}

	return resp.Run, cvCleanup
}

func randomString(t *testing.T) string {
	v, err := uuid.GenerateUUID()
	if err != nil {
		t.Fatal(err)
	}
	return v
}
