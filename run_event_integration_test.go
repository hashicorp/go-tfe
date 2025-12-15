// Copyright IBM Corp. 2018, 2025
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunEventsList_RunDependent(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	wTest, _ := createWorkspace(t, client, orgTest)
	rTest, _ := createRun(t, client, wTest)
	commentText := "Test comment"
	_, err := client.Comments.Create(ctx, rTest.ID, CommentCreateOptions{
		Body: commentText,
	})
	require.NoError(t, err)

	t.Run("without list options", func(t *testing.T) {
		rl, err := client.RunEvents.List(ctx, rTest.ID, nil)
		require.NoError(t, err)

		require.NotEmpty(t, rl.Items)
		// Find the comment that was added
		var commentEvent *RunEvent = nil
		for _, event := range rl.Items {
			if event.Action == "commented" {
				commentEvent = event
			}
		}
		assert.NotNil(t, commentEvent)
		// We didn't include any resources so these should be empty
		assert.Empty(t, commentEvent.Actor.Username)
		assert.Empty(t, commentEvent.Comment.Body)
	})

	t.Run("with all includes", func(t *testing.T) {
		rl, err := client.RunEvents.List(ctx, rTest.ID, &RunEventListOptions{
			Include: []RunEventIncludeOpt{RunEventActor, RunEventComment},
		})
		require.NoError(t, err)

		// Find the comment that was added
		var commentEvent *RunEvent = nil
		for _, event := range rl.Items {
			if event.Action == "commented" {
				commentEvent = event
			}
		}
		require.NotNil(t, commentEvent)

		// Assert that the include resources are included
		require.NotNil(t, commentEvent.Actor)
		assert.NotEmpty(t, commentEvent.Actor.Username)
		require.NotNil(t, commentEvent.Comment)
		assert.Equal(t, commentEvent.Comment.Body, commentText)
	})

	t.Run("without a valid run ID", func(t *testing.T) {
		rl, err := client.RunEvents.List(ctx, badIdentifier, nil)
		assert.Nil(t, rl)
		assert.EqualError(t, err, ErrInvalidRunID.Error())
	})
}

func TestRunEventsRead_RunDependent(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	wTest, _ := createWorkspace(t, client, orgTest)
	rTest, _ := createRun(t, client, wTest)
	commentText := "Test comment"
	_, err := client.Comments.Create(ctx, rTest.ID, CommentCreateOptions{
		Body: commentText,
	})
	require.NoError(t, err)

	rl, err := client.RunEvents.List(ctx, rTest.ID, nil)
	require.NoError(t, err)
	// Find the comment that was added
	var commentEvent *RunEvent = nil
	for _, event := range rl.Items {
		if event.Action == "commented" {
			commentEvent = event
		}
	}
	assert.NotNil(t, commentEvent)

	t.Run("without read options", func(t *testing.T) {
		re, err := client.RunEvents.Read(ctx, commentEvent.ID)
		require.NoError(t, err)

		// We didn't include any resources so these should be empty
		assert.Empty(t, re.Actor.Username)
		assert.Empty(t, re.Comment.Body)
	})

	t.Run("with all includes", func(t *testing.T) {
		re, err := client.RunEvents.ReadWithOptions(ctx, commentEvent.ID, &RunEventReadOptions{
			Include: []RunEventIncludeOpt{RunEventActor, RunEventComment},
		})
		require.NoError(t, err)

		// Assert that the include resources are included
		require.NotNil(t, re.Actor)
		assert.NotEmpty(t, re.Actor.Username)
		require.NotNil(t, re.Comment)
		assert.Equal(t, re.Comment.Body, commentText)
	})

	t.Run("without a valid run event ID", func(t *testing.T) {
		rl, err := client.RunEvents.Read(ctx, badIdentifier)
		assert.Nil(t, rl)
		assert.EqualError(t, err, ErrInvalidRunEventID.Error())
	})
}
