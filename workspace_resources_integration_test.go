// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestWorkspaceResourcesList(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	wTest, wTestCleanup := createWorkspace(t, client, orgTest)
	t.Cleanup(wTestCleanup)

	svTest, svTestCleanup := createStateVersion(t, client, 0, wTest)
	t.Cleanup(svTestCleanup)

	// give TFC some time to process the statefile and extract the outputs.
	waitForSVOutputs(t, client, svTest.ID)

	t.Run("without list options", func(t *testing.T) {
		rs, err := client.WorkspaceResources.List(ctx, wTest.ID, nil)
		require.NoError(t, err)
		assert.Equal(t, 1, len(rs.Items))
		assert.Equal(t, 1, rs.CurrentPage)
		assert.Equal(t, 1, rs.TotalCount)

		assert.Equal(t, "null_resource.test", rs.Items[0].Address)
		assert.Equal(t, "test", rs.Items[0].Name)
		assert.Equal(t, "root", rs.Items[0].Module)
		assert.Equal(t, "null", rs.Items[0].Provider)
	})
	t.Run("with list options", func(t *testing.T) {
		rs, err := client.WorkspaceResources.List(ctx, wTest.ID, &WorkspaceResourceListOptions{
			ListOptions: ListOptions{
				PageNumber: 999,
				PageSize:   100,
			},
		})
		require.NoError(t, err)
		assert.Empty(t, rs.Items)
		assert.Equal(t, 999, rs.CurrentPage)
		assert.Equal(t, 1, rs.TotalCount)
	})
}
