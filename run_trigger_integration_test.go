package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunTriggerList(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	wTest, wTestCleanup := createWorkspace(t, client, orgTest)
	defer wTestCleanup()

	sourceable1Test, sourceable1TestCleanup := createWorkspace(t, client, orgTest)
	defer sourceable1TestCleanup()

	sourceable2Test, sourceable2TestCleanup := createWorkspace(t, client, orgTest)
	defer sourceable2TestCleanup()

	rtTest1, rtTestCleanup1 := createRunTrigger(t, client, wTest, sourceable1Test)
	defer rtTestCleanup1()
	rtTest2, rtTestCleanup2 := createRunTrigger(t, client, wTest, sourceable2Test)
	defer rtTestCleanup2()

	t.Run("without list options", func(t *testing.T) {
		rtl, err := client.RunTriggers.List(
			ctx,
			wTest.ID,
			RunTriggerListOptions{
				RunTriggerType: String("inbound"),
			},
		)
		require.NoError(t, err)
		assert.Contains(t, rtl.Items, rtTest1)
		assert.Contains(t, rtl.Items, rtTest2)
		assert.Equal(t, 1, rtl.CurrentPage)
		assert.Equal(t, 2, rtl.TotalCount)
	})

	t.Run("with list options", func(t *testing.T) {
		// Request a page number which is out of range. The result should
		// be successful, but return no results if the paging options are
		// properly passed along.
		rtl, err := client.RunTriggers.List(
			ctx,
			wTest.ID,
			RunTriggerListOptions{
				ListOptions: ListOptions{
					PageNumber: 999,
					PageSize:   100,
				},
				RunTriggerType: String("inbound"),
			},
		)
		require.NoError(t, err)
		assert.Empty(t, rtl.Items)
		assert.Equal(t, 999, rtl.CurrentPage)
		assert.Equal(t, 2, rtl.TotalCount)
	})

	t.Run("without a valid workspace", func(t *testing.T) {
		rtl, err := client.RunTriggers.List(
			ctx,
			badIdentifier,
			RunTriggerListOptions{
				RunTriggerType: String("inbound"),
			},
		)
		assert.Nil(t, rtl)
		assert.EqualError(t, err, ErrInvalidWorkspaceID.Error())
	})

	t.Run("without run-trigger type", func(t *testing.T) {
		rtl, err := client.RunTriggers.List(
			ctx,
			wTest.ID,
			RunTriggerListOptions{},
		)
		assert.Nil(t, rtl)
		assert.EqualError(t, err, "run-trigger type is required")
	})

	t.Run("with invalid run-trigger type", func(t *testing.T) {
		rtl, err := client.RunTriggers.List(
			ctx,
			wTest.ID,
			RunTriggerListOptions{
				RunTriggerType: String("invalid"),
			},
		)
		assert.Nil(t, rtl)
		assert.EqualError(t, err, "invalid value for run-trigger type")
	})
}

func TestRunTriggerCreate(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	wTest, wTestCleanup := createWorkspace(t, client, orgTest)
	defer wTestCleanup()

	sourceableTest, sourceableTestCleanup := createWorkspace(t, client, orgTest)
	defer sourceableTestCleanup()

	t.Run("with all required values", func(t *testing.T) {
		options := RunTriggerCreateOptions{
			Sourceable: sourceableTest,
		}

		_, err := client.RunTriggers.Create(ctx, wTest.ID, options)
		require.NoError(t, err)
	})

	t.Run("without a required value", func(t *testing.T) {
		options := RunTriggerCreateOptions{}

		rt, err := client.RunTriggers.Create(ctx, wTest.ID, options)
		assert.Nil(t, rt)
		assert.EqualError(t, err, "sourceable is required")
	})

	t.Run("without a valid workspace", func(t *testing.T) {
		rt, err := client.RunTriggers.Create(ctx, badIdentifier, RunTriggerCreateOptions{})
		assert.Nil(t, rt)
		assert.EqualError(t, err, ErrInvalidWorkspaceID.Error())
	})

	t.Run("when an error is returned from the api", func(t *testing.T) {
		// There are many cases that would cause the server to return an error
		// on run trigger creation. This tests one of them: setting workspace
		// and sourceable to the same workspace
		options := RunTriggerCreateOptions{
			Sourceable: sourceableTest,
		}

		rt, err := client.RunTriggers.Create(ctx, sourceableTest.ID, options)
		assert.Nil(t, rt)
		assert.Error(t, err)
	})
}

func TestRunTriggerRead(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	wTest, wTestCleanup := createWorkspace(t, client, orgTest)
	defer wTestCleanup()

	sourceableTest, sourceableTestCleanup := createWorkspace(t, client, orgTest)
	defer sourceableTestCleanup()

	rtTest, rtTestCleanup := createRunTrigger(t, client, wTest, sourceableTest)
	defer rtTestCleanup()

	t.Run("with a valid ID", func(t *testing.T) {
		rt, err := client.RunTriggers.Read(ctx, rtTest.ID)
		require.NoError(t, err)
		assert.Equal(t, rtTest.ID, rt.ID)
	})

	t.Run("when the run trigger does not exist", func(t *testing.T) {
		_, err := client.RunTriggers.Read(ctx, "nonexisting")
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("when the run trigger ID is invalid", func(t *testing.T) {
		_, err := client.RunTriggers.Read(ctx, badIdentifier)
		assert.EqualError(t, err, "invalid value for run trigger ID")
	})
}

func TestRunTriggerDelete(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	wTest, wTestCleanup := createWorkspace(t, client, orgTest)
	defer wTestCleanup()

	sourceableTest, sourceableTestCleanup := createWorkspace(t, client, orgTest)
	defer sourceableTestCleanup()

	// No need to cleanup here, as this test will delete this run trigger
	rtTest, _ := createRunTrigger(t, client, wTest, sourceableTest)

	t.Run("with a valid ID", func(t *testing.T) {
		err := client.RunTriggers.Delete(ctx, rtTest.ID)
		require.NoError(t, err)

		_, err = client.RunTriggers.Read(ctx, rtTest.ID)
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("when the run trigger does not exist", func(t *testing.T) {
		err := client.RunTriggers.Delete(ctx, "nonexisting")
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("when the run trigger ID is invalid", func(t *testing.T) {
		err := client.RunTriggers.Delete(ctx, badIdentifier)
		assert.EqualError(t, err, "invalid value for run trigger ID")
	})
}
