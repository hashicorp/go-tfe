// Copyright IBM Corp. 2018, 2025
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRegistryModules_Update_AgentExecutionValidation(t *testing.T) {
	// Create a test server for API calls
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// This shouldn't be called for validation errors, but provide a response just in case
		w.WriteHeader(http.StatusOK)
	}))
	defer testServer.Close()

	// Create a client pointing to the test server
	client, err := NewClient(&Config{
		Address: testServer.URL,
		Token:   "fake-token",
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	t.Run("errors when remote execution mode has agent pool ID", func(t *testing.T) {
		moduleID := RegistryModuleID{
			Organization: "test-org",
			Name:         "test-module",
			Provider:     "aws",
			Namespace:    "test-namespace",
			RegistryName: PrivateRegistry,
		}

		options := RegistryModuleUpdateOptions{
			TestConfig: &RegistryModuleTestConfigOptions{
				AgentExecutionMode: AgentExecutionModePtr(AgentExecutionModeRemote),
				AgentPoolID:        String("apool-123"),
			},
		}

		rm, err := client.RegistryModules.Update(ctx, moduleID, options)
		assert.Error(t, err)
		assert.Equal(t, ErrAgentPoolNotRequiredForRemoteExecution, err)
		assert.Nil(t, rm)
	})

	t.Run("succeeds when agent execution mode has agent pool ID", func(t *testing.T) {
		moduleID := RegistryModuleID{
			Organization: "test-org",
			Name:         "test-module",
			Provider:     "aws",
			Namespace:    "test-namespace",
			RegistryName: PrivateRegistry,
		}

		options := RegistryModuleUpdateOptions{
			TestConfig: &RegistryModuleTestConfigOptions{
				AgentExecutionMode: AgentExecutionModePtr(AgentExecutionModeAgent),
				AgentPoolID:        String("apool-123"),
			},
		}

		// This test only validates that the validation logic doesn't fail
		// The actual API call will fail since we're using a test client,
		// but that's expected and not what we're testing here
		_, err := client.RegistryModules.Update(ctx, moduleID, options)
		// We expect some error (likely network/API related), but NOT our validation error
		if err != nil {
			assert.NotEqual(t, ErrAgentPoolNotRequiredForRemoteExecution, err)
		}
	})

	t.Run("succeeds when remote execution mode has no agent pool ID", func(t *testing.T) {
		moduleID := RegistryModuleID{
			Organization: "test-org",
			Name:         "test-module",
			Provider:     "aws",
			Namespace:    "test-namespace",
			RegistryName: PrivateRegistry,
		}

		options := RegistryModuleUpdateOptions{
			TestConfig: &RegistryModuleTestConfigOptions{
				AgentExecutionMode: AgentExecutionModePtr(AgentExecutionModeRemote),
				AgentPoolID:        nil,
			},
		}

		// This test only validates that the validation logic doesn't fail
		_, err := client.RegistryModules.Update(ctx, moduleID, options)
		// We expect some error (likely network/API related), but NOT our validation error
		if err != nil {
			assert.NotEqual(t, ErrAgentPoolNotRequiredForRemoteExecution, err)
		}
	})
}

func TestRegistryModules_CreateWithVCSConnection_AgentExecutionValidation(t *testing.T) {
	// Create a test server for API calls
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// This shouldn't be called for validation errors, but provide a response just in case
		w.WriteHeader(http.StatusOK)
	}))
	defer testServer.Close()

	// Create a client pointing to the test server
	client, err := NewClient(&Config{
		Address: testServer.URL,
		Token:   "fake-token",
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	t.Run("errors when remote execution mode has agent pool ID", func(t *testing.T) {
		options := RegistryModuleCreateWithVCSConnectionOptions{
			VCSRepo: &RegistryModuleVCSRepoOptions{
				Identifier:        String("test/repo"),
				OAuthTokenID:      String("ot-123"),
				DisplayIdentifier: String("test/repo"),
			},
			TestConfig: &RegistryModuleTestConfigOptions{
				AgentExecutionMode: AgentExecutionModePtr(AgentExecutionModeRemote),
				AgentPoolID:        String("apool-123"),
			},
		}

		rm, err := client.RegistryModules.CreateWithVCSConnection(ctx, options)
		assert.Error(t, err)
		assert.Equal(t, ErrAgentPoolNotRequiredForRemoteExecution, err)
		assert.Nil(t, rm)
	})

	t.Run("succeeds when agent execution mode has agent pool ID", func(t *testing.T) {
		options := RegistryModuleCreateWithVCSConnectionOptions{
			VCSRepo: &RegistryModuleVCSRepoOptions{
				Identifier:        String("test/repo"),
				OAuthTokenID:      String("ot-123"),
				DisplayIdentifier: String("test/repo"),
			},
			TestConfig: &RegistryModuleTestConfigOptions{
				AgentExecutionMode: AgentExecutionModePtr(AgentExecutionModeAgent),
				AgentPoolID:        String("apool-123"),
			},
		}

		// This test only validates that the validation logic doesn't fail
		_, err := client.RegistryModules.CreateWithVCSConnection(ctx, options)
		// We expect some error (likely network/API related), but NOT our validation error
		if err != nil {
			assert.NotEqual(t, ErrAgentPoolNotRequiredForRemoteExecution, err)
		}
	})

	t.Run("succeeds when remote execution mode has no agent pool ID", func(t *testing.T) {
		options := RegistryModuleCreateWithVCSConnectionOptions{
			VCSRepo: &RegistryModuleVCSRepoOptions{
				Identifier:        String("test/repo"),
				OAuthTokenID:      String("ot-123"),
				DisplayIdentifier: String("test/repo"),
			},
			TestConfig: &RegistryModuleTestConfigOptions{
				AgentExecutionMode: AgentExecutionModePtr(AgentExecutionModeRemote),
				AgentPoolID:        nil,
			},
		}

		// This test only validates that the validation logic doesn't fail
		_, err := client.RegistryModules.CreateWithVCSConnection(ctx, options)
		// We expect some error (likely network/API related), but NOT our validation error
		if err != nil {
			assert.NotEqual(t, ErrAgentPoolNotRequiredForRemoteExecution, err)
		}
	})

	t.Run("succeeds when TestConfig is nil", func(t *testing.T) {
		options := RegistryModuleCreateWithVCSConnectionOptions{
			VCSRepo: &RegistryModuleVCSRepoOptions{
				Identifier:        String("test/repo"),
				OAuthTokenID:      String("ot-123"),
				DisplayIdentifier: String("test/repo"),
			},
			TestConfig: nil,
		}

		// This test only validates that the validation logic doesn't fail
		_, err := client.RegistryModules.CreateWithVCSConnection(ctx, options)
		// We expect some error (likely network/API related), but NOT our validation error
		if err != nil {
			assert.NotEqual(t, ErrAgentPoolNotRequiredForRemoteExecution, err)
		}
	})
}
