// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommentsList(t *testing.T) {
	client := testClient(t)
	acquireRunMutex(t, client)

	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	wTest1, wTest1Cleanup := createWorkspace(t, client, orgTest)
	defer wTest1Cleanup()

	rTest, rTest1Cleanup := createRunApply(t, client, wTest1)
	defer rTest1Cleanup()
	commentBody1 := "1st comment test"
	commentBody2 := "2nd comment test"

	t.Run("without comments", func(t *testing.T) {
		_, err := client.Comments.List(ctx, rTest.ID)
		require.NoError(t, err)
	})

	t.Run("without a valid run", func(t *testing.T) {
		cl, err := client.Comments.List(ctx, badIdentifier)
		assert.Nil(t, cl)
		assert.EqualError(t, err, ErrInvalidRunID.Error())
	})

	t.Run("create a comment", func(t *testing.T) {
		options := CommentCreateOptions{
			Body: commentBody1,
		}
		cl, err := client.Comments.Create(ctx, rTest.ID, options)
		require.NoError(t, err)
		assert.Equal(t, commentBody1, cl.Body)
	})

	t.Run("create 2nd comment", func(t *testing.T) {
		options := CommentCreateOptions{
			Body: commentBody2,
		}
		cl, err := client.Comments.Create(ctx, rTest.ID, options)
		require.NoError(t, err)
		assert.Equal(t, commentBody2, cl.Body)
	})

	t.Run("list comments", func(t *testing.T) {
		commentsList, err := client.Comments.List(ctx, rTest.ID)
		require.NoError(t, err)
		assert.Len(t, commentsList.Items, 2)
		assert.Equal(t, true, commentItemsContainsBody(commentsList.Items, commentBody1))
		assert.Equal(t, true, commentItemsContainsBody(commentsList.Items, commentBody2))
	})
}

func commentItemsContainsBody(items []*Comment, body string) bool {
	hasBody := false
	for _, item := range items {
		if item.Body == body {
			hasBody = true
			break
		}
	}

	return hasBody
}
