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

	resp, err := client.CreateWorkspace(&CreateWorkspaceInput{
		Organization: org.Name,
		Name:         String(randomString(t)),
	})
	if err != nil {
		t.Fatal(err)
	}
	return resp.Workspace, func() {
		client.DeleteWorkspace(&DeleteWorkspaceInput{
			Organization: org.Name,
			Name:         resp.Workspace.Name,
		})
	}
}

func randomString(t *testing.T) string {
	v, err := uuid.GenerateUUID()
	if err != nil {
		t.Fatal(err)
	}
	return v
}
