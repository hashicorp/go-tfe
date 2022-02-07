package tfe

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
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
		vl, err := client.VariableSetVariables.List(ctx, vsTest.ID, VariableSetVariableListOptions{})
		require.NoError(t, err)
		assert.Contains(t, vl.Items, vTest1)
		assert.Contains(t, vl.Items, vTest2)

		t.Skip("paging not supported yet in API")
		assert.Equal(t, 1, vl.CurrentPage)
		assert.Equal(t, 2, vl.TotalCount)
	})
}
