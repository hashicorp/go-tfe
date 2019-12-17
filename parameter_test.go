package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParametersList(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	psTest, _ := createPolicySet(t, client, orgTest, nil, nil)

	vTest1, _ := createParameter(t, client, psTest)
	vTest2, _ := createParameter(t, client, psTest)

	t.Run("without list options", func(t *testing.T) {
		vl, err := client.Parameters.List(ctx, psTest.ID, ParameterListOptions{})
		require.NoError(t, err)
		assert.Contains(t, vl.Items, vTest1)
		assert.Contains(t, vl.Items, vTest2)

		t.Skip("paging not supported yet in API")
		assert.Equal(t, 1, vl.CurrentPage)
		assert.Equal(t, 2, vl.TotalCount)
	})

	t.Run("with list options", func(t *testing.T) {
		t.Skip("paging not supported yet in API")
		// Request a page number which is out of range. The result should
		// be successful, but return no results if the paging options are
		// properly passed along.
		vl, err := client.Parameters.List(ctx, psTest.ID, ParameterListOptions{
			ListOptions: ListOptions{
				PageNumber: 999,
				PageSize:   100,
			},
		})
		require.NoError(t, err)
		assert.Empty(t, vl.Items)
		assert.Equal(t, 999, vl.CurrentPage)
		assert.Equal(t, 2, vl.TotalCount)
	})

	t.Run("when policy set ID is invalid ID", func(t *testing.T) {
		vl, err := client.Parameters.List(ctx, badIdentifier, ParameterListOptions{})
		assert.Nil(t, vl)
		assert.EqualError(t, err, "invalid value for policy set ID")
	})
}

func TestParametersCreate(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	psTest, psTestCleanup := createPolicySet(t, client, nil, nil, nil)
	defer psTestCleanup()

	t.Run("with valid options", func(t *testing.T) {
		options := ParameterCreateOptions{
			Key:      String(randomString(t)),
			Value:    String(randomString(t)),
			Category: Category(CategoryPolicySet),
		}

		v, err := client.Parameters.Create(ctx, psTest.ID, options)
		require.NoError(t, err)

		assert.NotEmpty(t, v.ID)
		assert.Equal(t, *options.Key, v.Key)
		assert.Equal(t, *options.Value, v.Value)
		assert.Equal(t, *options.Category, v.Category)
		// The policy set isn't returned correcly by the API.
		// assert.Equal(t, *options.PolicySet, v.PolicySet)
	})

	t.Run("when options has an empty string value", func(t *testing.T) {
		options := ParameterCreateOptions{
			Key:      String(randomString(t)),
			Value:    String(""),
			Category: Category(CategoryPolicySet),
		}

		v, err := client.Parameters.Create(ctx, psTest.ID, options)
		require.NoError(t, err)

		assert.NotEmpty(t, v.ID)
		assert.Equal(t, *options.Key, v.Key)
		assert.Equal(t, *options.Value, v.Value)
		assert.Equal(t, *options.Category, v.Category)
	})

	t.Run("when options is missing value", func(t *testing.T) {
		options := ParameterCreateOptions{
			Key:      String(randomString(t)),
			Category: Category(CategoryPolicySet),
		}

		v, err := client.Parameters.Create(ctx, psTest.ID, options)
		require.NoError(t, err)

		assert.NotEmpty(t, v.ID)
		assert.Equal(t, *options.Key, v.Key)
		assert.Equal(t, "", v.Value)
		assert.Equal(t, *options.Category, v.Category)
	})

	t.Run("when options is missing key", func(t *testing.T) {
		options := ParameterCreateOptions{
			Value:    String(randomString(t)),
			Category: Category(CategoryPolicySet),
		}

		_, err := client.Parameters.Create(ctx, psTest.ID, options)
		assert.EqualError(t, err, "key is required")
	})

	t.Run("when options has an empty key", func(t *testing.T) {
		options := ParameterCreateOptions{
			Key:      String(""),
			Value:    String(randomString(t)),
			Category: Category(CategoryPolicySet),
		}

		_, err := client.Parameters.Create(ctx, psTest.ID, options)
		assert.EqualError(t, err, "key is required")
	})

	t.Run("when options is missing category", func(t *testing.T) {
		options := ParameterCreateOptions{
			Key:   String(randomString(t)),
			Value: String(randomString(t)),
		}

		_, err := client.Parameters.Create(ctx, psTest.ID, options)
		assert.EqualError(t, err, "category is required")
	})

	t.Run("when policy set ID is invalid", func(t *testing.T) {
		options := ParameterCreateOptions{
			Key:      String(randomString(t)),
			Value:    String(randomString(t)),
			Category: Category(CategoryPolicySet),
		}

		_, err := client.Parameters.Create(ctx, badIdentifier, options)
		assert.EqualError(t, err, "invalid value for policy set ID")
	})
}

func TestParametersRead(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	vTest, vTestCleanup := createParameter(t, client, nil)
	defer vTestCleanup()

	t.Run("when the parameter exists", func(t *testing.T) {
		v, err := client.Parameters.Read(ctx, vTest.PolicySet.ID, vTest.ID)
		require.NoError(t, err)
		assert.Equal(t, vTest.ID, v.ID)
		assert.Equal(t, vTest.Category, v.Category)
		assert.Equal(t, vTest.Key, v.Key)
		assert.Equal(t, vTest.Sensitive, v.Sensitive)
		assert.Equal(t, vTest.Value, v.Value)
	})

	t.Run("when the parameter does not exist", func(t *testing.T) {
		v, err := client.Parameters.Read(ctx, vTest.PolicySet.ID, "nonexisting")
		assert.Nil(t, v)
		assert.Equal(t, ErrResourceNotFound, err)
	})

	t.Run("without a valid policy set ID", func(t *testing.T) {
		v, err := client.Parameters.Read(ctx, badIdentifier, vTest.ID)
		assert.Nil(t, v)
		assert.EqualError(t, err, "invalid value for policy set ID")
	})

	t.Run("without a valid parameter ID", func(t *testing.T) {
		v, err := client.Parameters.Read(ctx, vTest.PolicySet.ID, badIdentifier)
		assert.Nil(t, v)
		assert.EqualError(t, err, "invalid value for parameter ID")
	})
}

func TestParametersUpdate(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	vTest, vTestCleanup := createParameter(t, client, nil)
	defer vTestCleanup()

	t.Run("with valid options", func(t *testing.T) {
		options := ParameterUpdateOptions{
			Key:   String("newname"),
			Value: String("newvalue"),
		}

		v, err := client.Parameters.Update(ctx, vTest.PolicySet.ID, vTest.ID, options)
		require.NoError(t, err)

		assert.Equal(t, *options.Key, v.Key)
		assert.Equal(t, *options.Value, v.Value)
	})

	t.Run("when updating a subset of values", func(t *testing.T) {
		options := ParameterUpdateOptions{
			Key: String("someothername"),
		}

		v, err := client.Parameters.Update(ctx, vTest.PolicySet.ID, vTest.ID, options)
		require.NoError(t, err)

		assert.Equal(t, *options.Key, v.Key)
	})

	t.Run("with sensitive set", func(t *testing.T) {
		options := ParameterUpdateOptions{
			Sensitive: Bool(true),
		}

		v, err := client.Parameters.Update(ctx, vTest.PolicySet.ID, vTest.ID, options)
		require.NoError(t, err)

		assert.Equal(t, *options.Sensitive, v.Sensitive)
		assert.Empty(t, v.Value) // Because its now sensitive
	})

	t.Run("without any changes", func(t *testing.T) {
		vTest, vTestCleanup := createParameter(t, client, nil)
		defer vTestCleanup()

		v, err := client.Parameters.Update(ctx, vTest.PolicySet.ID, vTest.ID, ParameterUpdateOptions{})
		require.NoError(t, err)

		assert.Equal(t, vTest, v)
	})

	t.Run("with invalid parameter ID", func(t *testing.T) {
		_, err := client.Parameters.Update(ctx, badIdentifier, vTest.ID, ParameterUpdateOptions{})
		assert.EqualError(t, err, "invalid value for policy set ID")
	})

	t.Run("with invalid parameter ID", func(t *testing.T) {
		_, err := client.Parameters.Update(ctx, vTest.PolicySet.ID, badIdentifier, ParameterUpdateOptions{})
		assert.EqualError(t, err, "invalid value for parameter ID")
	})
}

func TestParametersDelete(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	psTest, psTestCleanup := createPolicySet(t, client, nil, nil, nil)
	defer psTestCleanup()

	vTest, _ := createParameter(t, client, psTest)

	t.Run("with valid options", func(t *testing.T) {
		err := client.Parameters.Delete(ctx, psTest.ID, vTest.ID)
		assert.NoError(t, err)
	})

	t.Run("with non existing parameter ID", func(t *testing.T) {
		err := client.Parameters.Delete(ctx, psTest.ID, "nonexisting")
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("with invalid policy set ID", func(t *testing.T) {
		err := client.Parameters.Delete(ctx, badIdentifier, vTest.ID)
		assert.EqualError(t, err, "invalid value for policy set ID")
	})

	t.Run("with invalid parameter ID", func(t *testing.T) {
		err := client.Parameters.Delete(ctx, psTest.ID, badIdentifier)
		assert.EqualError(t, err, "invalid value for parameter ID")
	})
}
