package tfe

import (
	"context"
	"errors"
)

// Users handles communication with the user related methods of the
// the Terraform Enterprise API.
//
// TFE API docs: https://www.terraform.io/docs/enterprise/api/user.html
type Users struct {
	client *Client
}

// User represents a Terraform Enterprise user.
type User struct {
	ID               string     `jsonapi:"primary,users"`
	AvatarURL        string     `jsonapi:"attr,avatar-url"`
	Email            string     `jsonapi:"attr,email"`
	IsServiceAccount bool       `jsonapi:"attr,is-service-account"`
	TwoFactor        *TwoFactor `jsonapi:"attr,two-factor"`
	UnconfirmedEmail string     `jsonapi:"attr,unconfirmed-email"`
	Username         string     `jsonapi:"attr,username"`
	V2Only           bool       `jsonapi:"attr,v2-only"`

	// Relations
	// AuthenticationTokens *AuthenticationTokens `jsonapi:"relation,authentication-tokens"`
}

// DeliveryType represents a two factor delivery type
type DeliveryType string

// List of available delivery types.
const (
	DeliveryAPP DeliveryType = "app"
	DeliverySMS DeliveryType = "sms"
)

// TwoFactor represents the organization permissions.
type TwoFactor struct {
	Delivery        DeliveryType `json:"delivery"`
	Enabled         bool         `json:"enabled"`
	ProvisioningURL string       `json:"provisioning-url"`
	SMSNumber       string       `json:"sms-number"`
	Verified        bool         `json:"verified"`
}

// ReadCurrent reads the details of the currently authenticated user.
func (s *Users) ReadCurrent(ctx context.Context) (*User, error) {
	req, err := s.client.newRequest("GET", "account/details", nil)
	if err != nil {
		return nil, err
	}

	u := &User{}
	err = s.client.do(ctx, req, u)
	if err != nil {
		return nil, err
	}

	return u, nil
}

// UserUpdateOptions represents the options for updating a user.
type UserUpdateOptions struct {
	// For internal use only!
	ID string `jsonapi:"primary,users"`

	// New username.
	Username *string `jsonapi:"attr,username,omitempty"`

	// New email address (must be consumed afterwards to take effect).
	Email *string `jsonapi:"attr,email,omitempty"`
}

// Update attributes of the currently authenticated user.
func (s *Users) Update(ctx context.Context, options UserUpdateOptions) (*User, error) {
	// Make sure we don't send a user provided ID.
	options.ID = ""

	req, err := s.client.newRequest("PATCH", "account/update", &options)
	if err != nil {
		return nil, err
	}

	u := &User{}
	err = s.client.do(ctx, req, u)
	if err != nil {
		return nil, err
	}

	return u, nil
}

// TwoFactorEnableOptions represents the options for enabling two factor
// authentication.
type TwoFactorEnableOptions struct {
	// For internal use only!
	ID string `jsonapi:"primary,users"`

	// The preferred delivery method for 2FA.
	Delivery *DeliveryType `jsonapi:"attr,delivery"`

	// An SMS number for the SMS delivery method.
	SMSNumber *string `jsonapi:"attr,sms-number,omitempty"`
}

func (o TwoFactorEnableOptions) valid() error {
	if o.Delivery == nil {
		return errors.New("Delivery is required")
	}
	return nil
}

// EnableTwoFactor enables two factor authentication.
func (s *Users) EnableTwoFactor(ctx context.Context, options TwoFactorEnableOptions) (*User, error) {
	if err := options.valid(); err != nil {
		return nil, err
	}

	// Make sure we don't send a user provided ID.
	options.ID = ""

	req, err := s.client.newRequest("POST", "account/actions/two-factor-enable", &options)
	if err != nil {
		return nil, err
	}

	u := &User{}
	err = s.client.do(ctx, req, u)
	if err != nil {
		return nil, err
	}

	return u, nil
}

// DisableTwoFactor disables two factor authentication.
func (s *Users) DisableTwoFactor(ctx context.Context) (*User, error) {
	req, err := s.client.newRequest("POST", "account/actions/two-factor-disable", nil)
	if err != nil {
		return nil, err
	}

	u := &User{}
	err = s.client.do(ctx, req, u)
	if err != nil {
		return nil, err
	}

	return u, nil
}

// TwoFactorVerifyOptions represents the options for verifying two factor
// authentication.
type TwoFactorVerifyOptions struct {
	// For internal use only!
	ID string `jsonapi:"primary,organizations"`

	// The verication code received by SMS or through an application.
	Code *string `jsonapi:"attr,code"`
}

func (o TwoFactorVerifyOptions) valid() error {
	if !validString(o.Code) {
		return errors.New("Code is required")
	}
	return nil
}

// VerifyTwoFactor verifies two factor authentication.
func (s *Users) VerifyTwoFactor(ctx context.Context, options TwoFactorVerifyOptions) (*User, error) {
	if err := options.valid(); err != nil {
		return nil, err
	}

	// Make sure we don't send a user provided ID.
	options.ID = ""

	req, err := s.client.newRequest("POST", "account/actions/two-factor-verify", &options)
	if err != nil {
		return nil, err
	}

	u := &User{}
	err = s.client.do(ctx, req, u)
	if err != nil {
		return nil, err
	}

	return u, nil
}

// ResendVerificationCode resends the two factor verification code.
func (s *Users) ResendVerificationCode(ctx context.Context) error {
	req, err := s.client.newRequest(
		"POST",
		"account/actions/two-factor-resend-verification-code",
		nil,
	)
	if err != nil {
		return err
	}

	return s.client.do(ctx, req, nil)
}
