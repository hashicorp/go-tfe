package tfe

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewClient(t *testing.T) {
	t.Run("fails if config is nil", func(t *testing.T) {
		_, err := NewClient(nil)
		assert.EqualError(t, err, "Missing client config")
	})

	t.Run("fails if token is empty", func(t *testing.T) {
		_, err := NewClient(&Config{})
		assert.EqualError(t, err, "Missing client token")
	})

	t.Run("makes a new client with good settings", func(t *testing.T) {
		httpClient := &http.Client{}

		config := &Config{
			Address:    "http://tfe.foo",
			Token:      "abcd1234",
			HTTPClient: httpClient,
		}

		client, err := NewClient(config)
		assert.Nil(t, err)

		assert.Equal(t, config.Address, client.config.Address)
		assert.Equal(t, config.Token, client.config.Token)
		assert.Equal(t, httpClient, client.http)
	})

	t.Run("creates a default http client", func(t *testing.T) {
		client, err := NewClient(&Config{
			Token: "abcd1234",
		})
		assert.Nil(t, err)

		assert.NotNil(t, client.http)
	})
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	assert.Equal(t, DefaultAddress, config.Address)
	assert.Equal(t, "", config.Token)
	assert.Nil(t, config.HTTPClient)
}
