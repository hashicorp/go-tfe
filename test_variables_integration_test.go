// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTestVariablesList(t *testing.T) {
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

func TestTestVariablesCreate(t *testing.T) {
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

	t.Run("with valid options", func(t *testing.T) {
		options := VariableCreateOptions{
			Key:         String(randomKeyValue(t)),
			Value:       String(randomStringWithoutSpecialChar(t)),
			Category:    Category(CategoryEnv),
			Description: String("testing"),
		}

		v, err := client.TestVariables.Create(ctx, id, options)
		require.NoError(t, err)

		assert.NotEmpty(t, v.ID)
		assert.Equal(t, *options.Key, v.Key)
		assert.Equal(t, *options.Value, v.Value)
		assert.Equal(t, *options.Description, v.Description)
		assert.Equal(t, *options.Category, v.Category)
		assert.NotEmpty(t, v.VersionID)
	})

	t.Run("when options has an empty string value", func(t *testing.T) {
		options := VariableCreateOptions{
			Key:         String(randomKeyValue(t)),
			Value:       String(""),
			Description: String("testing"),
			Category:    Category(CategoryEnv),
		}

		v, err := client.TestVariables.Create(ctx, id, options)
		require.NoError(t, err)

		assert.NotEmpty(t, v.ID)
		assert.Equal(t, *options.Key, v.Key)
		assert.Equal(t, *options.Value, v.Value)
		assert.Equal(t, *options.Description, v.Description)
		assert.Equal(t, *options.Category, v.Category)
		assert.NotEmpty(t, v.VersionID)
	})

	t.Run("when options has an empty string description", func(t *testing.T) {
		options := VariableCreateOptions{
			Key:         String(randomKeyValue(t)),
			Value:       String(randomStringWithoutSpecialChar(t)),
			Description: String(""),
			Category:    Category(CategoryEnv),
		}

		v, err := client.TestVariables.Create(ctx, id, options)
		require.NoError(t, err)

		assert.NotEmpty(t, v.ID)
		assert.Equal(t, *options.Key, v.Key)
		assert.Equal(t, *options.Value, v.Value)
		assert.Equal(t, *options.Description, v.Description)
		assert.Equal(t, *options.Category, v.Category)
		assert.NotEmpty(t, v.VersionID)
	})

	t.Run("when options has a too-long description", func(t *testing.T) {
		options := VariableCreateOptions{
			Key:         String(randomKeyValue(t)),
			Value:       String(randomStringWithoutSpecialChar(t)),
			Description: String("tortor aliquam nulla go lint is fussy about spelling cras fermentum odio eu feugiat pretium nibh ipsum consequat nisl vel pretium lectus quam id leo in vitae turpis massa sed elementum tempus egestas sed sed risus pretium quam vulputate dignissim suspendisse in est ante in nibh mauris cursus mattis molestie a iaculis at erat pellentesque adipiscing commodo elit at imperdiet dui accumsan sit amet nulla redacted morbi tempus iaculis urna id volutpat lacus laoreet non curabitur gravida arcu ac tortor dignissim convallis aenean et tortor"),
			Category:    Category(CategoryEnv),
		}

		_, err := client.TestVariables.Create(ctx, id, options)
		assert.Error(t, err)
	})

	t.Run("when options is missing value", func(t *testing.T) {
		options := VariableCreateOptions{
			Key:      String(randomKeyValue(t)),
			Category: Category(CategoryEnv),
		}

		v, err := client.TestVariables.Create(ctx, id, options)
		require.NoError(t, err)

		assert.NotEmpty(t, v.ID)
		assert.Equal(t, *options.Key, v.Key)
		assert.Equal(t, "", v.Value)
		assert.Equal(t, *options.Category, v.Category)
		assert.NotEmpty(t, v.VersionID)
	})

	t.Run("when options is missing key", func(t *testing.T) {
		options := VariableCreateOptions{
			Value:    String(randomStringWithoutSpecialChar(t)),
			Category: Category(CategoryEnv),
		}

		_, err := client.TestVariables.Create(ctx, id, options)
		assert.Equal(t, err, ErrRequiredKey)
	})

	t.Run("when options has an empty key", func(t *testing.T) {
		options := VariableCreateOptions{
			Key:      String(""),
			Value:    String(randomStringWithoutSpecialChar(t)),
			Category: Category(CategoryEnv),
		}

		_, err := client.TestVariables.Create(ctx, id, options)
		assert.Equal(t, err, ErrRequiredKey)
	})

	t.Run("when options is missing category", func(t *testing.T) {
		options := VariableCreateOptions{
			Key:   String(randomKeyValue(t)),
			Value: String(randomStringWithoutSpecialChar(t)),
		}

		_, err := client.TestVariables.Create(ctx, id, options)
		assert.Equal(t, err, ErrRequiredCategory)
	})
}

func TestTestVariablesUpdate(t *testing.T) {
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

	vTest, tvCleanup1 := createTestVariable(t, client, rmTest)

	defer tvCleanup1()

	t.Run("with valid options", func(t *testing.T) {
		options := VariableUpdateOptions{
			Key:   String("newname"),
			Value: String("newvalue"),
			HCL:   Bool(true),
		}

		v, err := client.TestVariables.Update(ctx, id, vTest.ID, options)
		require.NoError(t, err)

		assert.Equal(t, *options.Key, v.Key)
		assert.Equal(t, *options.HCL, v.HCL)
		assert.Equal(t, *options.Value, v.Value)
		assert.NotEqual(t, vTest.VersionID, v.VersionID)
	})

	t.Run("when updating a subset of values", func(t *testing.T) {
		options := VariableUpdateOptions{
			Key: String("someothername"),
			HCL: Bool(false),
		}

		v, err := client.TestVariables.Update(ctx, id, vTest.ID, options)
		require.NoError(t, err)

		assert.Equal(t, *options.Key, v.Key)
		assert.Equal(t, *options.HCL, v.HCL)
		assert.NotEqual(t, vTest.VersionID, v.VersionID)
	})

	t.Run("with sensitive set", func(t *testing.T) {
		options := VariableUpdateOptions{
			Sensitive: Bool(true),
		}

		v, err := client.TestVariables.Update(ctx, id, vTest.ID, options)
		require.NoError(t, err)

		assert.Equal(t, *options.Sensitive, v.Sensitive)
		assert.Empty(t, v.Value) // Because its now sensitive
		assert.NotEqual(t, vTest.VersionID, v.VersionID)
	})

	t.Run("without any changes", func(t *testing.T) {
		v, err := client.TestVariables.Update(ctx, id, vTest.ID, VariableUpdateOptions{})
		require.NoError(t, err)

		assert.Equal(t, vTest.ID, v.ID)
		assert.Equal(t, vTest.Key, v.Key)
		assert.Equal(t, vTest.Value, v.Value)
		assert.Equal(t, vTest.Description, v.Description)
		assert.Equal(t, vTest.Category, v.Category)
		assert.Equal(t, vTest.HCL, v.HCL)
		assert.Equal(t, vTest.Sensitive, v.Sensitive)
		assert.NotEqual(t, vTest.VersionID, v.VersionID)
	})

	t.Run("with invalid variable ID", func(t *testing.T) {
		_, err := client.TestVariables.Update(ctx, id, badIdentifier, VariableUpdateOptions{})
		assert.Equal(t, err, ErrInvalidVariableID)
	})
}

func TestTestVariablesDelete(t *testing.T) {
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

	vTest, _ := createTestVariable(t, client, rmTest)

	t.Run("with valid options", func(t *testing.T) {
		err := client.TestVariables.Delete(ctx, id, vTest.ID)
		require.NoError(t, err)
	})

	t.Run("with non existing variable ID", func(t *testing.T) {
		err := client.TestVariables.Delete(ctx, id, "nonexisting")
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("with invalid variable ID", func(t *testing.T) {
		err := client.TestVariables.Delete(ctx, id, badIdentifier)
		assert.Equal(t, err, ErrInvalidVariableID)
	})
}
