package tfe

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListConfigurationVersions(t *testing.T) {
	client := testClient(t)

	ws, wsCleanup := createWorkspace(t, client, nil)
	defer wsCleanup()

	cv1, _ := createConfigurationVersion(t, client, ws)
	cv2, _ := createConfigurationVersion(t, client, ws)

	resp, err := client.ListConfigurationVersions(
		&ListConfigurationVersionsInput{
			WorkspaceID: ws.ID,
		},
	)
	require.Nil(t, err)

	found := []string{}
	for _, cv := range resp {
		found = append(found, *cv.ID)
	}

	assert.Contains(t, found, *cv1.ID)
	assert.Contains(t, found, *cv2.ID)
}
