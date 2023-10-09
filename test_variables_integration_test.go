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

	t.Run("with page size", func(t *testing.T) {
		// Request a page number which is out of range. The result should
		// be successful, but return no results if the paging options are
		// properly passed along.
		trl, err := client.TestVariables.List(ctx, id, &VariableListOptions{
			ListOptions: ListOptions{
				PageNumber: 999,
				PageSize:   100,
			},
		})

		require.NoError(t, err)
		assert.Empty(t, trl.Items)
		assert.Equal(t, 999, trl.CurrentPage)
		assert.Equal(t, 2, trl.TotalCount)
	})
}
