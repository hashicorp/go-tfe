package tfe

import (
	"testing"
)

func TestOrganizations(t *testing.T) {
	client := testClient(t)

	orgs, err := client.Organizations()
	if err != nil {
		t.Fatal(err)
	}

	if v := len(orgs); v != 1 {
		t.Fatalf("expect 1 org, got %d", v)
	}
}
