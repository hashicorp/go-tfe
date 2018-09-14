package tfe

import (
	"context"
	"fmt"
	"testing"

	tfe "github.com/scottwinkler/go-tfe"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistryModulesPublish(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()
	oTest, oTestCleanup := createOrganization(t, client)
	defer oTestCleanup()
	otTest, otTestCleanup := createOAuthToken(t, client, oTest.Name)
	defer otTestCleanup()
	ocTest, ocTestCleanup := createOAuthClient(t, client, nil)
	defer ocTestCleanup()

	t.Run("normal publish", func(t *testing.T) {
		name := lambda
		provider := iam
		identifier := fmt.Sprintf("scottwinkler/terraform-%s-%s", name, provider)
		rm, err := client.RegistryModules.Publish(ctx, RegistryModulePublishOptions{
			VCSRepo: &tfe.VCSRepo{Identifier: identifier, OAuthTokenID: ot.ID},
		})
		require.NoError(t, err)
		assert.Equal(t, name, rm.Name)
		assert.Equal(t, provider, rm.Provider)
	})
}

func TestRegistryModulesDelete(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()
	oTest, oTestCleanup := createOrganization(t, client)
	rmTest, rmTestCleanup := createRegistryModule(t, client, oTest.Name)
	defer rmTestCleanup()

	t.Run("normal delete", func(t *testing.T) {
		_, err := client.RegistryModules.Delete(ctx, oTest.Name, rmTest.Name, rmTest.Provider, "", "")
		assert.NoError(t, err)
	})
}
