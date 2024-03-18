// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVariableSetVariablesList(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	vsTest, vsTestCleanup := createVariableSet(t, client, orgTest, VariableSetCreateOptions{})
	defer vsTestCleanup()

	vTest1, vTestCleanup1 := createVariableSetVariable(t, client, vsTest, VariableSetVariableCreateOptions{
		Key:      String("vTest1"),
		Value:    String("vTest1"),
		Category: Category(CategoryTerraform),
	})
	defer vTestCleanup1()
	vTest2, vTestCleanup2 := createVariableSetVariable(t, client, vsTest, VariableSetVariableCreateOptions{
		Key:      String("vTest2"),
		Value:    String("vTest2"),
		Category: Category(CategoryTerraform),
	})
	defer vTestCleanup2()

	t.Run("without list options", func(t *testing.T) {
		vl, err := client.VariableSetVariables.List(ctx, vsTest.ID, nil)
		require.NoError(t, err)
		require.NotEmpty(t, vl.Items)
		assert.Contains(t, vl.Items, vTest1)
		assert.Contains(t, vl.Items, vTest2)

		t.Run("variable set relationship is deserialized", func(t *testing.T) {
			require.NotNil(t, vl.Items[0].VariableSet)
			assert.Equal(t, vsTest.ID, vl.Items[0].VariableSet.ID)
		})
	})

	t.Run("with list options", func(t *testing.T) {
		t.Skip("paging not supported yet in API")
		// Request a page number which is out of range. The result should
		// be successful, but return no results if the paging options are
		// properly passed along.
		vl, err := client.VariableSetVariables.List(ctx, vsTest.ID, &VariableSetVariableListOptions{
			ListOptions: ListOptions{
				PageNumber: 999,
				PageSize:   100,
			},
		})
		require.NoError(t, err)
		assert.Empty(t, vl.Items)
		assert.Equal(t, 999, vl.CurrentPage)
		assert.Equal(t, 2, vl.TotalCount)
	})

	t.Run("when variable set ID is invalid ID", func(t *testing.T) {
		vl, err := client.VariableSetVariables.List(ctx, badIdentifier, nil)
		assert.Nil(t, vl)
		assert.EqualError(t, err, ErrInvalidVariableSetID.Error())
	})
}

func TestVariableSetVariablesCreate(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	vsTest, vsTestCleanup := createVariableSet(t, client, orgTest, VariableSetCreateOptions{})
	defer vsTestCleanup()

	t.Run("with valid options", func(t *testing.T) {
		options := VariableSetVariableCreateOptions{
			Key:         String(randomString(t)),
			Value:       String(randomString(t)),
			Category:    Category(CategoryTerraform),
			Description: String(randomString(t)),
			HCL:         Bool(false),
			Sensitive:   Bool(false),
		}

		v, err := client.VariableSetVariables.Create(ctx, vsTest.ID, &options)
		require.NoError(t, err)

		assert.NotEmpty(t, v.ID)
		assert.Equal(t, *options.Key, v.Key)
		assert.Equal(t, *options.Value, v.Value)
		assert.Equal(t, *options.Description, v.Description)
		assert.Equal(t, *options.Category, v.Category)
		assert.NotEmpty(t, v.VersionID)
	})

	t.Run("when options has an empty string value", func(t *testing.T) {
		options := VariableSetVariableCreateOptions{
			Key:         String(randomString(t)),
			Value:       String(""),
			Description: String(randomString(t)),
			Category:    Category(CategoryTerraform),
		}

		v, err := client.VariableSetVariables.Create(ctx, vsTest.ID, &options)
		require.NoError(t, err)

		assert.NotEmpty(t, v.ID)
		assert.Equal(t, *options.Key, v.Key)
		assert.Equal(t, *options.Value, v.Value)
		assert.Equal(t, *options.Description, v.Description)
		assert.Equal(t, *options.Category, v.Category)
		assert.NotEmpty(t, v.VersionID)
	})

	t.Run("when options has an empty string description", func(t *testing.T) {
		options := VariableSetVariableCreateOptions{
			Key:         String(randomString(t)),
			Value:       String(randomString(t)),
			Description: String(""),
			Category:    Category(CategoryTerraform),
		}

		v, err := client.VariableSetVariables.Create(ctx, vsTest.ID, &options)
		require.NoError(t, err)

		assert.NotEmpty(t, v.ID)
		assert.Equal(t, *options.Key, v.Key)
		assert.Equal(t, *options.Value, v.Value)
		assert.Equal(t, *options.Description, v.Description)
		assert.Equal(t, *options.Category, v.Category)
		assert.NotEmpty(t, v.VersionID)
	})

	t.Run("when options has a too-long description", func(t *testing.T) {
		options := VariableSetVariableCreateOptions{
			Key:         String(randomString(t)),
			Value:       String(randomString(t)),
			Description: String("tortor aliquam nulla redacted cras fermentum odio eu feugiat pretium nibh ipsum consequat nisl vel pretium lectus quam id leo in vitae turpis massa sed elementum tempus egestas sed sed risus pretium quam vulputate dignissim suspendisse in est ante in nibh mauris cursus mattis molestie a iaculis at erat pellentesque adipiscing commodo elit at imperdiet dui accumsan sit amet nulla redacted morbi tempus iaculis urna id volutpat lacus laoreet non curabitur gravida arcu ac tortor dignissim convallis aenean et tortor"),
			Category:    Category(CategoryTerraform),
		}

		_, err := client.VariableSetVariables.Create(ctx, vsTest.ID, &options)
		assert.Error(t, err)
	})

	t.Run("when options is missing value", func(t *testing.T) {
		options := VariableSetVariableCreateOptions{
			Key:      String(randomString(t)),
			Category: Category(CategoryTerraform),
		}

		v, err := client.VariableSetVariables.Create(ctx, vsTest.ID, &options)
		require.NoError(t, err)

		assert.NotEmpty(t, v.ID)
		assert.Equal(t, *options.Key, v.Key)
		assert.Equal(t, "", v.Value)
		assert.Equal(t, *options.Category, v.Category)
		assert.NotEmpty(t, v.VersionID)
	})

	t.Run("when options is missing key", func(t *testing.T) {
		options := VariableSetVariableCreateOptions{
			Value:    String(randomString(t)),
			Category: Category(CategoryTerraform),
		}

		_, err := client.VariableSetVariables.Create(ctx, vsTest.ID, &options)
		assert.EqualError(t, err, ErrRequiredKey.Error())
	})

	t.Run("when options has an empty key", func(t *testing.T) {
		options := VariableSetVariableCreateOptions{
			Key:      String(""),
			Value:    String(randomString(t)),
			Category: Category(CategoryTerraform),
		}

		_, err := client.VariableSetVariables.Create(ctx, vsTest.ID, &options)
		assert.EqualError(t, err, ErrRequiredKey.Error())
	})

	t.Run("when options is missing category", func(t *testing.T) {
		options := VariableSetVariableCreateOptions{
			Key:   String(randomString(t)),
			Value: String(randomString(t)),
		}

		_, err := client.VariableSetVariables.Create(ctx, vsTest.ID, &options)
		assert.EqualError(t, err, ErrRequiredCategory.Error())
	})

	t.Run("when workspace ID is invalid", func(t *testing.T) {
		options := VariableSetVariableCreateOptions{
			Key:      String(randomString(t)),
			Value:    String(randomString(t)),
			Category: Category(CategoryTerraform),
		}

		_, err := client.VariableSetVariables.Create(ctx, badIdentifier, &options)
		assert.EqualError(t, err, ErrInvalidVariableSetID.Error())
	})
}

func TestVariableSetVariablesRead(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	vsTest, vsTestCleanup := createVariableSet(t, client, orgTest, VariableSetCreateOptions{})
	defer vsTestCleanup()

	vTest, vTestCleanup := createVariableSetVariable(t, client, vsTest, VariableSetVariableCreateOptions{})
	defer vTestCleanup()

	t.Run("when the variable exists", func(t *testing.T) {
		v, err := client.VariableSetVariables.Read(ctx, vsTest.ID, vTest.ID)
		require.NoError(t, err)
		assert.Equal(t, vTest.ID, v.ID)
		assert.Equal(t, vTest.Category, v.Category)
		assert.Equal(t, vTest.HCL, v.HCL)
		assert.Equal(t, vTest.Key, v.Key)
		assert.Equal(t, vTest.Sensitive, v.Sensitive)
		assert.Equal(t, vTest.Value, v.Value)
		assert.Equal(t, vTest.VersionID, v.VersionID)
	})

	t.Run("when the variable does not exist", func(t *testing.T) {
		v, err := client.VariableSetVariables.Read(ctx, vsTest.ID, "nonexisting")
		assert.Nil(t, v)
		assert.ErrorIs(t, ErrResourceNotFound, err)
	})

	t.Run("without a valid variable set ID", func(t *testing.T) {
		v, err := client.VariableSetVariables.Read(ctx, badIdentifier, vTest.ID)
		assert.Nil(t, v)
		assert.EqualError(t, err, ErrInvalidVariableSetID.Error())
	})

	t.Run("without a valid variable ID", func(t *testing.T) {
		v, err := client.VariableSetVariables.Read(ctx, vsTest.ID, badIdentifier)
		assert.Nil(t, v)
		assert.EqualError(t, err, ErrInvalidVariableID.Error())
	})
}

func TestVariableSetVariablesUpdate(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	vsTest, vsTestCleanup := createVariableSet(t, client, nil, VariableSetCreateOptions{})
	defer vsTestCleanup()

	vTest, vTestCleanup := createVariableSetVariable(t, client, vsTest, VariableSetVariableCreateOptions{})
	defer vTestCleanup()

	t.Run("with valid options", func(t *testing.T) {
		options := VariableSetVariableUpdateOptions{
			Key:   String("newname"),
			Value: String("newvalue"),
			HCL:   Bool(true),
		}

		v, err := client.VariableSetVariables.Update(ctx, vsTest.ID, vTest.ID, &options)
		require.NoError(t, err)

		assert.Equal(t, *options.Key, v.Key)
		assert.Equal(t, *options.HCL, v.HCL)
		assert.Equal(t, *options.Value, v.Value)
		assert.NotEqual(t, vTest.VersionID, v.VersionID)
	})

	t.Run("when updating a subset of values", func(t *testing.T) {
		options := VariableSetVariableUpdateOptions{
			Key: String("someothername"),
			HCL: Bool(false),
		}

		v, err := client.VariableSetVariables.Update(ctx, vsTest.ID, vTest.ID, &options)
		require.NoError(t, err)

		assert.Equal(t, *options.Key, v.Key)
		assert.Equal(t, *options.HCL, v.HCL)
		assert.NotEqual(t, vTest.VersionID, v.VersionID)
	})

	t.Run("with sensitive set", func(t *testing.T) {
		options := VariableSetVariableUpdateOptions{
			Sensitive: Bool(true),
		}

		v, err := client.VariableSetVariables.Update(ctx, vsTest.ID, vTest.ID, &options)
		require.NoError(t, err)

		assert.Equal(t, *options.Sensitive, v.Sensitive)
		assert.Empty(t, v.Value) // Because its now sensitive
		assert.NotEqual(t, vTest.VersionID, v.VersionID)
	})

	t.Run("without any changes", func(t *testing.T) {
		vTest, vTestCleanup := createVariableSetVariable(t, client, vsTest, VariableSetVariableCreateOptions{})
		defer vTestCleanup()

		options := VariableSetVariableUpdateOptions{
			Key:         String(vTest.Key),
			Value:       String(vTest.Value),
			Description: String(vTest.Description),
			Sensitive:   Bool(vTest.Sensitive),
			HCL:         Bool(vTest.HCL),
		}

		v, err := client.VariableSetVariables.Update(ctx, vsTest.ID, vTest.ID, &options)
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
		_, err := client.VariableSetVariables.Update(ctx, badIdentifier, vTest.ID, nil)
		assert.EqualError(t, err, ErrInvalidVariableSetID.Error())
	})

	t.Run("with invalid variable ID", func(t *testing.T) {
		_, err := client.VariableSetVariables.Update(ctx, vsTest.ID, badIdentifier, nil)
		assert.EqualError(t, err, ErrInvalidVariableID.Error())
	})
}

func TestVariableSetVariablesDelete(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	vsTest, vsTestCleanup := createVariableSet(t, client, nil, VariableSetCreateOptions{})
	defer vsTestCleanup()

	vTest, _ := createVariableSetVariable(t, client, vsTest, VariableSetVariableCreateOptions{})

	t.Run("with valid options", func(t *testing.T) {
		err := client.VariableSetVariables.Delete(ctx, vsTest.ID, vTest.ID)
		require.NoError(t, err)
	})

	t.Run("with non existing variable ID", func(t *testing.T) {
		err := client.VariableSetVariables.Delete(ctx, vsTest.ID, "nonexisting")
		assert.EqualError(t, err, ErrResourceNotFound.Error())
	})

	t.Run("with invalid workspace ID", func(t *testing.T) {
		err := client.VariableSetVariables.Delete(ctx, badIdentifier, vTest.ID)
		assert.EqualError(t, err, ErrInvalidVariableSetID.Error())
	})

	t.Run("with invalid variable ID", func(t *testing.T) {
		err := client.VariableSetVariables.Delete(ctx, vsTest.ID, badIdentifier)
		assert.EqualError(t, err, ErrInvalidVariableID.Error())
	})
}
