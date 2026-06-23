// Copyright IBM Corp. 2018, 2026
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRegistryModules_Update_VCSRepoMutualExclusionValidation(t *testing.T) {
	t.Parallel()
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer testServer.Close()

	client, err := NewClient(&Config{
		Address: testServer.URL,
		Token:   "fake-token",
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	moduleID := RegistryModuleID{
		Organization: "test-org",
		Name:         "test-module",
		Provider:     "aws",
		Namespace:    "test-namespace",
		RegistryName: PrivateRegistry,
	}

	t.Run("errors when both OAuthTokenID and GHAInstallationID are set", func(t *testing.T) {
		options := RegistryModuleUpdateOptions{
			VCSRepo: &RegistryModuleVCSRepoUpdateOptions{
				Identifier:        String("test-org/terraform-aws-module"),
				OAuthTokenID:      String("ot-123"),
				GHAInstallationID: String("ghain-456"),
			},
		}

		rm, err := client.RegistryModules.Update(ctx, moduleID, options)
		assert.Error(t, err)
		assert.Equal(t, ErrMutuallyExclusiveOAuthTokenAndGHAInstallation, err)
		assert.Nil(t, rm)
	})

	t.Run("succeeds when only OAuthTokenID is set", func(t *testing.T) {
		options := RegistryModuleUpdateOptions{
			VCSRepo: &RegistryModuleVCSRepoUpdateOptions{
				Identifier:   String("test-org/terraform-aws-module"),
				OAuthTokenID: String("ot-123"),
			},
		}

		// Validation passes; expect a network/API error, not our validation error
		_, err := client.RegistryModules.Update(ctx, moduleID, options)
		if err != nil {
			assert.NotEqual(t, ErrMutuallyExclusiveOAuthTokenAndGHAInstallation, err)
		}
	})

	t.Run("succeeds when only GHAInstallationID is set", func(t *testing.T) {
		options := RegistryModuleUpdateOptions{
			VCSRepo: &RegistryModuleVCSRepoUpdateOptions{
				Identifier:        String("test-org/terraform-aws-module"),
				GHAInstallationID: String("ghain-456"),
			},
		}

		// Validation passes; expect a network/API error, not our validation error
		_, err := client.RegistryModules.Update(ctx, moduleID, options)
		if err != nil {
			assert.NotEqual(t, ErrMutuallyExclusiveOAuthTokenAndGHAInstallation, err)
		}
	})

	t.Run("succeeds when neither OAuthTokenID nor GHAInstallationID is set", func(t *testing.T) {
		options := RegistryModuleUpdateOptions{
			VCSRepo: &RegistryModuleVCSRepoUpdateOptions{
				Identifier: String("test-org/terraform-aws-module"),
			},
		}

		// Validation passes; expect a network/API error, not our validation error
		_, err := client.RegistryModules.Update(ctx, moduleID, options)
		if err != nil {
			assert.NotEqual(t, ErrMutuallyExclusiveOAuthTokenAndGHAInstallation, err)
		}
	})
}
