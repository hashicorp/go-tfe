package tfe

// func TestOAuthTokensList(t *testing.T) {
// 	client := testClient(t)
//	ctx := context.Background()

// 	orgTest, orgwTestCleanup := createOrganization(t, client)
// 	defer orgwTestCleanup()

// 	otTest1, _ := createOAuthToken(t, client, orgTest)
// 	otTest2, _ := createOAuthToken(t, client, orgTest)

// 	t.Run("with valid options", func(t *testing.T) {
// 		ots, err := client.OAuthTokens.List(ctx, orgTest.Name)
// 		require.NoError(t, err)

// 		assert.Contains(t, ots, otTest1)
// 		assert.Contains(t, ots, otTest2)

// 		t.Run("the OAuth client relationship is decoded correcly", func(t *testing.T) {
// 			for _, ot := range ots {
// 				assert.NotEmpty(t, ot.OAuthClient)
// 			}
// 		})
// 	})

// 	t.Run("without a valid organization", func(t *testing.T) {
// 		ots, err := client.OAuthTokens.List(ctx, badIdentifier)
// 		assert.Nil(t, ots)
// 		assert.EqualError(t, err, "Invalid value for organization")
// 	})
// }
