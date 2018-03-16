package tfe

import (
	"os"
	"testing"

	"github.com/hashicorp/go-uuid"
)

// testClient wraps a client with some useful base functionality.
type testClient struct {
	// The initialized API client.
	client *Client

	// A randomly generated organization name.
	orgName string
}

func (c *testClient) cleanup() {
	//c.client.DeleteOrg(c.orgName)
}

func newTestClient(t *testing.T, fn ...func(*Config)) *testClient {
	if v := os.Getenv("TFE_TOKEN"); v == "" {
		t.Fatal("TFE_TOKEN must be set")
	}

	config := DefaultConfig()
	config.Token = os.Getenv("TFE_TOKEN")

	client, err := NewClient(config)
	if err != nil {
		t.Fatal(err)
	}

	orgName, err := uuid.GenerateUUID()
	if err != nil {
		t.Fatal(err)
	}

	return &testClient{
		client:  client,
		orgName: orgName,
	}
}
