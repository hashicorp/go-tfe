package tfe

import (
	"context"
)

// Compile-time proof of interface implementation.
var _ Saml = (*saml)(nil)

// Saml describes SAML settings
// TFE API docs: 
// - https://www.terraform.io/docs/cloud/api/admin/settings.html#list-saml-settings
// - https://www.terraform.io/docs/cloud/api/admin/settings.html#update-saml-settings
type Saml interface {
	// Lists the SAML configuration for the TFE instances
	List(ctx context.Context) (*SamlSettings, error)

	// Updates the SAML configuration for the TFE instance
	Update(ctx context.Context, options SamlUpdateOptions) (*SamlSettings, error)
}

// saml implements Saml config.
type saml struct {
	client *Client
}

// SamlSettings represents the current SAML Settings of a TFE instance
type SamlSettings struct {
	Enabled                   bool   `jsonapi:"attr,enabled"`
	Debug                     bool   `jsonapi:"attr,debug"`
	OldIDPCert                string `jsonapi:"attr,old-idp-cert"`
	IDPCert                   string `jsonapi:"attr,idp-cert"`
	SLOEndpointURL            string `jsonapi:"attr,slo-endpoint-url"`
	SSOEndpointURL            string `jsonapi:"attr,sso-endpoint-url"`
	AttrUsername              string `jsonapi:"attr,attr-username"`
	AttrGroups                string `jsonapi:"attr,attr-groups"`
	AttrSiteAdmin             string `jsonapi:"attr,attr-side-admin"`
	SiteAdminRole             string `jsonapi:"attr,site-admin-role"`
	SSOApiTokenSessionTimeout int    `jsonapi:"attr,sso-api-token-session-timeout"`
	ACSConsumerURL            string `jsonapi:"attr,acs-consumer-url"`
	MetadataURL               string `jsonapi:"attr,metadata-url"`
}

// List all of the current SAML configuration parameters
func (s *saml) List(ctx context.Context) (*SamlSettings, error) {

	req, err := s.client.newRequest("GET", "admin/saml-settings", nil)

	if err != nil {
		return nil, err
	}

	samlSettings := &SamlSettings{}
	err = s.client.do(ctx, req, samlSettings)
	if err != nil {
		return nil, err
	}

	return samlSettings, nil
}

// SamlUpdateOptions represents the options to update SAML configuration
type SamlUpdateOptions struct {
	Enabled                   bool   `jsonapi:"attr,enabled"`
	Debug                     bool   `jsonapi:"attr,debug"`
	IDPCert                   string `jsonapi:"attr,idp-cert"`
	SLOEndpointURL            string `jsonapi:"attr,slo-endpoint-url"`
	SSOEndpointURL            string `jsonapi:"attr,sso-endpoint-url"`
	AttrUsername              string `jsonapi:"attr,attr-username"`
	AttrGroups                string `jsonapi:"attr,attr-groups"`
	AttrSiteAdmin             string `jsonapi:"attr,attr-side-admin"`
	SiteAdminRole             string `jsonapi:"attr,site-admin-role"`
	SSOApiTokenSessionTimeout int    `jsonapi:"attr,sso-api-token-session-timeout"`
	ACSConsumerURL            string `jsonapi:"attr,acs-consumer-url"`
	MetadataURL               string `jsonapi:"attr,metadata-url"`
}

// Update the SAML settings for a TFE instance
func (s *saml) Update(ctx context.Context, options SamlUpdateOptions) (*SamlSettings, error) {

	req, err := s.client.newRequest("PATCH", "admin/saml-settings", &options)

	if err != nil {
		return nil, err
	}

	k := &SamlSettings{}
	err = s.client.do(ctx, req, k)
	if err != nil {
		return nil, err
	}

	return k, nil
}
