package tfe

import (
	"context"
)

// Compile-time proof of interface implementation.
var _ AdminSettings = (*adminSettings)(nil)

// AdminSettings describes all the admin settings related methods that the Terraform Enterprise API supports.
// Note that admin settings are only available in Terraform Enterprise.
//
// TFE API docs: https://www.terraform.io/docs/cloud/api/admin/settings.html
type AdminSettings interface {
	// GetGeneral returns the general settings
	GetGeneral(ctx context.Context) (*AdminGeneralSetting, error)

	// UpdateGeneral updates general settings.
	UpdateGeneral(ctx context.Context, options AdminGeneralSettingsUpdateOptions) (*AdminGeneralSetting, error)

	// GetCostEstimation returns the cost estimation settings.
	GetCostEstimation(ctx context.Context) (*AdminCostEstimationSetting, error)

	// UpdateCostEstimation updates the cost estimation settings.
	UpdateCostEstimation(ctx context.Context, options AdminCostEstimationSettingOptions) (*AdminCostEstimationSetting, error)

	// GetSAML returns the SAML settings.
	GetSAML(ctx context.Context) (*AdminSAMLSetting, error)

	// UpdateSAML updates the SAML settings.
	UpdateSAML(ctx context.Context, options AdminSAMLSettingsUpdateOptions) (*AdminSAMLSetting, error)

	// RevokeSAMLIdpCert revokes the older IdP certificate when the new IdP
	// certificate is known to be functioning correctly.
	RevokeSAMLIdpCert(ctx context.Context) (*AdminSAMLSetting, error)

	// GetSMTP returns the SMTP settings.
	GetSMTP(ctx context.Context) (*AdminSMTPSetting, error)

	// UpdateSMTP updates SMTP settings.
	UpdateSMTP(ctx context.Context, options AdminSMTPSettingsUpdateOptions) (*AdminSMTPSetting, error)

	// GetTwilio returns the Twilio settings.
	GetTwilio(ctx context.Context) (*AdminTwilioSetting, error)

	// UpdateTwilio updates Twilio settings.
	UpdateTwilio(ctx context.Context, options AdminTwilioSettingsUpdateOptions) (*AdminTwilioSetting, error)

	// VerifyTwilio verifies Twilio settings.
	VerifyTwilio(ctx context.Context, options AdminTwilioSettingsVerifyOptions) error

	// GetCustomization returns the customization settings.
	GetCustomization(ctx context.Context) (*AdminCustomizationSetting, error)

	// UpdateCustomization updates the customization settings.
	UpdateCustomization(ctx context.Context, options AdminCustomizationSettingsUpdateOptions) (*AdminCustomizationSetting, error)
}

// adminSettings implements AdminSettings.
type adminSettings struct {
	client *Client
}

// AdminGeneralSetting represents a the general settings in Terraform Enterprise.
type AdminGeneralSetting struct {
	ID                            string `jsonapi:"primary,general-settings"`
	LimitUserOrganizationCreation bool   `jsonapi:"attr,limit-user-organization-creation"`
	APIRateLimitingEnabled        bool   `jsonapi:"attr,api-rate-limiting-enabled"`
	APIRateLimit                  int    `jsonapi:"attr,api-rate-limit"`
	SendPassingStatusesEnabled    bool   `jsonapi:"attr,send-passing-statuses-for-untriggered-speculative-plans"`
	AllowSpeculativePlansOnPR     bool   `jsonapi:"attr,allow-speculative-plans-on-pull-requests-from-forks"`
}

// GetGeneral returns the general settings.
func (s *adminSettings) GetGeneral(ctx context.Context) (*AdminGeneralSetting, error) {
	req, err := s.client.newRequest("GET", "admin/general-settings", nil)
	if err != nil {
		return nil, err
	}

	ags := &AdminGeneralSetting{}
	err = s.client.do(ctx, req, ags)
	if err != nil {
		return nil, err
	}

	return ags, nil
}

// AdminGeneralSettingsUpdateOptions represents the admin options for updating
// general settings.
// https://www.terraform.io/docs/cloud/api/admin/settings.html#request-body
type AdminGeneralSettingsUpdateOptions struct {
	LimitUserOrgCreation              *bool `jsonapi:"attr,limit-user-organization-creation,omitempty"`
	APIRateLimitingEnabled            *bool `jsonapi:"attr,api-rate-limiting-enabled,omitempty"`
	APIRateLimit                      *int  `jsonapi:"attr,api-rate-limit,omitempty"`
	SendPassingStatusUntriggeredPlans *bool `jsonapi:"attr,send-passing-statuses-for-untriggered-speculative-plans,omitempty"`
	AllowSpeculativePlansOnPR         *bool `jsonapi:"attr,allow-speculative-plans-on-pull-requests-from-forks,omitempty"`
}

// UpdateGeneral updates the general settings.
func (s *adminSettings) UpdateGeneral(ctx context.Context, options AdminGeneralSettingsUpdateOptions) (*AdminGeneralSetting, error) {
	req, err := s.client.newRequest("PATCH", "admin/general-settings", &options)
	if err != nil {
		return nil, err
	}

	ags := &AdminGeneralSetting{}
	err = s.client.do(ctx, req, ags)
	if err != nil {
		return nil, err
	}

	return ags, nil
}

// AdminCostEstimationSetting represents the admin cost estimation settings.
type AdminCostEstimationSetting struct {
	ID                  string `jsonapi:"primary,cost-estimation-settings"`
	Enabled             bool   `jsonapi:"attr,enabled"`
	AWSAccessKeyID      string `jsonapi:"attr,aws-access-key-id"`
	AWSAccessKey        string `jsonapi:"attr,aws-secret-key"`
	GCPCredentials      string `jsonapi:"attr,gcp-credentials"`
	AzureClientID       string `jsonapi:"attr,azure-client-id"`
	AzureClientSecret   string `jsonapi:"attr,azure-client-secret"`
	AzureSubscriptionID string `jsonapi:"attr,azure-subscription-id"`
	AzureTenantID       string `jsonapi:"attr,azure-tenant-id"`
}

// GetCostEstimation returns the cost estimation settings.
func (s *adminSettings) GetCostEstimation(ctx context.Context) (*AdminCostEstimationSetting, error) {
	req, err := s.client.newRequest("GET", "admin/cost-estimation-settings", nil)
	if err != nil {
		return nil, err
	}

	ags := &AdminCostEstimationSetting{}
	err = s.client.do(ctx, req, ags)
	if err != nil {
		return nil, err
	}

	return ags, nil
}

// AdminCostEstimationSettingOptions represents the admin options for updating
// the cost estimation settings.
// https://www.terraform.io/docs/cloud/api/admin/settings.html#request-body-1
type AdminCostEstimationSettingOptions struct {
	Enabled             *bool   `jsonapi:"attr,enabled,omitempty"`
	AWSAccessKeyID      *string `jsonapi:"attr,aws-access-key-id,omitempty"`
	AWSAccessKey        *string `jsonapi:"attr,aws-secret-key,omitempty"`
	GCPCredentials      *string `jsonapi:"attr,gcp-credentials,omitempty"`
	AzureClientID       *string `jsonapi:"attr,azure-client-id,omitempty"`
	AzureClientSecret   *string `jsonapi:"attr,azure-client-secret,omitempty"`
	AzureSubscriptionID *string `jsonapi:"attr,azure-subscription-id,omitempty"`
	AzureTenantID       *string `jsonapi:"attr,azure-tenant-id,omitempty"`
}

// UpdateCostEstimation updates the cost-estimation settings.
func (s *adminSettings) UpdateCostEstimation(ctx context.Context, options AdminCostEstimationSettingOptions) (*AdminCostEstimationSetting, error) {
	req, err := s.client.newRequest("PATCH", "admin/cost-estimation-settings", &options)
	if err != nil {
		return nil, err
	}

	ace := &AdminCostEstimationSetting{}
	err = s.client.do(ctx, req, ace)
	if err != nil {
		return nil, err
	}

	return ace, nil
}

// AdminSAMLSetting represents the SAML settings in Terraform Enterprise.
type AdminSAMLSetting struct {
	ID                        string `jsonapi:"primary,saml-settings"`
	Enabled                   bool   `jsonapi:"attr,enabled"`
	Debug                     bool   `jsonapi:"attr,debug"`
	OldIDPCert                string `jsonapi:"attr,old-idp-cert"`
	IDPCert                   string `jsonapi:"attr,idp-cert"`
	SLOEndpointURL            string `jsonapi:"attr,slo-endpoint-url"`
	SSOEndpointURL            string `jsonapi:"attr,sso-endpoint-url"`
	AttrUsername              string `jsonapi:"attr,attr-username"`
	AttrGroups                string `jsonapi:"attr,attr-groups"`
	AttrSiteAdmin             string `jsonapi:"attr,attr-site-admin"`
	SiteAdminRole             string `jsonapi:"attr,site-admin-role"`
	SSOAPITokenSessionTimeout int    `jsonapi:"attr,sso-api-token-session-timeout"`
	ACSConsumerURL            string `jsonapi:"attr,acs-consumer-url"`
	MetadataURL               string `jsonapi:"attr,metadata-url"`
}

// GetSAML returns the SAML settings.
func (s *adminSettings) GetSAML(ctx context.Context) (*AdminSAMLSetting, error) {
	req, err := s.client.newRequest("GET", "admin/saml-settings", nil)
	if err != nil {
		return nil, err
	}

	saml := &AdminSAMLSetting{}
	err = s.client.do(ctx, req, saml)
	if err != nil {
		return nil, err
	}

	return saml, nil
}

// AdminSAMLSettingsUpdateOptions represents the admin options for updating
// SAML settings.
// https://www.terraform.io/docs/cloud/api/admin/settings.html#request-body-2
type AdminSAMLSettingsUpdateOptions struct {
	Enabled                   *bool   `jsonapi:"attr,enabled,omitempty"`
	Debug                     *bool   `jsonapi:"attr,debug,omitempty"`
	IDPCert                   *string `jsonapi:"attr,idp-cert,omitempty"`
	SLOEndpointURL            *string `jsonapi:"attr,slo-endpoint-url,omitempty"`
	SSOEndpointURL            *string `jsonapi:"attr,sso-endpoint-url,omitempty"`
	AttrUsername              *string `jsonapi:"attr,attr-username,omitempty"`
	AttrGroups                *string `jsonapi:"attr,attr-groups,omitempty"`
	AttrSiteAdmin             *string `jsonapi:"attr,attr-site-admin,omitempty"`
	SiteAdminRole             *string `jsonapi:"attr,site-admin-role,omitempty"`
	SSOAPITokenSessionTimeout *int    `jsonapi:"attr,sso-api-token-session-timeout,omitempty"`
}

// UpdateSAML updates the SAML settings.
func (s *adminSettings) UpdateSAML(ctx context.Context, options AdminSAMLSettingsUpdateOptions) (*AdminSAMLSetting, error) {
	req, err := s.client.newRequest("PATCH", "admin/saml-settings", &options)
	if err != nil {
		return nil, err
	}

	saml := &AdminSAMLSetting{}
	err = s.client.do(ctx, req, saml)
	if err != nil {
		return nil, err
	}

	return saml, nil
}

// RevokeSAMLIdpCert revokes the older IdP certificate when the new IdP
// certificate is known to be functioning correctly.
func (s *adminSettings) RevokeSAMLIdpCert(ctx context.Context) (*AdminSAMLSetting, error) {
	req, err := s.client.newRequest("POST", "admin/saml-settings/actions/revoke-old-certificate", nil)
	if err != nil {
		return nil, err
	}

	saml := &AdminSAMLSetting{}
	err = s.client.do(ctx, req, saml)
	if err != nil {
		return nil, err
	}

	return saml, nil
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

// GetSMTP returns the SMTP settings.
func (s *adminSettings) GetSMTP(ctx context.Context) (*AdminSMTPSetting, error) {
	req, err := s.client.newRequest("GET", "admin/smtp-settings", nil)
	if err != nil {
		return nil, err
	}

	saml := &AdminSMTPSetting{}
	err = s.client.do(ctx, req, saml)
	if err != nil {
		return nil, err
	}

	return saml, nil
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

// UpdateSMTP updates the SMTP settings.
func (s *adminSettings) UpdateSMTP(ctx context.Context, options AdminSMTPSettingsUpdateOptions) (*AdminSMTPSetting, error) {
	if !options.valid() {
		return nil, ErrInvalidSMTPAuth
	}
	req, err := s.client.newRequest("PATCH", "admin/smtp-settings", &options)
	if err != nil {
		return nil, err
	}

	saml := &AdminSMTPSetting{}
	err = s.client.do(ctx, req, saml)
	if err != nil {
		return nil, err
	}

	return saml, nil
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

// AdminTwilioSetting represents the Twilio settings in Terraform Enterprise.
type AdminTwilioSetting struct {
	ID         string `jsonapi:"primary,twilio-settings"`
	Enabled    bool   `jsonapi:"attr,enabled"`
	AccountSid string `jsonapi:"attr,account-sid"`
	FromNumber string `jsonapi:"attr,from-number"`
}

// GetTwilio returns the Twilio settings.
func (s *adminSettings) GetTwilio(ctx context.Context) (*AdminTwilioSetting, error) {
	req, err := s.client.newRequest("GET", "admin/twilio-settings", nil)
	if err != nil {
		return nil, err
	}

	twilio := &AdminTwilioSetting{}
	err = s.client.do(ctx, req, twilio)
	if err != nil {
		return nil, err
	}

	return twilio, nil
}

// AdminTwilioSettingsUpdateOptions represents the admin options for updating
// Twilio settings.
// https://www.terraform.io/docs/cloud/api/admin/settings.html#request-body-4
type AdminTwilioSettingsUpdateOptions struct {
	Enabled    *bool   `jsonapi:"attr,enabled,omitempty"`
	AccountSid *string `jsonapi:"attr,account-sid,omitempty"`
	AuthToken  *string `jsonapi:"attr,auth-token,omitempty"`
	FromNumber *string `jsonapi:"attr,from-number,omitempty"`
}

// UpdateTwilio updates the Twilio settings.
func (s *adminSettings) UpdateTwilio(ctx context.Context, options AdminTwilioSettingsUpdateOptions) (*AdminTwilioSetting, error) {
	req, err := s.client.newRequest("PATCH", "admin/twilio-settings", &options)
	if err != nil {
		return nil, err
	}

	twilio := &AdminTwilioSetting{}
	err = s.client.do(ctx, req, twilio)
	if err != nil {
		return nil, err
	}

	return twilio, nil
}

// AdminTwilioSettingsVerifyOptions represents the test number to verify Twilio.
// https://www.terraform.io/docs/cloud/api/admin/settings.html#verify-twilio-settings
type AdminTwilioSettingsVerifyOptions struct {
	TestNumber *string `jsonapi:"attr,test-number"`
}

// VerifyTwilio verifies Twilio settings.
func (s *adminSettings) VerifyTwilio(ctx context.Context, options AdminTwilioSettingsVerifyOptions) error {
	req, err := s.client.newRequest("PATCH", "admin/twilio-settings/verify", &options)
	if err != nil {
		return err
	}

	return s.client.do(ctx, req, nil)
}

// AdminCustomizationSetting represents the Customization settings in Terraform Enterprise.
type AdminCustomizationSetting struct {
	ID           string `jsonapi:"primary,customization-settings"`
	SupportEmail string `jsonapi:"attr,support-email-address"`
	LoginHelp    string `jsonapi:"attr,login-help"`
	Footer       string `jsonapi:"attr,footer"`
	Error        string `jsonapi:"attr,error"`
	NewUser      string `jsonapi:"attr,new-user"`
}

// GetCustomization returns the Customization settings.
func (s *adminSettings) GetCustomization(ctx context.Context) (*AdminCustomizationSetting, error) {
	req, err := s.client.newRequest("GET", "admin/customization-settings", nil)
	if err != nil {
		return nil, err
	}

	cs := &AdminCustomizationSetting{}
	err = s.client.do(ctx, req, cs)
	if err != nil {
		return nil, err
	}

	return cs, nil
}

// AdminCustomizationSettingsUpdateOptions represents the admin options for updating
// Customization settings.
// https://www.terraform.io/docs/cloud/api/admin/settings.html#request-body-6
type AdminCustomizationSettingsUpdateOptions struct {
	SupportEmail *string `jsonapi:"attr,support-email-address,omitempty"`
	LoginHelp    *string `jsonapi:"attr,login-help,omitempty"`
	Footer       *string `jsonapi:"attr,footer,omitempty"`
	Error        *string `jsonapi:"attr,error,omitempty"`
	NewUser      *string `jsonapi:"attr,new-user,omitempty"`
}

// UpdateCustomization updates the customization settings.
func (s *adminSettings) UpdateCustomization(ctx context.Context, options AdminCustomizationSettingsUpdateOptions) (*AdminCustomizationSetting, error) {
	req, err := s.client.newRequest("PATCH", "admin/customization-settings", &options)
	if err != nil {
		return nil, err
	}

	cs := &AdminCustomizationSetting{}
	err = s.client.do(ctx, req, cs)
	if err != nil {
		return nil, err
	}

	return cs, nil
}
