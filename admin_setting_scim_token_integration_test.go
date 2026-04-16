// Copyright IBM Corp. 2018, 2026
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdminSCIMTokens_Create(t *testing.T) {
	skipUnlessEnterprise(t)
	client := testClient(t)
	ctx := context.Background()

	enableSCIM(ctx, t, client, true)
	defer enableSCIM(ctx, t, client, false)

	scimTokenClient := client.Admin.Settings.SCIM.Token

	t.Run("create token", func(t *testing.T) {
		testCases := []struct {
			name       string
			options    AdminSCIMTokenCreateOptions
			raiseError bool
		}{
			{"with no options", AdminSCIMTokenCreateOptions{}, false},
			{
				"with description",
				AdminSCIMTokenCreateOptions{
					Description: String("Test Description"),
				},
				false,
			},
			{
				"with expiration",
				AdminSCIMTokenCreateOptions{
					ExpiredAt: Ptr(time.Now().Add(30 * 24 * time.Hour)),
				},
				false,
			},
			{
				"with description and expiration",
				AdminSCIMTokenCreateOptions{
					Description: String("Test Description"),
					ExpiredAt:   Ptr(time.Now().Add(60 * 24 * time.Hour)),
				},
				false,
			},
			{
				"with expiration in 20 days - should fail",
				AdminSCIMTokenCreateOptions{
					ExpiredAt: Ptr(time.Now().Add(20 * 24 * time.Hour)),
				},
				true,
			},
			{
				"with expiration in 400 days - should fail",
				AdminSCIMTokenCreateOptions{
					ExpiredAt: Ptr(time.Now().Add(400 * 24 * time.Hour)),
				},
				true,
			},
			{
				"with expiration in 29 days",
				AdminSCIMTokenCreateOptions{
					ExpiredAt: Ptr(time.Now().Add(29*24*time.Hour + 10*time.Second)), // adding 10 sec to account for any delays in test execution
				},
				false,
			},
			{
				"with expiration in 365 days",
				AdminSCIMTokenCreateOptions{
					ExpiredAt: Ptr(time.Now().Add(365 * 24 * time.Hour)),
				},
				false,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				var scimToken *AdminSCIMToken
				var err error

				if tc.options == (AdminSCIMTokenCreateOptions{}) {
					scimToken, err = scimTokenClient.Create(ctx)
				} else {
					scimToken, err = scimTokenClient.CreateWithOptions(ctx, tc.options)
				}

				if tc.raiseError {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)
				require.NotNil(t, scimToken)
				assert.NotEmpty(t, scimToken)
				assert.NotEmpty(t, scimToken.ID)

				t.Cleanup(func() {
					err := scimTokenClient.Delete(ctx, scimToken.ID)
					assert.NoError(t, err)
				})

				if tc.options.Description != nil {
					assert.Equal(t, *tc.options.Description, scimToken.Description)
				} else {
					assert.Empty(t, scimToken.Description)
				}

				if tc.options.ExpiredAt != nil {
					assert.WithinDuration(t, *tc.options.ExpiredAt, scimToken.ExpiredAt, 10*time.Second)
				} else {
					expectedExpiredAt := scimToken.CreatedAt.Add(365 * 24 * time.Hour)
					assert.WithinDuration(t, expectedExpiredAt, scimToken.ExpiredAt, 10*time.Second)
				}
			})
		}
	})
}

func TestAdminSCIMTokens_List(t *testing.T) {
	skipUnlessEnterprise(t)
	client := testClient(t)
	ctx := context.Background()

	enableSCIM(ctx, t, client, true)
	defer enableSCIM(ctx, t, client, false)

	scimTokenClient := client.Admin.Settings.SCIM.Token

	t.Run("list tokens", func(t *testing.T) {
		// create tokens to ensure there is data to list
		var scimTokens []*AdminSCIMToken
		for i := 0; i < 3; i++ {
			scimToken, err := scimTokenClient.Create(ctx)
			require.NoError(t, err)
			t.Cleanup(func() {
				_ = scimTokenClient.Delete(ctx, scimToken.ID)
			})
			scimTokens = append(scimTokens, scimToken)
		}

		tokenList, err := scimTokenClient.List(ctx)
		require.NoError(t, err)
		require.NotNil(t, tokenList)
		assert.NotEmpty(t, tokenList.Items)

		var expectedIDs []string
		var actualIDs []string
		for _, listedToken := range tokenList.Items {
			actualIDs = append(actualIDs, listedToken.ID)
		}

		for _, token := range scimTokens {
			expectedIDs = append(expectedIDs, token.ID)
			assert.Contains(t, actualIDs, token.ID)
		}

		assert.Subset(t, actualIDs, expectedIDs)
	})
}

func TestAdminSCIMTokens_Read(t *testing.T) {
	skipUnlessEnterprise(t)
	client := testClient(t)
	ctx := context.Background()

	enableSCIM(ctx, t, client, true)
	defer enableSCIM(ctx, t, client, false)

	scimTokenClient := client.Admin.Settings.SCIM.Token

	t.Run("read token", func(t *testing.T) {
		// create a token to ensure there is data to read
		scimToken, err := scimTokenClient.CreateWithOptions(ctx, AdminSCIMTokenCreateOptions{
			Description: String("Test Desc"),
			ExpiredAt:   Ptr(time.Now().Add(60 * 24 * time.Hour)),
		})
		require.NoError(t, err)
		require.NotNil(t, scimToken)

		t.Cleanup(func() {
			err := scimTokenClient.Delete(ctx, scimToken.ID)
			require.NoError(t, err)
		})

		testCases := []struct {
			name       string
			tokenID    string
			raiseError bool
		}{
			{"with valid token ID", scimToken.ID, false},
			{"with invalid token ID", "invalid id", true},
			{"with empty token ID", "", true},
			{"with non-existent token ID", "this-does-not-exist", true},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				token, err := scimTokenClient.Read(ctx, tc.tokenID)
				if tc.raiseError {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)
				require.NotNil(t, token)
				assert.Equal(t, tc.tokenID, token.ID)

				// Verify specific field properties for the valid token
				if !tc.raiseError {
					assert.Equal(t, scimToken.Description, token.Description)
					assert.WithinDuration(t, scimToken.ExpiredAt, token.ExpiredAt, time.Second)
					assert.NotEmpty(t, scimToken.Token)
					assert.Empty(t, token.Token)
				}
			})
		}
	})
}

func TestAdminSCIMTokens_Delete(t *testing.T) {
	skipUnlessEnterprise(t)
	client := testClient(t)
	ctx := context.Background()

	enableSCIM(ctx, t, client, true)
	defer enableSCIM(ctx, t, client, false)

	scimTokenClient := client.Admin.Settings.SCIM.Token

	t.Run("delete token", func(t *testing.T) {
		// create a token to ensure there is data to delete
		scimToken, err := scimTokenClient.Create(ctx)
		require.NoError(t, err)
		require.NotNil(t, scimToken)

		testCases := []struct {
			name       string
			tokenID    string
			raiseError bool
		}{
			{"with valid token ID", scimToken.ID, false},
			{"with invalid token ID", "invalid id", true},
			{"with empty token ID", "", true},
			{"with non-existent token ID", "this-does-not-exist", true},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				err = scimTokenClient.Delete(ctx, tc.tokenID)
				if tc.raiseError {
					require.Error(t, err)
					if tc.tokenID == "this-does-not-exist" {
						assert.ErrorIs(t, err, ErrResourceNotFound)
					} else {
						assert.ErrorIs(t, err, ErrInvalidTokenID)
					}
					return
				}
				require.NoError(t, err)

				// verify deletion
				_, err = scimTokenClient.Read(ctx, tc.tokenID)
				require.Error(t, err)
			})
		}
	})
}
