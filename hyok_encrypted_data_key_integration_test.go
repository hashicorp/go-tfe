package tfe

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// These tests are intended for local execution only, as data encryption keys for HYOK requires specific conditions
// for tests to run successfully. To test locally:
// 1. Follow the instructions outlined in hyok_configuration_integration_test.go.
// 2. Set hyokEncryptedDataKeyID to the ID of an existing data encryption key

func TestHYOKEncryptedDataKeyRead(t *testing.T) {
	skipHYOKIntegrationTests(t)

	client := testClient(t)
	ctx := context.Background()

	t.Run("read an existing encrypted data key", func(t *testing.T) {
		hyokEncryptedDataKeyID := os.Getenv("HYOK_ENCRYPTED_DATA_KEY_ID")
		if hyokEncryptedDataKeyID == "" {
			t.Fatal("Export a valid HYOK_ENCRYPTED_DATA_KEY_ID before running this test!")
		}

		_, err := client.HYOKEncryptedDataKeys.Read(ctx, hyokEncryptedDataKeyID)
		require.NoError(t, err)
	})
}
