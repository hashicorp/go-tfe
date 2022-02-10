//go:build integration
// +build integration

package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVariablesList(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	wTest, wTestCleanup := createWorkspace(t, client, orgTest)
	defer wTestCleanup()

	vTest1, vTestCleanup1 := createVariable(t, client, wTest)
	defer vTestCleanup1()
	vTest2, vTestCleanup2 := createVariable(t, client, wTest)
	defer vTestCleanup2()

	t.Run("without list options", func(t *testing.T) {
		vl, err := client.Variables.List(ctx, wTest.ID, nil)
		require.NoError(t, err)
		assert.Contains(t, vl.Items, vTest1)
		assert.Contains(t, vl.Items, vTest2)

		t.Skip("paging not supported yet in API")
		assert.Equal(t, 1, vl.CurrentPage)
		assert.Equal(t, 2, vl.TotalCount)
	})

	t.Run("with list options", func(t *testing.T) {
		t.Skip("paging not supported yet in API")
		// Request a page number which is out of range. The result should
		// be successful, but return no results if the paging options are
		// properly passed along.
		vl, err := client.Variables.List(ctx, wTest.ID, &VariableListOptions{
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

	t.Run("when workspace ID is invalid ID", func(t *testing.T) {
		vl, err := client.Variables.List(ctx, badIdentifier, nil)
		assert.Nil(t, vl)
		assert.EqualError(t, err, ErrInvalidWorkspaceID.Error())
	})
}

func TestVariablesCreate(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	wTest, wTestCleanup := createWorkspace(t, client, nil)
	defer wTestCleanup()

	t.Run("with valid options", func(t *testing.T) {
		options := VariableCreateOptions{
			Key:         String(randomString(t)),
			Value:       String(randomString(t)),
			Category:    Category(CategoryTerraform),
			Description: String(randomString(t)),
		}

		v, err := client.Variables.Create(ctx, wTest.ID, options)
		require.NoError(t, err)

		assert.NotEmpty(t, v.ID)
		assert.Equal(t, *options.Key, v.Key)
		assert.Equal(t, *options.Value, v.Value)
		assert.Equal(t, *options.Description, v.Description)
		assert.Equal(t, *options.Category, v.Category)
		// The workspace isn't returned correcly by the API.
		// assert.Equal(t, *options.Workspace, v.Workspace)
	})

	t.Run("when options has an empty string value", func(t *testing.T) {
		options := VariableCreateOptions{
			Key:         String(randomString(t)),
			Value:       String(""),
			Description: String(randomString(t)),
			Category:    Category(CategoryTerraform),
		}

		v, err := client.Variables.Create(ctx, wTest.ID, options)
		require.NoError(t, err)

		assert.NotEmpty(t, v.ID)
		assert.Equal(t, *options.Key, v.Key)
		assert.Equal(t, *options.Value, v.Value)
		assert.Equal(t, *options.Description, v.Description)
		assert.Equal(t, *options.Category, v.Category)
	})

	t.Run("when options has an empty string description", func(t *testing.T) {
		options := VariableCreateOptions{
			Key:         String(randomString(t)),
			Value:       String(randomString(t)),
			Description: String(""),
			Category:    Category(CategoryTerraform),
		}

		v, err := client.Variables.Create(ctx, wTest.ID, options)
		require.NoError(t, err)

		assert.NotEmpty(t, v.ID)
		assert.Equal(t, *options.Key, v.Key)
		assert.Equal(t, *options.Value, v.Value)
		assert.Equal(t, *options.Description, v.Description)
		assert.Equal(t, *options.Category, v.Category)
	})

	t.Run("when options has a too-long description", func(t *testing.T) {
		options := VariableCreateOptions{
			Key:         String(randomString(t)),
			Value:       String(randomString(t)),
			Description: String("tortor aliquam nulla facilisi cras fermentum odio eu feugiat pretium nibh ipsum consequat nisl vel pretium lectus quam id leo in vitae turpis massa sed elementum tempus egestas sed sed risus pretium quam vulputate dignissim suspendisse in est ante in nibh mauris cursus mattis molestie a iaculis at erat pellentesque adipiscing commodo elit at imperdiet dui accumsan sit amet nulla facilisi morbi tempus iaculis urna id volutpat lacus laoreet non curabitur gravida arcu ac tortor dignissim convallis aenean et tortor"),
			Category:    Category(CategoryTerraform),
		}

		_, err := client.Variables.Create(ctx, wTest.ID, options)
		assert.Error(t, err)
	})

	t.Run("when options is missing value", func(t *testing.T) {
		options := VariableCreateOptions{
			Key:      String(randomString(t)),
			Category: Category(CategoryTerraform),
		}

		v, err := client.Variables.Create(ctx, wTest.ID, options)
		require.NoError(t, err)

		assert.NotEmpty(t, v.ID)
		assert.Equal(t, *options.Key, v.Key)
		assert.Equal(t, "", v.Value)
		assert.Equal(t, *options.Category, v.Category)
	})

	t.Run("when options is missing key", func(t *testing.T) {
		options := VariableCreateOptions{
			Value:    String(randomString(t)),
			Category: Category(CategoryTerraform),
		}

		_, err := client.Variables.Create(ctx, wTest.ID, options)
		assert.EqualError(t, err, "key is required")
	})

	t.Run("when options has an empty key", func(t *testing.T) {
		options := VariableCreateOptions{
			Key:      String(""),
			Value:    String(randomString(t)),
			Category: Category(CategoryTerraform),
		}

		_, err := client.Variables.Create(ctx, wTest.ID, options)
		assert.EqualError(t, err, "key is required")
	})

	t.Run("when options is missing category", func(t *testing.T) {
		options := VariableCreateOptions{
			Key:   String(randomString(t)),
			Value: String(randomString(t)),
		}

		_, err := client.Variables.Create(ctx, wTest.ID, options)
		assert.EqualError(t, err, "category is required")
	})

	t.Run("when workspace ID is invalid", func(t *testing.T) {
		options := VariableCreateOptions{
			Key:      String(randomString(t)),
			Value:    String(randomString(t)),
			Category: Category(CategoryTerraform),
		}

		_, err := client.Variables.Create(ctx, badIdentifier, options)
		assert.EqualError(t, err, ErrInvalidWorkspaceID.Error())
	})
}

func TestVariablesRead(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	vTest, vTestCleanup := createVariable(t, client, nil)
	defer vTestCleanup()

	t.Run("when the variable exists", func(t *testing.T) {
		v, err := client.Variables.Read(ctx, vTest.Workspace.ID, vTest.ID)
		require.NoError(t, err)
		assert.Equal(t, vTest.ID, v.ID)
		assert.Equal(t, vTest.Category, v.Category)
		assert.Equal(t, vTest.HCL, v.HCL)
		assert.Equal(t, vTest.Key, v.Key)
		assert.Equal(t, vTest.Sensitive, v.Sensitive)
		assert.Equal(t, vTest.Value, v.Value)
	})

	t.Run("when the variable does not exist", func(t *testing.T) {
		v, err := client.Variables.Read(ctx, vTest.Workspace.ID, "nonexisting")
		assert.Nil(t, v)
		assert.Equal(t, ErrResourceNotFound, err)
	})

	t.Run("without a valid workspace ID", func(t *testing.T) {
		v, err := client.Variables.Read(ctx, badIdentifier, vTest.ID)
		assert.Nil(t, v)
		assert.EqualError(t, err, ErrInvalidWorkspaceID.Error())
	})

	t.Run("without a valid variable ID", func(t *testing.T) {
		v, err := client.Variables.Read(ctx, vTest.Workspace.ID, badIdentifier)
		assert.Nil(t, v)
		assert.EqualError(t, err, "invalid value for variable ID")
	})
}

func TestVariablesUpdate(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	vTest, vTestCleanup := createVariable(t, client, nil)
	defer vTestCleanup()

	t.Run("with valid options", func(t *testing.T) {
		options := VariableUpdateOptions{
			Key:   String("newname"),
			Value: String("newvalue"),
			HCL:   Bool(true),
		}

		v, err := client.Variables.Update(ctx, vTest.Workspace.ID, vTest.ID, options)
		require.NoError(t, err)

		assert.Equal(t, *options.Key, v.Key)
		assert.Equal(t, *options.HCL, v.HCL)
		assert.Equal(t, *options.Value, v.Value)
	})

	t.Run("when updating a subset of values", func(t *testing.T) {
		options := VariableUpdateOptions{
			Key: String("someothername"),
			HCL: Bool(false),
		}

		v, err := client.Variables.Update(ctx, vTest.Workspace.ID, vTest.ID, options)
		require.NoError(t, err)

		assert.Equal(t, *options.Key, v.Key)
		assert.Equal(t, *options.HCL, v.HCL)
	})

	t.Run("with sensitive set", func(t *testing.T) {
		options := VariableUpdateOptions{
			Sensitive: Bool(true),
		}

		v, err := client.Variables.Update(ctx, vTest.Workspace.ID, vTest.ID, options)
		require.NoError(t, err)

		assert.Equal(t, *options.Sensitive, v.Sensitive)
		assert.Empty(t, v.Value) // Because its now sensitive
	})

	t.Run("without any changes", func(t *testing.T) {
		vTest, vTestCleanup := createVariable(t, client, nil)
		defer vTestCleanup()

		v, err := client.Variables.Update(ctx, vTest.Workspace.ID, vTest.ID, VariableUpdateOptions{})
		require.NoError(t, err)

		assert.Equal(t, vTest, v)
	})

	t.Run("with invalid variable ID", func(t *testing.T) {
		_, err := client.Variables.Update(ctx, badIdentifier, vTest.ID, VariableUpdateOptions{})
		assert.EqualError(t, err, ErrInvalidWorkspaceID.Error())
	})

	t.Run("with invalid variable ID", func(t *testing.T) {
		_, err := client.Variables.Update(ctx, vTest.Workspace.ID, badIdentifier, VariableUpdateOptions{})
		assert.EqualError(t, err, "invalid value for variable ID")
	})
}

func TestVariablesDelete(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	wTest, wTestCleanup := createWorkspace(t, client, nil)
	defer wTestCleanup()

	vTest, _ := createVariable(t, client, wTest)

	t.Run("with valid options", func(t *testing.T) {
		err := client.Variables.Delete(ctx, wTest.ID, vTest.ID)
		assert.NoError(t, err)
	})

	t.Run("with non existing variable ID", func(t *testing.T) {
		err := client.Variables.Delete(ctx, wTest.ID, "nonexisting")
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("with invalid workspace ID", func(t *testing.T) {
		err := client.Variables.Delete(ctx, badIdentifier, vTest.ID)
		assert.EqualError(t, err, ErrInvalidWorkspaceID.Error())
	})

	t.Run("with invalid variable ID", func(t *testing.T) {
		err := client.Variables.Delete(ctx, wTest.ID, badIdentifier)
		assert.EqualError(t, err, "invalid value for variable ID")
	})
}
