package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPolicySetParametersList(t *testing.T) {
	skipIfFreeOnly(t)

	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	psTest, pTestCleanup := createPolicySet(t, client, orgTest, nil, nil)
	defer pTestCleanup()

	pTest1, pTestCleanup1 := createPolicySetParameter(t, client, psTest)
	defer pTestCleanup1()
	pTest2, pTestCleanup2 := createPolicySetParameter(t, client, psTest)
	defer pTestCleanup2()

	t.Run("without list options", func(t *testing.T) {
		pl, err := client.PolicySetParameters.List(ctx, psTest.ID, PolicySetParameterListOptions{})
		require.NoError(t, err)
		assert.Contains(t, pl.Items, pTest1)
		assert.Contains(t, pl.Items, pTest2)

		t.Skip("paging not supported yet in API")
		assert.Equal(t, 1, pl.CurrentPage)
		assert.Equal(t, 2, pl.TotalCount)
	})

	t.Run("with list options", func(t *testing.T) {
		t.Skip("paging not supported yet in API")
		// Request a page number which is out of range. The result should
		// be successful, but return no results if the paging options are
		// properly passed along.
		pl, err := client.PolicySetParameters.List(ctx, psTest.ID, PolicySetParameterListOptions{
			ListOptions: ListOptions{
				PageNumber: 999,
				PageSize:   100,
			},
		})
		require.NoError(t, err)
		assert.Empty(t, pl.Items)
		assert.Equal(t, 999, pl.CurrentPage)
		assert.Equal(t, 2, pl.TotalCount)
	})

	t.Run("when policy set ID is invalid ID", func(t *testing.T) {
		pl, err := client.PolicySetParameters.List(ctx, badIdentifier, PolicySetParameterListOptions{})
		assert.Nil(t, pl)
		assert.EqualError(t, err, "invalid value for policy set ID")
	})
}

func TestPolicySetParametersCreate(t *testing.T) {
	skipIfFreeOnly(t)

	client := testClient(t)
	ctx := context.Background()

	psTest, psTestCleanup := createPolicySet(t, client, nil, nil, nil)
	defer psTestCleanup()

	t.Run("with valid options", func(t *testing.T) {
		options := PolicySetParameterCreateOptions{
			Key:      String(randomString(t)),
			Value:    String(randomString(t)),
			Category: Category(CategoryPolicySet),
		}

		p, err := client.PolicySetParameters.Create(ctx, psTest.ID, options)
		require.NoError(t, err)

		assert.NotEmpty(t, p.ID)
		assert.Equal(t, *options.Key, p.Key)
		assert.Equal(t, *options.Value, p.Value)
		assert.Equal(t, *options.Category, p.Category)
		assert.Equal(t, psTest.ID, p.PolicySet.ID)
		// The policy set isn't returned correcly by the API.
		// assert.Equal(t, *options.PolicySet, v.PolicySet)
	})

	t.Run("when options has an empty string value", func(t *testing.T) {
		options := PolicySetParameterCreateOptions{
			Key:      String(randomString(t)),
			Value:    String(""),
			Category: Category(CategoryPolicySet),
		}

		p, err := client.PolicySetParameters.Create(ctx, psTest.ID, options)
		require.NoError(t, err)

		assert.NotEmpty(t, p.ID)
		assert.Equal(t, *options.Key, p.Key)
		assert.Equal(t, *options.Value, p.Value)
		assert.Equal(t, *options.Category, p.Category)
	})

	t.Run("when options is missing value", func(t *testing.T) {
		options := PolicySetParameterCreateOptions{
			Key:      String(randomString(t)),
			Category: Category(CategoryPolicySet),
		}

		p, err := client.PolicySetParameters.Create(ctx, psTest.ID, options)
		require.NoError(t, err)

		assert.NotEmpty(t, p.ID)
		assert.Equal(t, *options.Key, p.Key)
		assert.Equal(t, "", p.Value)
		assert.Equal(t, *options.Category, p.Category)
	})

	t.Run("when options is missing key", func(t *testing.T) {
		options := PolicySetParameterCreateOptions{
			Value:    String(randomString(t)),
			Category: Category(CategoryPolicySet),
		}

		_, err := client.PolicySetParameters.Create(ctx, psTest.ID, options)
		assert.EqualError(t, err, "key is required")
	})

	t.Run("when options has an empty key", func(t *testing.T) {
		options := PolicySetParameterCreateOptions{
			Key:      String(""),
			Value:    String(randomString(t)),
			Category: Category(CategoryPolicySet),
		}

		_, err := client.PolicySetParameters.Create(ctx, psTest.ID, options)
		assert.EqualError(t, err, "key is required")
	})

	t.Run("when options is missing category", func(t *testing.T) {
		options := PolicySetParameterCreateOptions{
			Key:   String(randomString(t)),
			Value: String(randomString(t)),
		}

		_, err := client.PolicySetParameters.Create(ctx, psTest.ID, options)
		assert.EqualError(t, err, "category is required")
	})

	t.Run("when policy set ID is invalid", func(t *testing.T) {
		options := PolicySetParameterCreateOptions{
			Key:      String(randomString(t)),
			Value:    String(randomString(t)),
			Category: Category(CategoryPolicySet),
		}

		_, err := client.PolicySetParameters.Create(ctx, badIdentifier, options)
		assert.EqualError(t, err, "invalid value for policy set ID")
	})
}

func TestPolicySetParametersRead(t *testing.T) {
	skipIfFreeOnly(t)

	client := testClient(t)
	ctx := context.Background()

	pTest, pTestCleanup := createPolicySetParameter(t, client, nil)
	defer pTestCleanup()

	t.Run("when the parameter exists", func(t *testing.T) {
		p, err := client.PolicySetParameters.Read(ctx, pTest.PolicySet.ID, pTest.ID)
		require.NoError(t, err)
		assert.Equal(t, pTest.ID, p.ID)
		assert.Equal(t, pTest.Category, p.Category)
		assert.Equal(t, pTest.Key, p.Key)
		assert.Equal(t, pTest.Sensitive, p.Sensitive)
		assert.Equal(t, pTest.Value, p.Value)
		assert.Equal(t, pTest.PolicySet.ID, p.PolicySet.ID)
	})

	t.Run("when the parameter does not exist", func(t *testing.T) {
		p, err := client.PolicySetParameters.Read(ctx, pTest.PolicySet.ID, "nonexisting")
		assert.Nil(t, p)
		assert.Equal(t, ErrResourceNotFound, err)
	})

	t.Run("without a valid policy set ID", func(t *testing.T) {
		p, err := client.PolicySetParameters.Read(ctx, badIdentifier, pTest.ID)
		assert.Nil(t, p)
		assert.EqualError(t, err, "invalid value for policy set ID")
	})

	t.Run("without a valid parameter ID", func(t *testing.T) {
		p, err := client.PolicySetParameters.Read(ctx, pTest.PolicySet.ID, badIdentifier)
		assert.Nil(t, p)
		assert.EqualError(t, err, "invalid value for parameter ID")
	})
}

func TestPolicySetParametersUpdate(t *testing.T) {
	skipIfFreeOnly(t)

	client := testClient(t)
	ctx := context.Background()

	pTest, pTestCleanup := createPolicySetParameter(t, client, nil)
	defer pTestCleanup()

	t.Run("with valid options", func(t *testing.T) {
		options := PolicySetParameterUpdateOptions{
			Key:   String("newname"),
			Value: String("newvalue"),
		}

		p, err := client.PolicySetParameters.Update(ctx, pTest.PolicySet.ID, pTest.ID, options)
		require.NoError(t, err)

		assert.Equal(t, *options.Key, p.Key)
		assert.Equal(t, *options.Value, p.Value)
	})

	t.Run("when updating a subset of values", func(t *testing.T) {
		options := PolicySetParameterUpdateOptions{
			Key: String("someothername"),
		}

		p, err := client.PolicySetParameters.Update(ctx, pTest.PolicySet.ID, pTest.ID, options)
		require.NoError(t, err)

		assert.Equal(t, *options.Key, p.Key)
	})

	t.Run("with sensitive set", func(t *testing.T) {
		options := PolicySetParameterUpdateOptions{
			Sensitive: Bool(true),
		}

		p, err := client.PolicySetParameters.Update(ctx, pTest.PolicySet.ID, pTest.ID, options)
		require.NoError(t, err)

		assert.Equal(t, *options.Sensitive, p.Sensitive)
		assert.Empty(t, p.Value) // Because its now sensitive
	})

	t.Run("without any changes", func(t *testing.T) {
		pTest, pTestCleanup := createPolicySetParameter(t, client, nil)
		defer pTestCleanup()

		p, err := client.PolicySetParameters.Update(ctx, pTest.PolicySet.ID, pTest.ID, PolicySetParameterUpdateOptions{})
		require.NoError(t, err)

		assert.Equal(t, pTest, p)
	})

	t.Run("with invalid parameter ID", func(t *testing.T) {
		_, err := client.PolicySetParameters.Update(ctx, badIdentifier, pTest.ID, PolicySetParameterUpdateOptions{})
		assert.EqualError(t, err, "invalid value for policy set ID")
	})

	t.Run("with invalid parameter ID", func(t *testing.T) {
		_, err := client.PolicySetParameters.Update(ctx, pTest.PolicySet.ID, badIdentifier, PolicySetParameterUpdateOptions{})
		assert.EqualError(t, err, "invalid value for parameter ID")
	})
}

func TestPolicySetParametersDelete(t *testing.T) {
	skipIfFreeOnly(t)

	client := testClient(t)
	ctx := context.Background()

	psTest, psTestCleanup := createPolicySet(t, client, nil, nil, nil)
	defer psTestCleanup()

	pTest, _ := createPolicySetParameter(t, client, psTest)

	t.Run("with valid options", func(t *testing.T) {
		err := client.PolicySetParameters.Delete(ctx, psTest.ID, pTest.ID)
		assert.NoError(t, err)
	})

	t.Run("with non existing parameter ID", func(t *testing.T) {
		err := client.PolicySetParameters.Delete(ctx, psTest.ID, "nonexisting")
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("with invalid policy set ID", func(t *testing.T) {
		err := client.PolicySetParameters.Delete(ctx, badIdentifier, pTest.ID)
		assert.EqualError(t, err, "invalid value for policy set ID")
	})

	t.Run("with invalid parameter ID", func(t *testing.T) {
		err := client.PolicySetParameters.Delete(ctx, psTest.ID, badIdentifier)
		assert.EqualError(t, err, "invalid value for parameter ID")
	})
}
