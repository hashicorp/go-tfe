package tfe

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUserTokens_List tests listing user tokens
func TestUserTokens_List(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()
	user, err := client.Users.ReadCurrent(ctx)
	if err != nil {
		t.Fatal(err)
	}

	token, cleanupFunc := createToken(t, client, user)
	defer cleanupFunc()

	t.Run("listing existing tokens", func(t *testing.T) {
		ctx := context.Background()
		tl, err := client.UserTokens.List(ctx, user.ID)
		require.NoError(t, err)
		var found bool
		for _, j := range tl.Items {
			if j.ID == token.ID {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("token (%s) not found in token list", token.ID)
		}
	})
}

// TestUserTokens_Generate tests basic creation of user tokens
func TestUserTokens_Generate(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()
	user, err := client.Users.ReadCurrent(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// collect the created tokens for revoking after the test
	var tokens []string
	defer func(t *testing.T) {
		for _, token := range tokens {
			err := client.UserTokens.Delete(ctx, token)
			if err != nil {
				t.Fatalf("Error deleting token in cleanup:%s", err)
			}
		}
	}(t)

	t.Run("create token with no description", func(t *testing.T) {
		token, err := client.UserTokens.Generate(ctx, user.ID, UserTokenGenerateOptions{})
		tokens = append(tokens, token.ID)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("create token with description", func(t *testing.T) {
		token, err := client.UserTokens.Generate(ctx, user.ID, UserTokenGenerateOptions{
			Description: fmt.Sprintf("go-tfe-user-token-test-%s", randomString(t)),
		})
		tokens = append(tokens, token.ID)
		if err != nil {
			t.Fatal(err)
		}
	})
}

// TestUserTokens_Read tests basic creation of user tokens
func TestUserTokens_Read(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()
	user, err := client.Users.ReadCurrent(ctx)
	if err != nil {
		t.Fatal(err)
	}

	token, tokenCleanupFunc := createToken(t, client, user)
	defer tokenCleanupFunc()

	t.Run("read token", func(t *testing.T) {
		to, err := client.UserTokens.Read(ctx, token.ID)
		if err != nil {
			t.Fatalf("expected to read token (%s), got error: %s", token.ID, err)
		}
		// The initial API call to create a token will return a value in the token
		// object. Empty that out for comparison
		token.Token = ""
		assert.Equal(t, token, to)
	})
}

// createToken is a helper method to create a valid token for a given user,
// which returns both the token and a function to revoke it
func createToken(t *testing.T, client *Client, user *User) (*UserToken, func()) {
	t.Helper()
	ctx := context.Background()
	if user == nil {
		t.Fatal("Nil user in createToken")
	}
	token, err := client.UserTokens.Generate(ctx, user.ID, UserTokenGenerateOptions{
		Description: fmt.Sprintf("go-tfe-user-token-test-%s", randomString(t)),
	})
	if err != nil {
		t.Fatal(err)
	}

	return token, func() {
		if err := client.UserTokens.Delete(ctx, token.ID); err != nil {
			t.Errorf("Error destroying token! WARNING: Dangling resources\n"+
				"may exist! The full error is shown below.\n\n"+
				"Token: %s\nError: %s", token.ID, err)
		}
	}
}
