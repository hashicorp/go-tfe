// Copyright IBM Corp. 2018, 2026
// SPDX-License-Identifier: MPL-2.0

package testserver_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	tfe "github.com/hashicorp/go-tfe"
	"github.com/hashicorp/go-tfe/testserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServer_OrganizationsAndWorkspaces(t *testing.T) {
	t.Parallel()

	srv := testserver.New()
	t.Cleanup(srv.Close)

	client, err := tfe.NewClient(srv.ClientConfig())
	require.NoError(t, err)

	assert.Equal(t, "2.0", client.RemoteAPIVersion())
	assert.True(t, client.IsCloud())

	ctx := context.Background()

	org, err := client.Organizations.Create(ctx, tfe.OrganizationCreateOptions{
		Name:                  tfe.String("acme"),
		Email:                 tfe.String("platform@acme.local"),
		CostEstimationEnabled: tfe.Bool(true),
	})
	require.NoError(t, err)
	assert.Equal(t, "acme", org.Name)

	orgList, err := client.Organizations.List(ctx, nil)
	require.NoError(t, err)
	require.Len(t, orgList.Items, 1)
	assert.Equal(t, "acme", orgList.Items[0].Name)

	workspace, err := client.Workspaces.Create(ctx, org.Name, tfe.WorkspaceCreateOptions{
		Name:             tfe.String("payments-prod"),
		Description:      tfe.String("payments production workspace"),
		AutoApply:        tfe.Bool(true),
		TerraformVersion: tfe.String("1.11.2"),
	})
	require.NoError(t, err)
	assert.NotEmpty(t, workspace.ID)

	workspaceByName, err := client.Workspaces.Read(ctx, org.Name, workspace.Name)
	require.NoError(t, err)
	assert.Equal(t, workspace.ID, workspaceByName.ID)

	workspaceByID, err := client.Workspaces.ReadByID(ctx, workspace.ID)
	require.NoError(t, err)
	assert.Equal(t, "acme", workspaceByID.Organization.Name)

	workspaceList, err := client.Workspaces.List(ctx, org.Name, &tfe.WorkspaceListOptions{Search: "payments"})
	require.NoError(t, err)
	require.Len(t, workspaceList.Items, 1)

	updatedWorkspace, err := client.Workspaces.UpdateByID(ctx, workspace.ID, tfe.WorkspaceUpdateOptions{
		Name:        tfe.String("payments-primary"),
		Description: tfe.String("primary workspace"),
		AutoApply:   tfe.Bool(false),
	})
	require.NoError(t, err)
	assert.Equal(t, "payments-primary", updatedWorkspace.Name)
	assert.Equal(t, "primary workspace", updatedWorkspace.Description)
	assert.False(t, updatedWorkspace.AutoApply)

	updatedOrg, err := client.Organizations.Update(ctx, org.Name, tfe.OrganizationUpdateOptions{
		Name: tfe.String("acme-platform"),
	})
	require.NoError(t, err)
	assert.Equal(t, "acme-platform", updatedOrg.Name)

	workspaceAfterOrgRename, err := client.Workspaces.ReadByID(ctx, workspace.ID)
	require.NoError(t, err)
	assert.Equal(t, "acme-platform", workspaceAfterOrgRename.Organization.Name)

	err = client.Workspaces.DeleteByID(ctx, workspace.ID)
	require.NoError(t, err)

	err = client.Organizations.Delete(ctx, "acme-platform")
	require.NoError(t, err)

	_, err = client.Organizations.Read(ctx, "acme-platform")
	require.Error(t, err)
	assert.True(t, errors.Is(err, tfe.ErrResourceNotFound))
}

func TestServer_DiscoveryEndpoint(t *testing.T) {
	t.Parallel()

	srv := testserver.New()
	t.Cleanup(srv.Close)

	resp, err := srv.Client().Get(srv.URL())
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = resp.Body.Close()
	})

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var payload struct {
		Name               string   `json:"name"`
		Token              string   `json:"token"`
		SupportedEndpoints []string `json:"supported_endpoints"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&payload))

	assert.Equal(t, "go-tfe steel-thread test server", payload.Name)
	assert.Equal(t, srv.Token(), payload.Token)
	assert.Contains(t, payload.SupportedEndpoints, "GET /api/v2/ping")
	assert.Contains(t, payload.SupportedEndpoints, "GET|POST /api/v2/organizations")
}
