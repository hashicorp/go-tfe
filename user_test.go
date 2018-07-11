package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUsersReadCurrent(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	u, err := client.Users.ReadCurrent(ctx)
	assert.NoError(t, err)
	assert.NotEmpty(t, u.ID)
	assert.NotEmpty(t, u.AvatarURL)
	assert.NotEmpty(t, u.Username)

	t.Run("two factor options are decoded", func(t *testing.T) {
		assert.NotNil(t, u.TwoFactor)
	})
}

func TestUsersUpdate(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	uTest, err := client.Users.ReadCurrent(ctx)
	require.NoError(t, err)

	// Make sure we reset the current user when were done.
	defer func() {
		client.Users.Update(ctx, UserUpdateOptions{
			Email:    String(uTest.Email),
			Username: String(uTest.Username),
		})
	}()

	t.Run("without any options", func(t *testing.T) {
		_, err := client.Users.Update(ctx, UserUpdateOptions{})
		require.NoError(t, err)

		u, err := client.Users.ReadCurrent(ctx)
		assert.NoError(t, err)
		assert.Equal(t, u, uTest)
	})

	t.Run("with a new username", func(t *testing.T) {
		_, err := client.Users.Update(ctx, UserUpdateOptions{
			Username: String("NewTestUsername"),
		})
		require.NoError(t, err)

		u, err := client.Users.ReadCurrent(ctx)
		assert.NoError(t, err)
		assert.Equal(t, "NewTestUsername", u.Username)
	})

	t.Run("with a new email address", func(t *testing.T) {
		_, err := client.Users.Update(ctx, UserUpdateOptions{
			Email: String("newtestemail@hashicorp.com"),
		})
		require.NoError(t, err)

		u, err := client.Users.ReadCurrent(ctx)
		assert.NoError(t, err)
		assert.Equal(t, "newtestemail@hashicorp.com", u.UnconfirmedEmail)
	})

	t.Run("with invalid email address", func(t *testing.T) {
		u, err := client.Users.Update(ctx, UserUpdateOptions{
			Email: String("notamailaddress"),
		})
		assert.Nil(t, u)
		assert.Error(t, err)
	})
}

func TestUsersEnableTwoFactor(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	defer client.Users.DisableTwoFactor(ctx)

	t.Run("using sms as the second factor", func(t *testing.T) {
		u, err := client.Users.EnableTwoFactor(ctx, TwoFactorEnableOptions{
			Delivery:  Delivery(DeliverySMS),
			SMSNumber: String("+49123456789"),
		})
		require.NoError(t, err)
		require.NotNil(t, u.TwoFactor)
		assert.Equal(t, DeliverySMS, u.TwoFactor.Delivery)
		assert.True(t, u.TwoFactor.Enabled)
		assert.Equal(t, "+49123456789", u.TwoFactor.SMSNumber)

		// Reset the two factor settings again.
		_, err = client.Users.DisableTwoFactor(ctx)
		require.NoError(t, err)
	})

	t.Run("using an app as the second factor", func(t *testing.T) {
		u, err := client.Users.EnableTwoFactor(ctx, TwoFactorEnableOptions{
			Delivery: Delivery(DeliveryAPP),
		})
		require.NoError(t, err)
		require.NotNil(t, u.TwoFactor)
		assert.Equal(t, DeliveryAPP, u.TwoFactor.Delivery)
		assert.True(t, u.TwoFactor.Enabled)
		assert.Empty(t, u.TwoFactor.SMSNumber)

		// Reset the two factor settings again.
		_, err = client.Users.DisableTwoFactor(ctx)
		require.NoError(t, err)
	})

	t.Run("using an app as second factor with sms as backup", func(t *testing.T) {
		u, err := client.Users.EnableTwoFactor(ctx, TwoFactorEnableOptions{
			Delivery:  Delivery(DeliveryAPP),
			SMSNumber: String("+49123456789"),
		})
		require.NoError(t, err)
		require.NotNil(t, u.TwoFactor)
		assert.Equal(t, DeliveryAPP, u.TwoFactor.Delivery)
		assert.True(t, u.TwoFactor.Enabled)
		assert.Equal(t, "+49123456789", u.TwoFactor.SMSNumber)

		// Reset the two factor settings again.
		_, err = client.Users.DisableTwoFactor(ctx)
		require.NoError(t, err)
	})

	t.Run("without a delivery type", func(t *testing.T) {
		_, err := client.Users.EnableTwoFactor(ctx, TwoFactorEnableOptions{
			SMSNumber: String("+49123456789"),
		})
		assert.EqualError(t, err, "Delivery is required")
	})
}

func TestUsersDisableTwoFactor(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	defer client.Users.DisableTwoFactor(ctx)

	uTest, err := client.Users.EnableTwoFactor(ctx, TwoFactorEnableOptions{
		Delivery:  Delivery(DeliveryAPP),
		SMSNumber: String("+49123456789"),
	})
	require.NoError(t, err)
	require.NotEmpty(t, uTest.TwoFactor)

	t.Run("when two factor authentication is enabled", func(t *testing.T) {
		u, err := client.Users.DisableTwoFactor(ctx)
		require.NoError(t, err)
		assert.Empty(t, u.TwoFactor)
	})

	t.Run("when two factor authentication is disabled", func(t *testing.T) {
		_, err := client.Users.DisableTwoFactor(ctx)
		assert.Error(t, err)
	})
}

func TestUsersVerifyTwoFactor(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	defer client.Users.DisableTwoFactor(ctx)

	t.Run("when using an invalid code", func(t *testing.T) {
		uTest, err := client.Users.EnableTwoFactor(ctx, TwoFactorEnableOptions{
			Delivery: Delivery(DeliveryAPP),
		})
		require.NoError(t, err)
		require.True(t, uTest.TwoFactor.Enabled)

		_, err = client.Users.VerifyTwoFactor(ctx, TwoFactorVerifyOptions{
			Code: String("123456"),
		})
		assert.Contains(t, err.Error(), "Two factor code is incorrect")
	})

	t.Run("when two factor authentication is disabled", func(t *testing.T) {
		uTest, err := client.Users.DisableTwoFactor(ctx)
		require.NoError(t, err)
		require.False(t, uTest.TwoFactor.Enabled)

		_, err = client.Users.VerifyTwoFactor(ctx, TwoFactorVerifyOptions{
			Code: String("123456"),
		})
		assert.Contains(t, err.Error(), "Two-factor authentication is not enabled")
	})

	t.Run("without a verification code", func(t *testing.T) {
		_, err := client.Users.VerifyTwoFactor(ctx, TwoFactorVerifyOptions{})
		assert.EqualError(t, err, "Code is required")
	})
}

func TestUsersResendVerificationCode(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	t.Run("when two factor authentication is app enabled", func(t *testing.T) {
		uTest, err := client.Users.EnableTwoFactor(ctx, TwoFactorEnableOptions{
			Delivery: Delivery(DeliveryAPP),
		})
		require.NoError(t, err)
		require.True(t, uTest.TwoFactor.Enabled)

		err = client.Users.ResendVerificationCode(ctx)
		assert.Error(t, err)
	})

	t.Run("when two factor authentication is sms enabled", func(t *testing.T) {
		uTest, err := client.Users.EnableTwoFactor(ctx, TwoFactorEnableOptions{
			Delivery:  Delivery(DeliverySMS),
			SMSNumber: String("+49123456789"),
		})
		require.NoError(t, err)
		require.True(t, uTest.TwoFactor.Enabled)

		err = client.Users.ResendVerificationCode(ctx)
		assert.NoError(t, err)
	})

	t.Run("when two factor authentication is disabled", func(t *testing.T) {
		uTest, err := client.Users.DisableTwoFactor(ctx)
		require.NoError(t, err)
		require.False(t, uTest.TwoFactor.Enabled)

		_, err = client.Users.DisableTwoFactor(ctx)
		assert.Contains(t, err.Error(), "Two-factor authentication is not enabled")
	})
}
