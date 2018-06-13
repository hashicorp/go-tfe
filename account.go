package tfe

import (
	"errors"
)

// Accounts handles communication with the account related methods of the
// the Terraform Enterprise API.
//
// TFE API docs: https://www.terraform.io/docs/enterprise/api/account.html
type Accounts struct {
	client *Client
}

// Account represents a Terraform Enterprise account.
type Account struct {
	ID               string     `jsonapi:"primary,users"`
	AvatarURL        string     `jsonapi:"attr,avatar-url"`
	IsServiceAccount bool       `jsonapi:"attr,is-service-account"`
	TwoFactor        *TwoFactor `jsonapi:"attr,two-factor"`
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
	Delivery          DeliveryType `json:"Delivery"`
	Enabled           bool         `json:"enabled"`
	ProvisioningURL   string       `json:"provisioning-url"`
	RecoveryCodes     []string     `json:"recovery-codes"`
	SMSNumber         string       `json:"sms-number"`
	UsedRecoveryCodes []string     `json:"used-recovery-codes"`
	Verified          bool         `json:"verified"`
}

// AccountUpdateOptions represents the options for updating an account.
type AccountUpdateOptions struct {
	// For internal use only!
	ID string `jsonapi:"primary,organizations"`

	// New username.
	Username *string `jsonapi:"attr,username,omitempty"`

	// New email address (must be consumed afterwards to take effect).
	Email *string `jsonapi:"attr,email,omitempty"`
}

// Update attributes of the currently authenticated user.
func (s *Accounts) Update(options AccountUpdateOptions) (*Account, error) {
	// Make sure we don't send a user provided ID.
	options.ID = ""

	req, err := s.client.newRequest("PATCH", "account/update", &options)
	if err != nil {
		return nil, err
	}

	a, err := s.client.do(req, &Account{})
	if err != nil {
		return nil, err
	}

	return a.(*Account), nil
}

// TwoFactorEnableOptions represents the options for enabling two factor
// authentication.
type TwoFactorEnableOptions struct {
	// For internal use only!
	ID string `jsonapi:"primary,organizations"`

	// The preferred delivery method for 2FA.
	Delivery *string `jsonapi:"attr,delivery,omitempty"`

	// An SMS number for the SMS delivery method.
	SMSNumber *string `jsonapi:"attr,sms-number,omitempty"`
}

func (o TwoFactorEnableOptions) valid() error {
	if !validString(o.Delivery) {
		return errors.New("Invalid value for delivery")
	}
	return nil
}

// EnableTwoFactor enables two factor authentication.
func (s *Accounts) EnableTwoFactor(options TwoFactorEnableOptions) (*Account, error) {
	if err := options.valid(); err != nil {
		return nil, err
	}

	// Make sure we don't send a user provided ID.
	options.ID = ""

	req, err := s.client.newRequest("POST", "account/actions/two-factor-enable", &options)
	if err != nil {
		return nil, err
	}

	a, err := s.client.do(req, &Account{})
	if err != nil {
		return nil, err
	}

	return a.(*Account), nil
}

// DisableTwoFactor disables two factor authentication.
func (s *Accounts) DisableTwoFactor() (*Account, error) {
	req, err := s.client.newRequest("POST", "account/actions/two-factor-disable", nil)
	if err != nil {
		return nil, err
	}

	a, err := s.client.do(req, &Account{})
	if err != nil {
		return nil, err
	}

	return a.(*Account), nil
}

// TwoFactorVerifyOptions represents the options for verifying two factor
// authentication.
type TwoFactorVerifyOptions struct {
	// For internal use only!
	ID string `jsonapi:"primary,organizations"`

	// The verication code received by SMS or through an application.
	Code *string `jsonapi:"attr,code,omitempty"`
}

func (o TwoFactorVerifyOptions) valid() error {
	if !validString(o.Code) {
		return errors.New("Invalid value for code")
	}
	return nil
}

// VerifyTwoFactor verifies two factor authentication.
func (s *Accounts) VerifyTwoFactor(options TwoFactorVerifyOptions) (*Account, error) {
	if err := options.valid(); err != nil {
		return nil, err
	}

	// Make sure we don't send a user provided ID.
	options.ID = ""

	req, err := s.client.newRequest("POST", "account/actions/two-factor-verify", &options)
	if err != nil {
		return nil, err
	}

	a, err := s.client.do(req, &Account{})
	if err != nil {
		return nil, err
	}

	return a.(*Account), nil
}

// ResendVerificationCode resends the two factor verification code.
func (s *Accounts) ResendVerificationCode() error {
	req, err := s.client.newRequest(
		"POST",
		"account/actions/two-factor-resend-verification-code",
		nil,
	)
	if err != nil {
		return err
	}

	_, err = s.client.do(req, nil)

	return err
}
