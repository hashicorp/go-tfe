package tfe

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWorkspaces(t *testing.T) {
	client := testClient(t)

	org, orgCleanup := createOrganization(t, client)
	defer orgCleanup()

	ws1, ws1Cleanup := createWorkspace(t, client, org)
	defer ws1Cleanup()
	ws2, ws2Cleanup := createWorkspace(t, client, org)
	defer ws2Cleanup()

	// List the workspaces within the organization.
	workspaces, err := client.Workspaces(*org.Name)
	assert.Nil(t, err)

	expect := []*Workspace{ws1, ws2}

	// Sort to ensure we get a non-flaky comparison.
	sort.Stable(WorkspaceNameSort(expect))
	sort.Stable(WorkspaceNameSort(workspaces))

	assert.Equal(t, expect, workspaces)
}
