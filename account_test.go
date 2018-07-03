package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccountsRead(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	a, err := client.Accounts.Read(ctx)
	assert.NoError(t, err)
	assert.NotEmpty(t, a.ID)
	assert.NotEmpty(t, a.AvatarURL)
	assert.NotEmpty(t, a.Username)

	t.Run("two factor options are decoded", func(t *testing.T) {
		assert.NotNil(t, a.TwoFactor)
	})
}

func TestAccountsUpdate(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	aTest, err := client.Accounts.Read(ctx)
	require.NoError(t, err)

	// Make sure we reset the current account when were done.
	defer func() {
		client.Accounts.Update(ctx, AccountUpdateOptions{
			Email:    String(aTest.Email),
			Username: String(aTest.Username),
		})
	}()

	t.Run("without any options", func(t *testing.T) {
		_, err := client.Accounts.Update(ctx, AccountUpdateOptions{})
		require.NoError(t, err)

		a, err := client.Accounts.Read(ctx)
		assert.NoError(t, err)
		assert.Equal(t, a, aTest)
	})

	t.Run("with a new username", func(t *testing.T) {
		_, err := client.Accounts.Update(ctx, AccountUpdateOptions{
			Username: String("NewTestUsername"),
		})
		require.NoError(t, err)

		a, err := client.Accounts.Read(ctx)
		assert.NoError(t, err)
		assert.Equal(t, "NewTestUsername", a.Username)
	})

	t.Run("with a new email address", func(t *testing.T) {
		_, err := client.Accounts.Update(ctx, AccountUpdateOptions{
			Email: String("newtestemail@hashicorp.com"),
		})
		require.NoError(t, err)

		a, err := client.Accounts.Read(ctx)
		assert.NoError(t, err)
		assert.Equal(t, "newtestemail@hashicorp.com", a.UnconfirmedEmail)
	})

	t.Run("with invalid email address", func(t *testing.T) {
		a, err := client.Accounts.Update(ctx, AccountUpdateOptions{
			Email: String("notamailaddress"),
		})
		assert.Nil(t, a)
		assert.Error(t, err)
	})
}

func TestAccountsEnableTwoFactor(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	defer client.Accounts.DisableTwoFactor(ctx)

	t.Run("using sms as the second factor", func(t *testing.T) {
		a, err := client.Accounts.EnableTwoFactor(ctx, TwoFactorEnableOptions{
			Delivery:  Delivery(DeliverySMS),
			SMSNumber: String("+49123456789"),
		})
		require.NoError(t, err)
		require.NotNil(t, a.TwoFactor)
		assert.Equal(t, DeliverySMS, a.TwoFactor.Delivery)
		assert.True(t, a.TwoFactor.Enabled)
		assert.Equal(t, "+49123456789", a.TwoFactor.SMSNumber)

		// Reset the two factor settings again.
		_, err = client.Accounts.DisableTwoFactor(ctx)
		require.NoError(t, err)
	})

	t.Run("using an app as the second factor", func(t *testing.T) {
		a, err := client.Accounts.EnableTwoFactor(ctx, TwoFactorEnableOptions{
			Delivery: Delivery(DeliveryAPP),
		})
		require.NoError(t, err)
		require.NotNil(t, a.TwoFactor)
		assert.Equal(t, DeliveryAPP, a.TwoFactor.Delivery)
		assert.True(t, a.TwoFactor.Enabled)
		assert.Empty(t, a.TwoFactor.SMSNumber)

		// Reset the two factor settings again.
		_, err = client.Accounts.DisableTwoFactor(ctx)
		require.NoError(t, err)
	})

	t.Run("using an app as second factor with sms as backup", func(t *testing.T) {
		a, err := client.Accounts.EnableTwoFactor(ctx, TwoFactorEnableOptions{
			Delivery:  Delivery(DeliveryAPP),
			SMSNumber: String("+49123456789"),
		})
		require.NoError(t, err)
		require.NotNil(t, a.TwoFactor)
		assert.Equal(t, DeliveryAPP, a.TwoFactor.Delivery)
		assert.True(t, a.TwoFactor.Enabled)
		assert.Equal(t, "+49123456789", a.TwoFactor.SMSNumber)

		// Reset the two factor settings again.
		_, err = client.Accounts.DisableTwoFactor(ctx)
		require.NoError(t, err)
	})

	t.Run("without a delivery type", func(t *testing.T) {
		_, err := client.Accounts.EnableTwoFactor(ctx, TwoFactorEnableOptions{
			SMSNumber: String("+49123456789"),
		})
		assert.EqualError(t, err, "Delivery is required")
	})
}

func TestAccountsDisableTwoFactor(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	defer client.Accounts.DisableTwoFactor(ctx)

	aTest, err := client.Accounts.EnableTwoFactor(ctx, TwoFactorEnableOptions{
		Delivery:  Delivery(DeliveryAPP),
		SMSNumber: String("+49123456789"),
	})
	require.NoError(t, err)
	require.NotEmpty(t, aTest.TwoFactor)

	t.Run("when two factor authentication is enabled", func(t *testing.T) {
		a, err := client.Accounts.DisableTwoFactor(ctx)
		require.NoError(t, err)
		assert.Empty(t, a.TwoFactor)
	})

	t.Run("when two factor authentication is disabled", func(t *testing.T) {
		_, err := client.Accounts.DisableTwoFactor(ctx)
		assert.Error(t, err)
	})
}

func TestAccountsVerifyTwoFactor(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	defer client.Accounts.DisableTwoFactor(ctx)

	t.Run("when using an invalid code", func(t *testing.T) {
		aTest, err := client.Accounts.EnableTwoFactor(ctx, TwoFactorEnableOptions{
			Delivery: Delivery(DeliveryAPP),
		})
		require.NoError(t, err)
		require.True(t, aTest.TwoFactor.Enabled)

		_, err = client.Accounts.VerifyTwoFactor(ctx, TwoFactorVerifyOptions{
			Code: String("123456"),
		})
		assert.Contains(t, err.Error(), "Two factor code is incorrect")
	})

	t.Run("when two factor authentication is disabled", func(t *testing.T) {
		aTest, err := client.Accounts.DisableTwoFactor(ctx)
		require.NoError(t, err)
		require.False(t, aTest.TwoFactor.Enabled)

		_, err = client.Accounts.VerifyTwoFactor(ctx, TwoFactorVerifyOptions{
			Code: String("123456"),
		})
		assert.Contains(t, err.Error(), "Two-factor authentication is not enabled")
	})

	t.Run("without a verification code", func(t *testing.T) {
		_, err := client.Accounts.VerifyTwoFactor(ctx, TwoFactorVerifyOptions{})
		assert.EqualError(t, err, "Code is required")
	})
}

func TestAccountsResendVerificationCode(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	t.Run("when two factor authentication is app enabled", func(t *testing.T) {
		aTest, err := client.Accounts.EnableTwoFactor(ctx, TwoFactorEnableOptions{
			Delivery: Delivery(DeliveryAPP),
		})
		require.NoError(t, err)
		require.True(t, aTest.TwoFactor.Enabled)

		err = client.Accounts.ResendVerificationCode(ctx)
		assert.Error(t, err)
	})

	t.Run("when two factor authentication is sms enabled", func(t *testing.T) {
		aTest, err := client.Accounts.EnableTwoFactor(ctx, TwoFactorEnableOptions{
			Delivery:  Delivery(DeliverySMS),
			SMSNumber: String("+49123456789"),
		})
		require.NoError(t, err)
		require.True(t, aTest.TwoFactor.Enabled)

		err = client.Accounts.ResendVerificationCode(ctx)
		assert.NoError(t, err)
	})

	t.Run("when two factor authentication is disabled", func(t *testing.T) {
		aTest, err := client.Accounts.DisableTwoFactor(ctx)
		require.NoError(t, err)
		require.False(t, aTest.TwoFactor.Enabled)

		_, err = client.Accounts.DisableTwoFactor(ctx)
		assert.Contains(t, err.Error(), "Two-factor authentication is not enabled")
	})
}
