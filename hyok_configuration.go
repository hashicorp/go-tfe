package tfe

import (
	"context"
	"fmt"
	"net/url"
)

type HyokConfigurations interface {
	List(ctx context.Context, organization string, options *HyokConfigurationsListOptions) (*HyokConfigurationsList, error)

	TestUnpersisted(ctx context.Context, organization string) error

	Create(ctx context.Context, organization string, options HyokConfigurationsCreateOptions) (*HyokConfiguration, error)

	Read(ctx context.Context, hyokID string, options *HyokConfigurationsReadOptions) (*HyokConfiguration, error)

	Update(ctx context.Context, hyokID string, options HyokConfigurationsUpdateOptions) (*HyokConfiguration, error)

	Delete(ctx context.Context, hyokID string) error

	Test(ctx context.Context, hyokID string) error

	Revoke(ctx context.Context, hyokID string) error
}

type hyokConfigurations struct {
	client *Client
}

var _ HyokConfigurations = &hyokConfigurations{}

type OidcConfigurationChoice struct {
	AwsOidcConfiguration   *AwsOidcConfiguration
	GcpOidcConfiguration   *GcpOidcConfiguration
	AzureOidcConfiguration *AzureOidcConfiguration
	VaultOidcConfiguration *VaultOidcConfiguration
}

type KMSOptions struct {
	KeyRegion   string `jsonapi:"attr,key-region,omitempty"`   // AWS KMS
	KeyLocation string `jsonapi:"attr,key-location,omitempty"` // GCP KMS
	KeyRingID   string `jsonapi:"attr,key-ring-id,omitempty"`  // GCP KMS
}

type HyokConfigurationsCustomerKeyVersion struct {
	ID   string `jsonapi:"primary,hyok-customer-key-versions"`
	Type string `jsonapi:"attr,type"`
}

type HyokConfiguration struct {
	ID string `jsonapi:"primary,hyok-configurations"`

	// Attributes
	KekID      string      `jsonapi:"attr,kek-id"`
	KMSOptions *KMSOptions `jsonapi:"attr,kms-options"`
	Name       string      `jsonapi:"attr,name"`
	Primary    bool        `jsonapi:"attr,primary"`
	Status     string      `jsonapi:"attr,status"`
	Error      *string     `jsonapi:"attr,error"`

	// Relationships
	Organization      *Organization            `jsonapi:"relation,organization"`
	OidcConfiguration *OidcConfigurationChoice `jsonapi:"polyrelation,oidc-configuration"`
	AgentPool         *AgentPool               `jsonapi:"relation,agent-pool"`
}

type HyokConfigurationsList struct {
	*Pagination
	Items []*HyokConfiguration
}

type HyokConfigurationsSortColumn string

const (
	// HyokConfigurationSortByName sorts by the name attribute.
	HyokConfigurationSortByName HyokConfigurationsSortColumn = "name"

	// HyokConfigurationSortByUpdatedAt sorts by the updated-at attribute.
	HyokConfigurationSortByUpdatedAt HyokConfigurationsSortColumn = "updated-at"

	// HyokConfigurationSortByNameDesc sorts by the name attribute in descending order.
	HyokConfigurationSortByNameDesc HyokConfigurationsSortColumn = "-name"

	// HyokConfigurationSortByUpdatedAtDesc sorts by the updated-at attribute in descending order.
	HyokConfigurationSortByUpdatedAtDesc HyokConfigurationsSortColumn = "-updated-at"
)

type HyokConfigurationsIncludeOpt string

const (
	HyokConfigurationsIncludeHyokCustomerKeyVersions HyokConfigurationsIncludeOpt = "hyok_customer_key_versions"
	HyokConfigurationsIncludeOidcCconfiguration      HyokConfigurationsIncludeOpt = "oidc_configuration"
)

type HyokConfigurationsListOptions struct {
	ListOptions
	ProjectID    string                         `url:"filter[project[id]],omitempty"`
	Sort         HyokConfigurationsSortColumn   `url:"sort,omitempty"`
	SearchByName string                         `url:"search[name],omitempty"`
	Include      []HyokConfigurationsIncludeOpt `url:"include,omitempty"`
}

type HyokConfigurationsReadOptions struct {
	Include []HyokConfigurationsIncludeOpt `url:"include,omitempty"`
}

type HyokConfigurationsCreateOptions struct {
	ID   string `jsonapi:"primary,hyok-configurations"`
	Type string `jsonapi:"attr,type"`

	// Attributes
	KekID      string      `jsonapi:"attr,kek-id"`
	KMSOptions *KMSOptions `jsonapi:"attr,kms-options"`
	Name       string      `jsonapi:"attr,name"`
	Primary    bool        `jsonapi:"attr,primary"`
	Status     string      `jsonapi:"attr,status"`
	Error      *string     `jsonapi:"attr,error"`

	// Relationships
	Organization      *Organization            `jsonapi:"relation,organization"`
	OidcConfiguration *OidcConfigurationChoice `jsonapi:"polyrelation,oidc-configuration"`
	AgentPool         *AgentPool               `jsonapi:"relation,agent-pool"`
}

type HyokConfigurationsUpdateOptions struct {
	ID   string `jsonapi:"primary,hyok-configurations"`
	Type string `jsonapi:"attr,type"`

	// Attributes
	KekID      string      `jsonapi:"attr,kek-id"`
	KMSOptions *KMSOptions `jsonapi:"attr,kms-options"`
	Name       string      `jsonapi:"attr,name"`
	Primary    bool        `jsonapi:"attr,primary"`
	Status     string      `jsonapi:"attr,status"`
	Error      *string     `jsonapi:"attr,error"`

	// Relationships
	Organization      *Organization            `jsonapi:"relation,organization"`
	OidcConfiguration *OidcConfigurationChoice `jsonapi:"polyrelation,oidc-configuration"`
	AgentPool         *AgentPool               `jsonapi:"relation,agent-pool"`
}

func (h *HyokConfigurationsListOptions) valid() error {
	return nil
}

func (h hyokConfigurations) List(ctx context.Context, organization string, options *HyokConfigurationsListOptions) (*HyokConfigurationsList, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}

	if err := options.valid(); err != nil {
		return nil, err
	}

	req, err := h.client.NewRequest("GET", fmt.Sprintf("organizations/%s/hyok-configurations", organization), options)
	if err != nil {
		return nil, err
	}

	hyokConfigurationList := &HyokConfigurationsList{}
	err = req.Do(ctx, hyokConfigurationList)
	if err != nil {
		return nil, err
	}

	return hyokConfigurationList, nil
}

func (h hyokConfigurations) TestUnpersisted(ctx context.Context, organization string) error {
	if !validStringID(&organization) {
		return ErrInvalidOrg
	}

	req, err := h.client.NewRequest("POST", fmt.Sprintf("organizations/%s/hyok-configurations/test", organization), nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

func (h *HyokConfigurationsReadOptions) valid() error {
	return nil
}

func (h hyokConfigurations) Read(ctx context.Context, hyokID string, options *HyokConfigurationsReadOptions) (*HyokConfiguration, error) {
	if !validStringID(&hyokID) {
		return nil, ErrInvalidHyok
	}

	if err := options.valid(); err != nil {
		return nil, err
	}

	req, err := h.client.NewRequest("GET", fmt.Sprintf("hyok-configurations/%s", url.PathEscape(hyokID)), options)
	if err != nil {
		return nil, err
	}

	hyokConfiguration := &HyokConfiguration{}
	err = req.Do(ctx, hyokConfiguration)
	if err != nil {
		return nil, err
	}

	return hyokConfiguration, nil
}

func (h *HyokConfigurationsCreateOptions) valid() error {
	if h.KekID == "" {
		return ErrRequiredKekID
	}

	if h.Name == "" {
		return ErrRequiredName
	}

	if h.OidcConfiguration == nil {
		return ErrRequiredOIDCConfiguration
	}

	if h.AgentPool == nil {
		return ErrRequiredAgentPool
	}

	if h.OidcConfiguration.AwsOidcConfiguration != nil {
		if h.KMSOptions == nil {
			return ErrRequiredKMSOptions
		}

		if h.KMSOptions.KeyRegion == "" {
			return ErrRequiredKMSOptionsKeyRegion
		}
	}

	if h.OidcConfiguration.GcpOidcConfiguration != nil {
		if h.KMSOptions == nil {
			return ErrRequiredKMSOptions
		}

		if h.KMSOptions.KeyLocation == "" {
			return ErrRequiredKMSOptionsKeyLocation
		}

		if h.KMSOptions.KeyRingID == "" {
			return ErrRequiredKMSOptionsKeyRingID
		}
	}

	return nil
}

func (h hyokConfigurations) Create(ctx context.Context, organization string, options HyokConfigurationsCreateOptions) (*HyokConfiguration, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}

	if err := options.valid(); err != nil {
		return nil, err
	}

	req, err := h.client.NewRequest("POST", fmt.Sprintf("organizations/%s/hyok-configurations", organization), &options)
	if err != nil {
		return nil, err
	}

	hyokConfiguration := &HyokConfiguration{}
	err = req.Do(ctx, hyokConfiguration)
	if err != nil {
		return nil, err
	}

	return hyokConfiguration, nil
}

func (h *HyokConfigurationsUpdateOptions) valid() error {
	return nil
}

func (h hyokConfigurations) Update(ctx context.Context, hyokID string, options HyokConfigurationsUpdateOptions) (*HyokConfiguration, error) {
	if !validStringID(&hyokID) {
		return nil, ErrInvalidHyok
	}

	if err := options.valid(); err != nil {
		return nil, err
	}

	req, err := h.client.NewRequest("PATCH", fmt.Sprintf("hyok-configurations/%s", url.PathEscape(hyokID)), &options)
	if err != nil {
		return nil, err
	}

	hyokConfiguration := &HyokConfiguration{}
	err = req.Do(ctx, hyokConfiguration)
	if err != nil {
		return nil, err
	}

	return hyokConfiguration, nil
}

func (h hyokConfigurations) Delete(ctx context.Context, hyokID string) error {
	if !validStringID(&hyokID) {
		return ErrInvalidHyok
	}

	req, err := h.client.NewRequest("DELETE", fmt.Sprintf("hyok-configurations/%s", url.PathEscape(hyokID)), nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

func (h hyokConfigurations) Test(ctx context.Context, hyokID string) error {
	if !validStringID(&hyokID) {
		return ErrInvalidHyok
	}

	req, err := h.client.NewRequest("POST", fmt.Sprintf("hyok-configurations/%s/actions/test", url.PathEscape(hyokID)), nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

func (h hyokConfigurations) Revoke(ctx context.Context, hyokID string) error {
	if !validStringID(&hyokID) {
		return ErrInvalidHyok
	}

	req, err := h.client.NewRequest("POST", fmt.Sprintf("hyok-configurations/%s/actions/revoke", url.PathEscape(hyokID)), nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}
