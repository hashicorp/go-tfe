// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVariablesList(t *testing.T) {
	t.Parallel()
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

func TestVariablesListAll(t *testing.T) {
	t.Parallel()
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	t.Cleanup(orgTestCleanup)

	prjTest, prjTestCleanup := createProject(t, client, orgTest)
	t.Cleanup(prjTestCleanup)

	wTest, wTestCleanup := createWorkspaceWithOptions(t, client, orgTest, WorkspaceCreateOptions{
		Name:    String(randomString(t)),
		Project: prjTest,
	})
	t.Cleanup(wTestCleanup)

	orgVarset, orgVarsetCleanup := createVariableSet(t, client, orgTest, VariableSetCreateOptions{})
	t.Cleanup(orgVarsetCleanup)

	prjVarset, prjVarsetCleanup := createVariableSet(t, client, orgTest, VariableSetCreateOptions{
		Parent: &Parent{
			Organization: orgTest,
			Project:      prjTest,
		},
	})
	t.Cleanup(prjVarsetCleanup)

	glVar, glVarCleanup := createVariableSetVariable(t, client, orgVarset, VariableSetVariableCreateOptions{
		Key:         String("key1"),
		Value:       String("gl_value1"),
		Category:    Category(CategoryTerraform),
		Description: String(randomString(t)),
	})
	t.Cleanup(glVarCleanup)

	glVarOverwrite, glVarOverwriteCleanup := createVariableSetVariable(t, client, orgVarset, VariableSetVariableCreateOptions{
		Key:         String("key2"),
		Value:       String("gl_value2"),
		Category:    Category(CategoryTerraform),
		Description: String(randomString(t)),
	})
	t.Cleanup(glVarOverwriteCleanup)

	prjVar, prjVarCleanup := createVariableSetVariable(t, client, prjVarset, VariableSetVariableCreateOptions{
		Key:         String("key3"),
		Value:       String("prj_value3"),
		Category:    Category(CategoryTerraform),
		Description: String(randomString(t)),
	})
	t.Cleanup(prjVarCleanup)

	prjVarOverwrite, prjVarOverwriteCleanup := createVariableSetVariable(t, client, prjVarset, VariableSetVariableCreateOptions{
		Key:         String("key4"),
		Value:       String("prj_value4"),
		Category:    Category(CategoryTerraform),
		Description: String(randomString(t)),
	})
	t.Cleanup(prjVarOverwriteCleanup)

	wsVar1, wsVar1Cleanup := createVariableWithOptions(t, client, wTest, VariableCreateOptions{
		Key:         String("key2"),
		Value:       String("ws_value2"),
		Category:    Category(CategoryTerraform),
		Description: String(randomString(t)),
	})
	t.Cleanup(wsVar1Cleanup)

	wsVar2, wsVar2Cleanup := createVariableWithOptions(t, client, wTest, VariableCreateOptions{
		Key:         String("key4"),
		Value:       String("ws_value4"),
		Category:    Category(CategoryTerraform),
		Description: String(randomString(t)),
	})
	t.Cleanup(wsVar2Cleanup)

	wsVar3, wsVar3Cleanup := createVariableWithOptions(t, client, wTest, VariableCreateOptions{
		Key:         String("key5"),
		Value:       String("ws_value5"),
		Category:    Category(CategoryTerraform),
		Description: String(randomString(t)),
	})
	t.Cleanup(wsVar3Cleanup)

	applyVariableSetToWorkspace(t, client, orgVarset.ID, wTest.ID)
	applyVariableSetToWorkspace(t, client, prjVarset.ID, wTest.ID)

	t.Run("when /workspaces/{external_id}/all-vars API is called", func(t *testing.T) {
		vl, err := client.Variables.ListAll(ctx, wTest.ID, nil)
		require.NoError(t, err)
		assert.NotNilf(t, vl, "expected to get a non-empty variables list")

		variableIDToValueMap := make(map[string]string)
		for _, variable := range vl.Items {
			variableIDToValueMap[variable.ID] = variable.Value
		}
		assert.Equal(t, len(vl.Items), 5)
		assert.NotContains(t, variableIDToValueMap, glVarOverwrite.ID)
		assert.NotContains(t, variableIDToValueMap, prjVarOverwrite.ID)
		assert.Contains(t, variableIDToValueMap, glVar.ID)
		assert.Contains(t, variableIDToValueMap, prjVar.ID)
		assert.Contains(t, variableIDToValueMap, wsVar1.ID)
		assert.Contains(t, variableIDToValueMap, wsVar2.ID)
		assert.Contains(t, variableIDToValueMap, wsVar3.ID)
		assert.Equal(t, glVar.Value, variableIDToValueMap[glVar.ID])
		assert.Equal(t, prjVar.Value, variableIDToValueMap[prjVar.ID])
		assert.Equal(t, wsVar1.Value, variableIDToValueMap[wsVar1.ID])
		assert.Equal(t, wsVar2.Value, variableIDToValueMap[wsVar2.ID])
		assert.Equal(t, wsVar3.Value, variableIDToValueMap[wsVar3.ID])
	})

	t.Run("when workspace ID is invalid ID", func(t *testing.T) {
		vl, err := client.Variables.ListAll(ctx, badIdentifier, nil)
		assert.Nilf(t, vl, "expected variables list to be nil")
		assert.EqualError(t, err, ErrInvalidWorkspaceID.Error())
	})
}

func TestVariablesCreate(t *testing.T) {
	t.Parallel()
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

		// Refresh workspace once the variable is created.
		reWorkspace, err := client.Workspaces.ReadByID(ctx, wTest.ID)
		require.NoError(t, err)

		assert.NotEmpty(t, v.ID)
		assert.Equal(t, *options.Key, v.Key)
		assert.Equal(t, *options.Value, v.Value)
		assert.Equal(t, *options.Description, v.Description)
		assert.Equal(t, *options.Category, v.Category)
		// The workspace isn't returned correcly by the API.
		// assert.Equal(t, *options.Workspace, v.Workspace)
		assert.NotEmpty(t, v.VersionID)
		// Validate that the same Variable is now listed in Workspace relations.
		assert.NotEmpty(t, reWorkspace.Variables)
		assert.Equal(t, reWorkspace.Variables[0].ID, v.ID)
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
		assert.NotEmpty(t, v.VersionID)
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
		assert.NotEmpty(t, v.VersionID)
	})

	t.Run("when options has a too-long description", func(t *testing.T) {
		options := VariableCreateOptions{
			Key:         String(randomString(t)),
			Value:       String(randomString(t)),
			Description: String("tortor aliquam nulla go lint is fussy about spelling cras fermentum odio eu feugiat pretium nibh ipsum consequat nisl vel pretium lectus quam id leo in vitae turpis massa sed elementum tempus egestas sed sed risus pretium quam vulputate dignissim suspendisse in est ante in nibh mauris cursus mattis molestie a iaculis at erat pellentesque adipiscing commodo elit at imperdiet dui accumsan sit amet nulla redacted morbi tempus iaculis urna id volutpat lacus laoreet non curabitur gravida arcu ac tortor dignissim convallis aenean et tortor"),
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
		assert.NotEmpty(t, v.VersionID)
	})

	t.Run("when options is missing key", func(t *testing.T) {
		options := VariableCreateOptions{
			Value:    String(randomString(t)),
			Category: Category(CategoryTerraform),
		}

		_, err := client.Variables.Create(ctx, wTest.ID, options)
		assert.Equal(t, err, ErrRequiredKey)
	})

	t.Run("when options has an empty key", func(t *testing.T) {
		options := VariableCreateOptions{
			Key:      String(""),
			Value:    String(randomString(t)),
			Category: Category(CategoryTerraform),
		}

		_, err := client.Variables.Create(ctx, wTest.ID, options)
		assert.Equal(t, err, ErrRequiredKey)
	})

	t.Run("when options is missing category", func(t *testing.T) {
		options := VariableCreateOptions{
			Key:   String(randomString(t)),
			Value: String(randomString(t)),
		}

		_, err := client.Variables.Create(ctx, wTest.ID, options)
		assert.Equal(t, err, ErrRequiredCategory)
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
	t.Parallel()
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
		assert.Equal(t, vTest.VersionID, v.VersionID)
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
		assert.Equal(t, err, ErrInvalidVariableID)
	})
}

func TestVariablesUpdate(t *testing.T) {
	t.Parallel()
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
		assert.NotEqual(t, vTest.VersionID, v.VersionID)
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
		assert.NotEqual(t, vTest.VersionID, v.VersionID)
	})

	t.Run("with sensitive set", func(t *testing.T) {
		options := VariableUpdateOptions{
			Sensitive: Bool(true),
		}

		v, err := client.Variables.Update(ctx, vTest.Workspace.ID, vTest.ID, options)
		require.NoError(t, err)

		assert.Equal(t, *options.Sensitive, v.Sensitive)
		assert.Empty(t, v.Value) // Because its now sensitive
		assert.NotEqual(t, vTest.VersionID, v.VersionID)
	})

	t.Run("with category set", func(t *testing.T) {
		category := CategoryEnv
		options := VariableUpdateOptions{
			Category: &category,
		}

		v, err := client.Variables.Update(ctx, vTest.Workspace.ID, vTest.ID, options)
		require.NoError(t, err)

		assert.Equal(t, *options.Category, v.Category)
		assert.NotEqual(t, vTest.VersionID, v.VersionID)
	})

	t.Run("without any changes", func(t *testing.T) {
		vTest, vTestCleanup := createVariable(t, client, nil)
		defer vTestCleanup()

		v, err := client.Variables.Update(ctx, vTest.Workspace.ID, vTest.ID, VariableUpdateOptions{})
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
		_, err := client.Variables.Update(ctx, badIdentifier, vTest.ID, VariableUpdateOptions{})
		assert.EqualError(t, err, ErrInvalidWorkspaceID.Error())
	})

	t.Run("with invalid variable ID", func(t *testing.T) {
		_, err := client.Variables.Update(ctx, vTest.Workspace.ID, badIdentifier, VariableUpdateOptions{})
		assert.Equal(t, err, ErrInvalidVariableID)
	})
}

func TestVariablesDelete(t *testing.T) {
	t.Parallel()
	client := testClient(t)
	ctx := context.Background()

	wTest, wTestCleanup := createWorkspace(t, client, nil)
	defer wTestCleanup()

	vTest, _ := createVariable(t, client, wTest)

	t.Run("with valid options", func(t *testing.T) {
		err := client.Variables.Delete(ctx, wTest.ID, vTest.ID)
		require.NoError(t, err)
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
		assert.Equal(t, err, ErrInvalidVariableID)
	})
}
