package tfe

import (
	"context"
)

// Compile-time proof of interface implementation.
var _ SMTPSettings = (*adminSMTPSettings)(nil)

// SMTPSettings describes all the SMTP admin settings.
type SMTPSettings interface {
	// Read returns the SMTP settings.
	Read(ctx context.Context) (*AdminSMTPSetting, error)

	// Update updates SMTP settings.
	Update(ctx context.Context, options AdminSMTPSettingsUpdateOptions) (*AdminSMTPSetting, error)
}

type adminSMTPSettings struct {
	client *Client
}

// AdminSMTPSetting represents a the SMTP settings in Terraform Enterprise.
type AdminSMTPSetting struct {
	ID       string `jsonapi:"primary,smtp-settings"`
	Enabled  bool   `jsonapi:"attr,enabled"`
	Host     string `jsonapi:"attr,host"`
	Port     int    `jsonapi:"attr,port"`
	Sender   string `jsonapi:"attr,sender"`
	Auth     string `jsonapi:"attr,auth"`
	Username string `jsonapi:"attr,username"`
}

// Read returns the SMTP settings.
func (s *adminSMTPSettings) Read(ctx context.Context) (*AdminSMTPSetting, error) {
	req, err := s.client.newRequest("GET", "admin/smtp-settings", nil)
	if err != nil {
		return nil, err
	}

	smtp := &AdminSMTPSetting{}
	err = s.client.do(ctx, req, smtp)
	if err != nil {
		return nil, err
	}

	return smtp, nil
}

// AdminSMTPSettingsUpdateOptions represents the admin options for updating
// SMTP settings.
// https://www.terraform.io/docs/cloud/api/admin/settings.html#request-body-3
type AdminSMTPSettingsUpdateOptions struct {
	Enabled          *bool   `jsonapi:"attr,enabled,omitempty"`
	Host             *string `jsonapi:"attr,host,omitempty"`
	Port             *int    `jsonapi:"attr,port,omitempty"`
	Sender           *string `jsonapi:"attr,sender,omitempty"`
	Auth             *string `jsonapi:"attr,auth,omitempty"`
	Username         *string `jsonapi:"attr,username,omitempty"`
	Password         *string `jsonapi:"attr,password,omitempty"`
	TestEmailAddress *string `jsonapi:"attr,test-email-address,omitempty"`
}

// Updat updates the SMTP settings.
func (s *adminSMTPSettings) Update(ctx context.Context, options AdminSMTPSettingsUpdateOptions) (*AdminSMTPSetting, error) {
	if !options.valid() {
		return nil, ErrInvalidSMTPAuth
	}
	req, err := s.client.newRequest("PATCH", "admin/smtp-settings", &options)
	if err != nil {
		return nil, err
	}

	smtp := &AdminSMTPSetting{}
	err = s.client.do(ctx, req, smtp)
	if err != nil {
		return nil, err
	}

	return smtp, nil
}

// SMTPAuthType represents valid SMTP Auth types.
type SMTPAuthType string

// List of all SMTP auth types.
const (
	SMTPAuthNone  SMTPAuthType = "none"
	SMTPAuthPlain SMTPAuthType = "plain"
	SMTPAuthLogin SMTPAuthType = "login"
)

func (o AdminSMTPSettingsUpdateOptions) valid() bool {
	if !validString(o.Auth) {
		return false
	}

	validSMTPAuthType := map[string]int{
		string(SMTPAuthNone):  1,
		string(SMTPAuthPlain): 1,
		string(SMTPAuthLogin): 1,
	}

	_, isValidType := validSMTPAuthType[*o.Auth]
	return isValidType
}
