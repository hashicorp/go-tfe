package tfe

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/go-uuid"
)

func testClient(t *testing.T, fn ...func(*Config)) *Client {
	if v := os.Getenv("TFE_TOKEN"); v == "" {
		t.Fatal("TFE_TOKEN must be set")
	}

	config := DefaultConfig()
	config.Token = os.Getenv("TFE_TOKEN")

	client, err := NewClient(config)
	if err != nil {
		t.Fatal(err)
	}

	return client
}

func createOrganization(t *testing.T, client *Client) (*Organization, func()) {
	resp, err := client.CreateOrganization(&CreateOrganizationInput{
		Name:  randomString(t),
		Email: fmt.Sprintf("%s@tfe.local", randomString(t)),
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

func randomString(t *testing.T) string {
	v, err := uuid.GenerateUUID()
	if err != nil {
		t.Fatal(err)
	}
	return v
}
