// Copyright IBM Corp. 2018, 2026
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdminSCIMGroups_List(t *testing.T) {
	skipUnlessEnterprise(t)
	client := testClient(t)
	ctx := context.Background()

	scimClient := client.Admin.Settings.SCIM

	enableSCIM(ctx, t, client, true)
	defer enableSCIM(ctx, t, client, false)

	scimToken, err := scimClient.Tokens.Create(ctx, "integration-test-token")
	require.NoError(t, err)

	t.Run("basic list operations", func(t *testing.T) {
		t.Run("empty list immediately after enabling SCIM", func(t *testing.T) {
			scimGroups, err := scimClient.Groups.List(ctx, nil)
			require.NoError(t, err)
			assert.Len(t, scimGroups.Items, 0)
			assert.Equal(t, 0, scimGroups.TotalCount)
		})

		t.Run("list all created groups", func(t *testing.T) {
			var groupIDs []string
			var expectedGroups []AdminSCIMGroup
			t.Cleanup(func() {
				for _, id := range groupIDs {
					deleteSCIMGroup(ctx, t, client, id, scimToken.Token)
				}
			})

			for range 2 {
				groupName := randomStringWithoutSpecialChar(t)
				id := createSCIMGroup(ctx, t, client, groupName, scimToken.Token)
				groupIDs = append(groupIDs, id)
				expectedGroups = append(expectedGroups, AdminSCIMGroup{ID: id, Name: groupName})
			}

			scimGroups, err := scimClient.Groups.List(ctx, nil)
			require.NoError(t, err)
			assert.Len(t, scimGroups.Items, 2)
			assert.Equal(t, 2, scimGroups.TotalCount)

			var found int
			for _, eg := range expectedGroups {
				for _, g := range scimGroups.Items {
					if g.ID == eg.ID {
						assert.Equal(t, eg.Name, g.Name)
						found++
						break
					}
				}
			}
			assert.Equal(t, 2, found, "all created groups should have matched ID and Name")
		})
	})

	t.Run("filter groups using search query", func(t *testing.T) {
		var groupIDs []string
		t.Cleanup(func() {
			for _, id := range groupIDs {
				deleteSCIMGroup(ctx, t, client, id, scimToken.Token)
			}
		})

		prefix := randomStringWithoutSpecialChar(t) + "-"
		// Create a cohesive set of 10 groups that satisfy all query scenarios
		groupIDs = append(groupIDs,
			createSCIMGroup(ctx, t, client, prefix+"this-group-exists", scimToken.Token),
			createSCIMGroup(ctx, t, client, prefix+"matching-group-1", scimToken.Token),
			createSCIMGroup(ctx, t, client, prefix+"matching-group-2", scimToken.Token),
			createSCIMGroup(ctx, t, client, prefix+"matching-group-3", scimToken.Token),
			createSCIMGroup(ctx, t, client, prefix+"CaSe-InSeNsItIvE-gRoUp", scimToken.Token),
		)
		for range 5 {
			id := createSCIMGroup(ctx, t, client, prefix+"random-"+randomStringWithoutSpecialChar(t), scimToken.Token)
			groupIDs = append(groupIDs, id)
		}

		testCases := []struct {
			name               string
			options            AdminSCIMGroupListOptions
			expectedGroupCount int
		}{
			{
				name:               "query returns no results for non-existent prefix",
				options:            AdminSCIMGroupListOptions{Query: prefix + "this-group-doesnot-exist"},
				expectedGroupCount: 0,
			},
			{
				name:               "query returns exact match for specific group",
				options:            AdminSCIMGroupListOptions{Query: prefix + "this-group-exists"},
				expectedGroupCount: 1,
			},
			{
				name:               "query returns multiple groups matching prefix",
				options:            AdminSCIMGroupListOptions{Query: prefix + "matching-group-"},
				expectedGroupCount: 3,
			},
			{
				name:               "query performs case-insensitive match",
				options:            AdminSCIMGroupListOptions{Query: prefix + "case-insensitive-group"},
				expectedGroupCount: 1,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				scimGroups, err := scimClient.Groups.List(ctx, &tc.options)
				require.NoError(t, err)
				assert.Len(t, scimGroups.Items, tc.expectedGroupCount)
				assert.Equal(t, tc.expectedGroupCount, scimGroups.TotalCount)
			})
		}
	})

	t.Run("paginate through groups", func(t *testing.T) {
		var groupIDs []string
		t.Cleanup(func() {
			for _, id := range groupIDs {
				deleteSCIMGroup(ctx, t, client, id, scimToken.Token)
			}
		})

		prefix := randomStringWithoutSpecialChar(t) + "-"
		// Create 30 groups to test default page size of 20
		for range 30 {
			groupName := prefix + randomStringWithoutSpecialChar(t)
			id := createSCIMGroup(ctx, t, client, groupName, scimToken.Token)
			groupIDs = append(groupIDs, id)
		}

		testCases := []struct {
			name               string
			options            AdminSCIMGroupListOptions
			excludeOptions     *AdminSCIMGroupListOptions
			expectedGroupCount int
			expectedTotalCount int
			expectedTotalPages int
			expectedPage       int
			expectedNextPage   int
			expectedPrevPage   int
		}{
			{
				name:               "default page size (20) returns first page",
				options:            AdminSCIMGroupListOptions{Query: prefix, ListOptions: ListOptions{PageNumber: 1}},
				expectedGroupCount: 20,
				expectedTotalCount: 30,
				expectedTotalPages: 2,
				expectedPage:       1,
				expectedNextPage:   2,
				expectedPrevPage:   0,
			},
			{
				name:               "default page size (20) returns second page",
				options:            AdminSCIMGroupListOptions{Query: prefix, ListOptions: ListOptions{PageNumber: 2}},
				excludeOptions:     &AdminSCIMGroupListOptions{Query: prefix, ListOptions: ListOptions{PageNumber: 1}},
				expectedGroupCount: 10,
				expectedTotalCount: 30,
				expectedTotalPages: 2,
				expectedPage:       2,
				expectedNextPage:   0,
				expectedPrevPage:   1,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				scimGroups, err := scimClient.Groups.List(ctx, &tc.options)
				require.NoError(t, err)
				assert.Len(t, scimGroups.Items, tc.expectedGroupCount)
				assert.Equal(t, tc.expectedTotalCount, scimGroups.TotalCount)
				assert.Equal(t, tc.expectedTotalPages, scimGroups.TotalPages)
				assert.Equal(t, tc.expectedPage, scimGroups.CurrentPage)
				assert.Equal(t, tc.expectedNextPage, scimGroups.NextPage)
				assert.Equal(t, tc.expectedPrevPage, scimGroups.PreviousPage)

				// Verify mutually exclusive items
				if tc.excludeOptions != nil {
					excludedGroups, err := scimClient.Groups.List(ctx, tc.excludeOptions)
					require.NoError(t, err)

					for _, g := range scimGroups.Items {
						for _, exGroup := range excludedGroups.Items {
							assert.NotEqual(t, g.ID, exGroup.ID)
						}
					}
				}
			})
		}
	})

	t.Run("combine query filtering and pagination", func(t *testing.T) {
		var groupIDs []string
		t.Cleanup(func() {
			for _, id := range groupIDs {
				deleteSCIMGroup(ctx, t, client, id, scimToken.Token)
			}
		})

		prefix := randomStringWithoutSpecialChar(t)

		// Create 4 random groups
		for range 4 {
			groupName := prefix + "-" + randomStringWithoutSpecialChar(t)
			id := createSCIMGroup(ctx, t, client, groupName, scimToken.Token)
			groupIDs = append(groupIDs, id)
		}

		// Create 6 matching groups with same suffix "-idp-group"
		for range 6 {
			groupName := fmt.Sprintf("%s-idp-group-%s", prefix, randomStringWithoutSpecialChar(t))
			id := createSCIMGroup(ctx, t, client, groupName, scimToken.Token)
			groupIDs = append(groupIDs, id)
		}

		testCases := []struct {
			name               string
			options            AdminSCIMGroupListOptions
			excludeOptions     *AdminSCIMGroupListOptions
			expectedGroupCount int
			expectedTotalCount int
			expectedTotalPages int
			expectedPage       int
		}{
			{
				name: "first page of filtered results",
				options: AdminSCIMGroupListOptions{
					Query:       prefix + "-idp-group",
					ListOptions: ListOptions{PageSize: 3, PageNumber: 1},
				},
				expectedGroupCount: 3,
				expectedTotalCount: 6,
				expectedTotalPages: 2,
				expectedPage:       1,
			},
			{
				name: "second page of filtered results",
				options: AdminSCIMGroupListOptions{
					Query:       prefix + "-idp-group",
					ListOptions: ListOptions{PageSize: 3, PageNumber: 2},
				},
				excludeOptions: &AdminSCIMGroupListOptions{
					Query:       prefix + "-idp-group",
					ListOptions: ListOptions{PageSize: 3, PageNumber: 1},
				},
				expectedGroupCount: 3,
				expectedTotalCount: 6,
				expectedTotalPages: 2,
				expectedPage:       2,
			},
			{
				name: "out of bounds page of filtered results returns empty list",
				options: AdminSCIMGroupListOptions{
					Query:       prefix + "-idp-group",
					ListOptions: ListOptions{PageSize: 3, PageNumber: 3},
				},
				expectedGroupCount: 0,
				expectedTotalCount: 6,
				expectedTotalPages: 2,
				expectedPage:       3,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				scimGroups, err := scimClient.Groups.List(ctx, &tc.options)
				require.NoError(t, err)
				assert.Len(t, scimGroups.Items, tc.expectedGroupCount)
				assert.Equal(t, tc.expectedTotalCount, scimGroups.TotalCount)
				assert.Equal(t, tc.expectedTotalPages, scimGroups.TotalPages)
				assert.Equal(t, tc.expectedPage, scimGroups.CurrentPage)

				// Verify mutually exclusive items
				if tc.excludeOptions != nil {
					excludedGroups, err := scimClient.Groups.List(ctx, tc.excludeOptions)
					require.NoError(t, err)

					for _, g := range scimGroups.Items {
						for _, exGroup := range excludedGroups.Items {
							assert.NotEqual(t, g.ID, exGroup.ID)
						}
					}
				}
			})
		}
	})
}
