package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPoliciesCreate_Beta(t *testing.T) {
	skipIfFreeOnly(t)
	skipUnlessBeta(t)

	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	t.Run("with valid options - Sentinel", func(t *testing.T) {
		name := randomString(t)
		options := PolicyCreateOptions{
			Name:        String(name),
			Description: String("A sample policy"),
			Kind:        Sentinel,
			Enforce: []*EnforcementOptions{
				{
					Path: String(name + ".sentinel"),
					Mode: EnforcementMode(EnforcementSoft),
				},
			},
		}

		p, err := client.Policies.Create(ctx, orgTest.Name, options)
		require.NoError(t, err)

		// Get a refreshed view from the API.
		refreshed, err := client.Policies.Read(ctx, p.ID)
		require.NoError(t, err)

		for _, item := range []*Policy{
			p,
			refreshed,
		} {
			assert.NotEmpty(t, item.ID)
			assert.Equal(t, *options.Name, item.Name)
			assert.Equal(t, options.Kind, item.Kind)
			assert.Nil(t, options.Query)
			assert.Equal(t, *options.Description, item.Description)
		}
	})

	t.Run("with no kind", func(t *testing.T) {
		name := randomString(t)
		options := PolicyCreateOptions{
			Name:        String(name),
			Description: String("A sample policy"),
			Enforce: []*EnforcementOptions{
				{
					Path: String(name + ".sentinel"),
					Mode: EnforcementMode(EnforcementSoft),
				},
			},
		}

		p, err := client.Policies.Create(ctx, orgTest.Name, options)
		require.NoError(t, err)

		// Get a refreshed view from the API.
		refreshed, err := client.Policies.Read(ctx, p.ID)
		require.NoError(t, err)

		for _, item := range []*Policy{
			p,
			refreshed,
		} {
			assert.NotEmpty(t, item.ID)
			assert.Equal(t, *options.Name, item.Name)
			assert.Equal(t, Sentinel, item.Kind)
			assert.Equal(t, *options.Description, item.Description)
		}
	})

	t.Run("with valid options - OPA", func(t *testing.T) {
		name := randomString(t)
		options := PolicyCreateOptions{
			Name:        String(name),
			Description: String("A sample policy"),
			Kind:        OPA,
			Query:       String("terraform.main"),
			Enforce: []*EnforcementOptions{
				{
					Path: String(name + ".rego"),
					Mode: EnforcementMode(EnforcementMandatory),
				},
			},
		}

		p, err := client.Policies.Create(ctx, orgTest.Name, options)
		require.NoError(t, err)

		// Get a refreshed view from the API.
		refreshed, err := client.Policies.Read(ctx, p.ID)
		require.NoError(t, err)

		for _, item := range []*Policy{
			p,
			refreshed,
		} {
			assert.NotEmpty(t, item.ID)
			assert.Equal(t, *options.Name, item.Name)
			assert.Equal(t, options.Kind, item.Kind)
			assert.Equal(t, *options.Query, *item.Query)
			assert.Equal(t, *options.Description, item.Description)
		}
	})

	t.Run("when options has an invalid name - OPA", func(t *testing.T) {
		p, err := client.Policies.Create(ctx, orgTest.Name, PolicyCreateOptions{
			Name:  String(badIdentifier),
			Kind:  OPA,
			Query: String("terraform.main"),
			Enforce: []*EnforcementOptions{
				{
					Path: String(badIdentifier + ".rego"),
					Mode: EnforcementMode(EnforcementAdvisory),
				},
			},
		})
		assert.Nil(t, p)
		assert.EqualError(t, err, ErrInvalidName.Error())
	})

	t.Run("when options is missing name - OPA", func(t *testing.T) {
		p, err := client.Policies.Create(ctx, orgTest.Name, PolicyCreateOptions{
			Kind:  OPA,
			Query: String("terraform.main"),
			Enforce: []*EnforcementOptions{
				{
					Path: String(randomString(t) + ".rego"),
					Mode: EnforcementMode(EnforcementSoft),
				},
			},
		})
		assert.Nil(t, p)
		assert.EqualError(t, err, ErrRequiredName.Error())
	})

	t.Run("when options is missing query - OPA", func(t *testing.T) {
		name := randomString(t)
		p, err := client.Policies.Create(ctx, orgTest.Name, PolicyCreateOptions{
			Name: String(name),
			Kind: OPA,
			Enforce: []*EnforcementOptions{
				{
					Path: String(randomString(t) + ".rego"),
					Mode: EnforcementMode(EnforcementSoft),
				},
			},
		})
		assert.Nil(t, p)
		assert.Equal(t, err, ErrRequiredQuery)
	})

	t.Run("when options is missing an enforcement", func(t *testing.T) {
		options := PolicyCreateOptions{
			Name:  String(randomString(t)),
			Kind:  OPA,
			Query: String("terraform.main"),
		}

		p, err := client.Policies.Create(ctx, orgTest.Name, options)
		assert.Nil(t, p)
		assert.Equal(t, err, ErrRequiredEnforce)
	})

	t.Run("when options is missing enforcement path", func(t *testing.T) {
		options := PolicyCreateOptions{
			Name:  String(randomString(t)),
			Kind:  OPA,
			Query: String("terraform.main"),
			Enforce: []*EnforcementOptions{
				{
					Mode: EnforcementMode(EnforcementSoft),
				},
			},
		}

		p, err := client.Policies.Create(ctx, orgTest.Name, options)
		assert.Nil(t, p)
		assert.Equal(t, err, ErrRequiredEnforcementPath)
	})

	t.Run("when options is missing enforcement mode", func(t *testing.T) {
		name := randomString(t)
		options := PolicyCreateOptions{
			Name:  String(name),
			Kind:  OPA,
			Query: String("terraform.main"),
			Enforce: []*EnforcementOptions{
				{
					Path: String(name + ".sentinel"),
				},
			},
		}

		p, err := client.Policies.Create(ctx, orgTest.Name, options)
		assert.Nil(t, p)
		assert.Equal(t, err, ErrRequiredEnforcementMode)
	})

	t.Run("when options has an invalid organization", func(t *testing.T) {
		p, err := client.Policies.Create(ctx, badIdentifier, PolicyCreateOptions{
			Name: String("foo"),
		})
		assert.Nil(t, p)
		assert.EqualError(t, err, ErrInvalidOrg.Error())
	})
}

func TestPoliciesList_Beta(t *testing.T) {
	skipIfFreeOnly(t)
	skipUnlessBeta(t)

	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	pTest1, pTestCleanup1 := createPolicy(t, client, orgTest)
	defer pTestCleanup1()
	pTest2, pTestCleanup2 := createPolicy(t, client, orgTest)
	defer pTestCleanup2()
	opaOptions := PolicyCreateOptions{
		Kind:  OPA,
		Query: String("data.example.rule"),
		Enforce: []*EnforcementOptions{
			{
				Mode: EnforcementMode(EnforcementMandatory),
			},
		},
	}
	pTest3, pTestCleanup3 := createPolicyWithOptions(t, client, orgTest, opaOptions)
	defer pTestCleanup3()

	t.Run("without list options", func(t *testing.T) {
		pl, err := client.Policies.List(ctx, orgTest.Name, nil)
		require.NoError(t, err)
		assert.Contains(t, pl.Items, pTest1)
		assert.Contains(t, pl.Items, pTest2)
		assert.Contains(t, pl.Items, pTest3)

		assert.Equal(t, 1, pl.CurrentPage)
		assert.Equal(t, 3, pl.TotalCount)
	})

	t.Run("with pagination", func(t *testing.T) {
		// Request a page number which is out of range. The result should
		// be successful, but return no results if the paging options are
		// properly passed along.
		pl, err := client.Policies.List(ctx, orgTest.Name, &PolicyListOptions{
			ListOptions: ListOptions{
				PageNumber: 999,
				PageSize:   100,
			},
		})
		require.NoError(t, err)

		assert.Empty(t, pl.Items)
		assert.Equal(t, 999, pl.CurrentPage)
		assert.Equal(t, 3, pl.TotalCount)
	})

	t.Run("with search", func(t *testing.T) {
		// Search by one of the policy's names; we should get only that policy
		// and pagination data should reflect the search as well
		pl, err := client.Policies.List(ctx, orgTest.Name, &PolicyListOptions{
			Search: pTest1.Name,
		})
		require.NoError(t, err)

		assert.Contains(t, pl.Items, pTest1)
		assert.NotContains(t, pl.Items, pTest2)
		assert.Equal(t, 1, pl.CurrentPage)
		assert.Equal(t, 1, pl.TotalCount)
	})

	t.Run("with filter by kind", func(t *testing.T) {
		pl, err := client.Policies.List(ctx, orgTest.Name, &PolicyListOptions{
			Kind: OPA,
		})
		require.NoError(t, err)

		assert.Contains(t, pl.Items, pTest3)
		assert.NotContains(t, pl.Items, pTest1)
		assert.NotContains(t, pl.Items, pTest2)
		assert.Equal(t, 1, pl.CurrentPage)
		assert.Equal(t, 1, pl.TotalCount)
	})

	t.Run("without a valid organization", func(t *testing.T) {
		ps, err := client.Policies.List(ctx, badIdentifier, nil)
		assert.Nil(t, ps)
		assert.EqualError(t, err, ErrInvalidOrg.Error())
	})
}

func TestPoliciesUpdate_Beta(t *testing.T) {
	skipIfFreeOnly(t)
	skipUnlessBeta(t)

	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	t.Run("with a new query", func(t *testing.T) {
		options := PolicyCreateOptions{
			Description: String("A sample policy"),
			Kind:        OPA,
			Query:       String("data.example.rule"),
			Enforce: []*EnforcementOptions{
				{
					Mode: EnforcementMode(EnforcementMandatory),
				},
			},
		}
		pBefore, pBeforeCleanup := createUploadedPolicyWithOptions(t, client, true, orgTest, options)
		defer pBeforeCleanup()

		pAfter, err := client.Policies.Update(ctx, pBefore.ID, PolicyUpdateOptions{
			Query: String("terraform.policy1.deny"),
		})
		require.NoError(t, err)

		assert.Equal(t, pBefore.Name, pAfter.Name)
		assert.Equal(t, pBefore.Enforce, pAfter.Enforce)
		assert.NotEqual(t, *pBefore.Query, *pAfter.Query)
		assert.Equal(t, "terraform.policy1.deny", *pAfter.Query)
	})

	t.Run("update query when kind is not OPA", func(t *testing.T) {
		pBefore, pBeforeCleanup := createUploadedPolicy(t, client, true, orgTest)
		defer pBeforeCleanup()

		pAfter, err := client.Policies.Update(ctx, pBefore.ID, PolicyUpdateOptions{
			Query: String("terraform.policy1.deny"),
		})
		require.NoError(t, err)

		assert.Equal(t, pBefore.Name, pAfter.Name)
		assert.Equal(t, pBefore.Enforce, pAfter.Enforce)
		assert.Equal(t, Sentinel, pAfter.Kind)
		assert.Nil(t, pAfter.Query)
	})
}
