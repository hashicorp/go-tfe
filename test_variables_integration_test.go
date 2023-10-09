// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTestConfigVarsList(t *testing.T) {
	skipUnlessBeta(t)

	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	rmTest, registryModuleTestCleanup := createBranchBasedRegistryModule(t, client, orgTest)
	defer registryModuleTestCleanup()

	id := RegistryModuleID{
		Organization: orgTest.Name,
		Name:         rmTest.Name,
		Provider:     rmTest.Provider,
		Namespace:    rmTest.Namespace,
		RegistryName: rmTest.RegistryName,
	}

	tv1, tvCleanup1 := createTestVariable(t, client, rmTest)
	tv2, tvCleanup2 := createTestVariable(t, client, rmTest)

	defer tvCleanup1()
	defer tvCleanup2()

	t.Run("without list options", func(t *testing.T) {
		trl, err := client.TestVariables.List(ctx, id, nil)
		var found []string
		for _, r := range trl.Items {
			found = append(found, r.ID)
		}

		require.NoError(t, err)
		assert.Contains(t, found, tv1.ID)
		assert.Contains(t, found, tv2.ID)
		assert.Equal(t, 1, trl.CurrentPage)
		assert.Equal(t, 2, trl.TotalCount)
	})

	t.Run("empty list options", func(t *testing.T) {
		trl, err := client.TestVariables.List(ctx, id, &VariableListOptions{})
		var found []string
		for _, r := range trl.Items {
			found = append(found, r.ID)
		}

		require.NoError(t, err)
		assert.Contains(t, found, tv1.ID)
		assert.Contains(t, found, tv2.ID)
		assert.Equal(t, 1, trl.CurrentPage)
		assert.Equal(t, 2, trl.TotalCount)
	})

	t.Run("with page size", func(t *testing.T) {
		// Request a page number which is out of range. The result should
		// be successful, but return no results if the paging options are
		// properly passed along.
		tvl, err := client.TestVariables.List(ctx, id, &VariableListOptions{
			ListOptions: ListOptions{
				PageNumber: 999,
				PageSize:   100,
			},
		})

		require.NoError(t, err)
		assert.Empty(t, tvl.Items)
		assert.Equal(t, 999, tvl.CurrentPage)
		assert.Equal(t, 2, tvl.TotalCount)
	})
}
