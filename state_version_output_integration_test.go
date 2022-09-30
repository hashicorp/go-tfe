//go:build integration
// +build integration

package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStateVersionOutputsRead(t *testing.T) {
	skipIfNotCINode(t)

	client := testClient(t)
	ctx := context.Background()

	wTest1, wTest1Cleanup := createWorkspace(t, client, nil)
	defer wTest1Cleanup()

	svTest, svTestCleanup := createStateVersion(t, client, 0, wTest1)
	defer svTestCleanup()

	// give TFC some time to process the statefile and extract the outputs.
	waitForSVOutputs(t, client, svTest.ID)

	curOpts := &StateVersionCurrentOptions{
		Include: []StateVersionIncludeOpt{SVoutputs},
	}

	sv, err := client.StateVersions.ReadCurrentWithOptions(ctx, wTest1.ID, curOpts)
	if err != nil {
		t.Fatal(err)
	}

	require.NotEmpty(t, sv.Outputs)
	require.NotNil(t, sv.Outputs[0])

	output := sv.Outputs[0]

	t.Run("Read by ID", func(t *testing.T) {
		t.Run("when a state output exists", func(t *testing.T) {
			so, err := client.StateVersionOutputs.Read(ctx, output.ID)
			require.NoError(t, err)

			assert.Equal(t, so.ID, output.ID)
			assert.Equal(t, so.Name, output.Name)
			assert.Equal(t, so.Value, output.Value)
		})

		t.Run("when a state output does not exist", func(t *testing.T) {
			so, err := client.StateVersionOutputs.Read(ctx, "wsout-J2zM24JPAAAAAAAA")
			assert.Nil(t, so)
			assert.Equal(t, ErrResourceNotFound, err)
		})
	})

	t.Run("Read current workspace outputs", func(t *testing.T) {
		so, err := client.StateVersionOutputs.ReadCurrent(ctx, wTest1.ID)
		require.NoError(t, err)
		assert.NotEmpty(t, so.Items)
	})

	t.Run("Sensitive secrets are null", func(t *testing.T) {
		so, err := client.StateVersionOutputs.ReadCurrent(ctx, wTest1.ID)
		require.NoError(t, err)
		require.NotEmpty(t, so.Items)

		var found *StateVersionOutput = nil
		for _, s := range so.Items {
			if s.Name == "test_output_string" {
				found = s
				break
			}
		}

		assert.NotNil(t, found)
		assert.True(t, found.Sensitive)
		assert.Nil(t, found.Value)
	})
}
