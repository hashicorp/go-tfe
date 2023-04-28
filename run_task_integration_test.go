// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunTasksCreate(t *testing.T) {
	// t.Skip("skipping run task integration tests until service migration is complete.")
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	upgradeOrganizationSubscription(t, client, orgTest)

	runTaskServerURL := os.Getenv("TFC_RUN_TASK_URL")
	if runTaskServerURL == "" {
		t.Error("Cannot create a run task with an empty URL. You must set TFC_RUN_TASK_URL for run task related tests.")
	}

	runTaskName := "tst-runtask-" + randomString(t)
	runTaskDescription := "A Run Task Description"

	t.Run("add run task to organization", func(t *testing.T) {
		r, err := client.RunTasks.Create(ctx, orgTest.Name, RunTaskCreateOptions{
			Name:        runTaskName,
			URL:         runTaskServerURL,
			Description: &runTaskDescription,
			Category:    "task",
			Enabled:     Bool(true),
		})
		require.NoError(t, err)

		assert.NotEmpty(t, r.ID)
		assert.Equal(t, r.Name, runTaskName)
		assert.Equal(t, r.URL, runTaskServerURL)
		assert.Equal(t, r.Category, "task")
		assert.Equal(t, r.Description, runTaskDescription)

		t.Run("ensure org is deserialized properly", func(t *testing.T) {
			assert.Equal(t, r.Organization.Name, orgTest.Name)
		})
	})
}

func TestRunTasksList(t *testing.T) {
	t.Skip("skipping run task integration tests until service migration is complete.")
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	_, runTaskTest1Cleanup := createRunTask(t, client, orgTest)
	defer runTaskTest1Cleanup()

	_, runTaskTest2Cleanup := createRunTask(t, client, orgTest)
	defer runTaskTest2Cleanup()

	t.Run("with no params", func(t *testing.T) {
		runTaskList, err := client.RunTasks.List(ctx, orgTest.Name, nil)
		require.NoError(t, err)
		assert.NotNil(t, runTaskList.Items)
		assert.NotEmpty(t, runTaskList.Items[0].ID)
		assert.NotEmpty(t, runTaskList.Items[0].URL)
		assert.NotEmpty(t, runTaskList.Items[1].ID)
		assert.NotEmpty(t, runTaskList.Items[1].URL)
	})
}

func TestRunTasksRead(t *testing.T) {
	t.Skip("skipping run task integration tests until service migration is complete.")
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	runTaskTest, runTaskTestCleanup := createRunTask(t, client, orgTest)
	defer runTaskTestCleanup()

	t.Run("by ID", func(t *testing.T) {
		r, err := client.RunTasks.Read(ctx, runTaskTest.ID)
		require.NoError(t, err)

		assert.Equal(t, runTaskTest.ID, r.ID)
		assert.Equal(t, runTaskTest.URL, r.URL)
		assert.Equal(t, runTaskTest.Category, r.Category)
		assert.Equal(t, runTaskTest.Description, r.Description)
		assert.Equal(t, runTaskTest.HMACKey, r.HMACKey)
		assert.Equal(t, runTaskTest.Enabled, r.Enabled)
	})

	t.Run("with options", func(t *testing.T) {
		wkTest1, wkTest1Cleanup := createWorkspace(t, client, orgTest)
		defer wkTest1Cleanup()

		wkTest2, wkTest2Cleanup := createWorkspace(t, client, orgTest)
		defer wkTest2Cleanup()

		_, wrTest1Cleanup := createWorkspaceRunTask(t, client, wkTest1, runTaskTest)
		defer wrTest1Cleanup()

		_, wrTest2Cleanup := createWorkspaceRunTask(t, client, wkTest2, runTaskTest)
		defer wrTest2Cleanup()

		r, err := client.RunTasks.ReadWithOptions(ctx, runTaskTest.ID, &RunTaskReadOptions{
			Include: []RunTaskIncludeOpt{RunTaskWorkspaceTasks},
		})

		require.NoError(t, err)

		require.NotEmpty(t, r.WorkspaceRunTasks)
		assert.NotEmpty(t, r.WorkspaceRunTasks[0].ID)
		assert.NotEmpty(t, r.WorkspaceRunTasks[0].EnforcementLevel)
		assert.NotEmpty(t, r.WorkspaceRunTasks[1].ID)
		assert.NotEmpty(t, r.WorkspaceRunTasks[1].EnforcementLevel)
	})
}

func TestRunTasksUpdate(t *testing.T) {
	t.Skip("skipping run task integration tests until service migration is complete.")
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	runTaskTest, runTaskTestCleanup := createRunTask(t, client, orgTest)
	defer runTaskTestCleanup()

	t.Run("rename task", func(t *testing.T) {
		rename := runTaskTest.Name + "-UPDATED"
		r, err := client.RunTasks.Update(ctx, runTaskTest.ID, RunTaskUpdateOptions{
			Name: &rename,
		})
		require.NoError(t, err)

		r, err = client.RunTasks.Read(ctx, r.ID)
		require.NoError(t, err)

		assert.Equal(t, rename, r.Name)
	})

	t.Run("toggle enabled", func(t *testing.T) {
		runTaskTest.Enabled = !runTaskTest.Enabled
		r, err := client.RunTasks.Update(ctx, runTaskTest.ID, RunTaskUpdateOptions{
			Enabled: &runTaskTest.Enabled,
		})
		require.NoError(t, err)

		r, err = client.RunTasks.Read(ctx, r.ID)
		require.NoError(t, err)

		assert.Equal(t, runTaskTest.Enabled, r.Enabled)
	})

	t.Run("update description", func(t *testing.T) {
		newDescription := "An updated task description"
		r, err := client.RunTasks.Update(ctx, runTaskTest.ID, RunTaskUpdateOptions{
			Description: &newDescription,
		})
		require.NoError(t, err)

		r, err = client.RunTasks.Read(ctx, r.ID)
		require.NoError(t, err)

		assert.Equal(t, newDescription, r.Description)
	})
}

func TestRunTasksDelete(t *testing.T) {
	t.Skip("skipping run task integration tests until service migration is complete.")
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	runTaskTest, _ := createRunTask(t, client, orgTest)

	t.Run("with valid options", func(t *testing.T) {
		err := client.RunTasks.Delete(ctx, runTaskTest.ID)
		require.NoError(t, err)

		_, err = client.RunTasks.Read(ctx, runTaskTest.ID)
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("when the run task does not exist", func(t *testing.T) {
		err := client.RunTasks.Delete(ctx, runTaskTest.ID)
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("when the run task ID is invalid", func(t *testing.T) {
		err := client.RunTasks.Delete(ctx, badIdentifier)
		assert.EqualError(t, err, ErrInvalidRunTaskID.Error())
	})
}

func TestRunTasksAttachToWorkspace(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	runTaskTest, runTaskTestCleanup := createRunTask(t, client, orgTest)
	defer runTaskTestCleanup()

	wkspaceTest, wkspaceTestCleanup := createWorkspace(t, client, orgTest)
	defer wkspaceTestCleanup()

	t.Run("to a valid workspace", func(t *testing.T) {
		wr, err := client.RunTasks.AttachToWorkspace(ctx, wkspaceTest.ID, runTaskTest.ID, Advisory)

		defer func() {
			err = client.WorkspaceRunTasks.Delete(ctx, wkspaceTest.ID, wr.ID)
			require.NoError(t, err)
		}()

		require.NoError(t, err)
		require.NotNil(t, wr.ID)
	})
}
