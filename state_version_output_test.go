package tfe

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

const waitForStateVersionOutputs = 500 * time.Millisecond

func TestStateVersionOutputsRead(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	wTest1, wTest1Cleanup := createWorkspace(t, client, nil)
	defer wTest1Cleanup()

	_, svTestCleanup := createStateVersion(t, client, 0, wTest1)
	defer svTestCleanup()

	// give TFC some time to process the statefile and extract the outputs.
	time.Sleep(waitForStateVersionOutputs)

	sv, err := client.StateVersions.Current(ctx, wTest1.ID)
	if err != nil {
		t.Fatal(err)
	}

	output := sv.Outputs[0]

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

}
