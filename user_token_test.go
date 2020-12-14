package tfe

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUserTokens_Basic(t *testing.T) {
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

func createToken(t *testing.T, client *Client, user *User) (*UserToken, func()) {
	t.Helper()
	ctx := context.Background()
	if user == nil {
		t.Fatal("Nil user in createToken")
	}
	token, err := client.UserTokens.Create(ctx, user.ID, UserTokenCreateOptions{
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
