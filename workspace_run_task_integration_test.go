package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkspaceRunTasksCreate(t *testing.T) {
	skipIfBeta(t)

	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	runTaskTest, runTaskTestCleanup := createRunTask(t, client, orgTest)
	defer runTaskTestCleanup()

	wkspaceTest, wkspaceTestCleanup := createWorkspace(t, client, orgTest)
	defer wkspaceTestCleanup()

	t.Run("attach run task to workspace", func(t *testing.T) {
		wr, err := client.WorkspaceRunTasks.Create(ctx, wkspaceTest.ID, WorkspaceRunTaskCreateOptions{
			EnforcementLevel: Mandatory,
			RunTask:          runTaskTest,
		})

		require.NoError(t, err)
		assert.NotEmpty(t, wr.ID)
		assert.Equal(t, wr.EnforcementLevel, Mandatory)

		t.Run("ensure run task is deserialized properly", func(t *testing.T) {
			assert.NotEmpty(t, wr.RunTask.ID)
		})
	})
}

func TestWorkspaceRunTasksList(t *testing.T) {
	skipIfBeta(t)

	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	wkspaceTest, wkspaceTestCleanup := createWorkspace(t, client, orgTest)
	defer wkspaceTestCleanup()

	runTaskTest1, runTaskTest1Cleanup := createRunTask(t, client, orgTest)
	defer runTaskTest1Cleanup()

	runTaskTest2, runTaskTest2Cleanup := createRunTask(t, client, orgTest)
	defer runTaskTest2Cleanup()

	_, wrTaskTest1Cleanup := createWorkspaceRunTask(t, client, wkspaceTest, runTaskTest1)
	defer wrTaskTest1Cleanup()

	_, wrTaskTest2Cleanup := createWorkspaceRunTask(t, client, wkspaceTest, runTaskTest2)
	defer wrTaskTest2Cleanup()

	t.Run("with no params", func(t *testing.T) {
		wrTaskList, err := client.WorkspaceRunTasks.List(ctx, wkspaceTest.ID, nil)
		require.NoError(t, err)
		assert.NotNil(t, wrTaskList.Items)
		assert.Equal(t, len(wrTaskList.Items), 2)
		assert.NotEmpty(t, wrTaskList.Items[0].ID)
		assert.NotEmpty(t, wrTaskList.Items[0].EnforcementLevel)
	})
}

func TestWorkspaceRunTasksRead(t *testing.T) {
	skipIfBeta(t)

	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	wkspaceTest, wkspaceTestCleanup := createWorkspace(t, client, orgTest)
	defer wkspaceTestCleanup()

	runTaskTest, runTaskTestCleanup := createRunTask(t, client, orgTest)
	defer runTaskTestCleanup()

	wrTaskTest, wrTaskTestCleanup := createWorkspaceRunTask(t, client, wkspaceTest, runTaskTest)
	defer wrTaskTestCleanup()

	t.Run("by ID", func(t *testing.T) {
		wr, err := client.WorkspaceRunTasks.Read(ctx, wkspaceTest.ID, wrTaskTest.ID)
		require.NoError(t, err)

		assert.Equal(t, wrTaskTest.ID, wr.ID)
		assert.Equal(t, wrTaskTest.EnforcementLevel, wr.EnforcementLevel)

		t.Run("ensure run task is deserialized", func(t *testing.T) {
			assert.Equal(t, wr.RunTask.ID, runTaskTest.ID)
		})

		t.Run("ensure workspace is deserialized", func(t *testing.T) {
			assert.Equal(t, wr.Workspace.ID, wkspaceTest.ID)
		})
	})
}

func TestWorkspaceRunTasksUpdate(t *testing.T) {
	skipIfBeta(t)

	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	wkspaceTest, wkspaceTestCleanup := createWorkspace(t, client, orgTest)
	defer wkspaceTestCleanup()

	runTaskTest, runTaskTestCleanup := createRunTask(t, client, orgTest)
	defer runTaskTestCleanup()

	wrTaskTest, wrTaskTestCleanup := createWorkspaceRunTask(t, client, wkspaceTest, runTaskTest)
	defer wrTaskTestCleanup()

	t.Run("rename task", func(t *testing.T) {
		wr, err := client.WorkspaceRunTasks.Update(ctx, wkspaceTest.ID, wrTaskTest.ID, WorkspaceRunTaskUpdateOptions{
			EnforcementLevel: Mandatory,
		})
		require.NoError(t, err)

		wr, err = client.WorkspaceRunTasks.Read(ctx, wkspaceTest.ID, wr.ID)
		require.NoError(t, err)

		assert.Equal(t, wr.EnforcementLevel, Mandatory)
	})
}

func TestWorkspaceRunTasksDelete(t *testing.T) {
	skipIfBeta(t)

	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	wkspaceTest, wkspaceTestCleanup := createWorkspace(t, client, orgTest)
	defer wkspaceTestCleanup()

	runTaskTest, runTaskTestCleanup := createRunTask(t, client, orgTest)
	defer runTaskTestCleanup()

	wrTaskTest, _ := createWorkspaceRunTask(t, client, wkspaceTest, runTaskTest)

	t.Run("with valid options", func(t *testing.T) {
		err := client.WorkspaceRunTasks.Delete(ctx, wkspaceTest.ID, wrTaskTest.ID)
		require.NoError(t, err)

		_, err = client.WorkspaceRunTasks.Read(ctx, wkspaceTest.ID, wrTaskTest.ID)
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("when the workspace run task does not exist", func(t *testing.T) {
		err := client.WorkspaceRunTasks.Delete(ctx, wkspaceTest.ID, wrTaskTest.ID)
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("when the workspace does not exist", func(t *testing.T) {
		wkspaceTestCleanup()
		err := client.WorkspaceRunTasks.Delete(ctx, wkspaceTest.ID, wrTaskTest.ID)
		assert.EqualError(t, err, ErrResourceNotFound.Error())
	})
}
