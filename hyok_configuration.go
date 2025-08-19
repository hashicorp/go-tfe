package tfe

import (
	"context"
	"fmt"
	"net/url"
)

type HYOKConfigurations interface {
	List(ctx context.Context, organization string, options *HYOKConfigurationsListOptions) (*HYOKConfigurationsList, error)

	Create(ctx context.Context, organization string, options HYOKConfigurationsCreateOptions) (*HYOKConfiguration, error)

	Read(ctx context.Context, hyokID string, options *HYOKConfigurationsReadOptions) (*HYOKConfiguration, error)

	Update(ctx context.Context, hyokID string, options HYOKConfigurationsUpdateOptions) (*HYOKConfiguration, error)

	Delete(ctx context.Context, hyokID string) error

	Test(ctx context.Context, hyokID string) error

	Revoke(ctx context.Context, hyokID string) error
}

type hyokConfigurations struct {
	client *Client
}

var _ HYOKConfigurations = &hyokConfigurations{}

type HYOKConfigurationStatus string

const (
	HYOKConfigurationUntested   HYOKConfigurationStatus = "untested"
	HYOKConfigurationTesting    HYOKConfigurationStatus = "testing"
	HYOKConfigurationTestFailed HYOKConfigurationStatus = "test_failed"
	HYOKConfigurationAvailable  HYOKConfigurationStatus = "available"
	HYOKConfigurationErrored    HYOKConfigurationStatus = "errored"
	HYOKConfigurationRevoking   HYOKConfigurationStatus = "revoking"
	HYOKConfigurationRevoked    HYOKConfigurationStatus = "revoked"
)

type OIDCConfigurationType struct {
	AWSOIDCConfiguration   *AWSOIDCConfiguration
	GCPOIDCConfiguration   *GCPOIDCConfiguration
	AzureOIDCConfiguration *AzureOIDCConfiguration
	VaultOIDCConfiguration *VaultOIDCConfiguration
}

type KMSOptions struct {
	// AWS
	KeyRegion string `jsonapi:"attr,key-region,omitempty"`
	// GCP
	KeyLocation string `jsonapi:"attr,key-location,omitempty"`
	KeyRingID   string `jsonapi:"attr,key-ring-id,omitempty"`
}

type HYOKConfiguration struct {
	ID string `jsonapi:"primary,hyok-configurations"`

	// Attributes
	KEKID      string                  `jsonapi:"attr,kek-id"`
	KMSOptions *KMSOptions             `jsonapi:"attr,kms-options,omitempty"`
	Name       string                  `jsonapi:"attr,name"`
	Primary    bool                    `jsonapi:"attr,primary"`
	Status     HYOKConfigurationStatus `jsonapi:"attr,status"`
	Error      *string                 `jsonapi:"attr,error"`

	// Relationships
	Organization      *Organization          `jsonapi:"relation,organization"`
	OIDCConfiguration *OIDCConfigurationType `jsonapi:"polyrelation,oidc-configuration"`
	AgentPool         *AgentPool             `jsonapi:"relation,agent-pool"`
}

type HYOKConfigurationsList struct {
	*Pagination
	Items []*HYOKConfiguration
}

type HYOKConfigurationsIncludeOpt string

const (
	HYOKConfigurationsIncludeHYOKCustomerKeyVersions HYOKConfigurationsIncludeOpt = "hyok_customer_key_versions"
	HYOKConfigurationsIncludeOIDCConfiguration       HYOKConfigurationsIncludeOpt = "oidc_configuration"
)

type HYOKConfigurationsListOptions struct {
	ListOptions
	SearchQuery string                         `url:"q,omitempty"`
	Include     []HYOKConfigurationsIncludeOpt `url:"include,omitempty"`
}

type HYOKConfigurationsCreateOptions struct {
	ID string `jsonapi:"primary,hyok-configurations"`

	// Attributes
	KEKID      string      `jsonapi:"attr,kek-id"`
	KMSOptions *KMSOptions `jsonapi:"attr,kms-options"`
	Name       string      `jsonapi:"attr,name"`

	// Relationships
	Organization      *Organization          `jsonapi:"relation,organization"`
	OIDCConfiguration *OIDCConfigurationType `jsonapi:"polyrelation,oidc-configuration"`
	AgentPool         *AgentPool             `jsonapi:"relation,agent-pool"`
}

type HYOKConfigurationsReadOptions struct {
	Include []HYOKConfigurationsIncludeOpt `url:"include,omitempty"`
}

type HYOKConfigurationsUpdateOptions struct {
	ID string `jsonapi:"primary,hyok-configurations"`

	// Attributes
	KEKID      string      `jsonapi:"attr,kek-id"`
	KMSOptions *KMSOptions `jsonapi:"attr,kms-options"`
	Name       string      `jsonapi:"attr,name"`
	Primary    bool        `jsonapi:"attr,primary"`

	// Relationships
	Organization      *Organization          `jsonapi:"relation,organization"`
	OIDCConfiguration *OIDCConfigurationType `jsonapi:"polyrelation,oidc-configuration"`
	AgentPool         *AgentPool             `jsonapi:"relation,agent-pool"`
}

func (h *HYOKConfigurationsListOptions) valid() error {
	return nil
}

func (h hyokConfigurations) List(ctx context.Context, organization string, options *HYOKConfigurationsListOptions) (*HYOKConfigurationsList, error) {
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

	hyokConfigurationList := &HYOKConfigurationsList{}
	err = req.Do(ctx, hyokConfigurationList)
	if err != nil {
		return nil, err
	}

	return hyokConfigurationList, nil
}

func (h *HYOKConfigurationsReadOptions) valid() error {
	return nil
}

func (h hyokConfigurations) Read(ctx context.Context, hyokID string, options *HYOKConfigurationsReadOptions) (*HYOKConfiguration, error) {
	if !validStringID(&hyokID) {
		return nil, ErrInvalidHYOK
	}

	if err := options.valid(); err != nil {
		return nil, err
	}

	req, err := h.client.NewRequest("GET", fmt.Sprintf("hyok-configurations/%s", url.PathEscape(hyokID)), options)
	if err != nil {
		return nil, err
	}

	hyokConfiguration := &HYOKConfiguration{}
	err = req.Do(ctx, hyokConfiguration)
	if err != nil {
		return nil, err
	}

	return hyokConfiguration, nil
}

func (h *HYOKConfigurationsCreateOptions) valid() error {
	if h.KEKID == "" {
		return ErrRequiredKEKID
	}

	if h.Name == "" {
		return ErrRequiredName
	}

	if h.OIDCConfiguration == nil {
		return ErrRequiredOIDCConfiguration
	}

	if h.AgentPool == nil {
		return ErrRequiredAgentPool
	}

	if h.OIDCConfiguration.AWSOIDCConfiguration != nil {
		if h.KMSOptions == nil {
			return ErrRequiredKMSOptions
		}

		if h.KMSOptions.KeyRegion == "" {
			return ErrRequiredKMSOptionsKeyRegion
		}
	}

	if h.OIDCConfiguration.GCPOIDCConfiguration != nil {
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

func (h hyokConfigurations) Create(ctx context.Context, organization string, options HYOKConfigurationsCreateOptions) (*HYOKConfiguration, error) {
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

	hyokConfiguration := &HYOKConfiguration{}
	err = req.Do(ctx, hyokConfiguration)
	if err != nil {
		return nil, err
	}

	return hyokConfiguration, nil
}

func (h *HYOKConfigurationsUpdateOptions) valid() error {
	return nil
}

func (h hyokConfigurations) Update(ctx context.Context, hyokID string, options HYOKConfigurationsUpdateOptions) (*HYOKConfiguration, error) {
	if !validStringID(&hyokID) {
		return nil, ErrInvalidHYOK
	}

	if err := options.valid(); err != nil {
		return nil, err
	}

	req, err := h.client.NewRequest("PATCH", fmt.Sprintf("hyok-configurations/%s", url.PathEscape(hyokID)), &options)
	if err != nil {
		return nil, err
	}

	hyokConfiguration := &HYOKConfiguration{}
	err = req.Do(ctx, hyokConfiguration)
	if err != nil {
		return nil, err
	}

	return hyokConfiguration, nil
}

func (h hyokConfigurations) Delete(ctx context.Context, hyokID string) error {
	if !validStringID(&hyokID) {
		return ErrInvalidHYOK
	}

	req, err := h.client.NewRequest("DELETE", fmt.Sprintf("hyok-configurations/%s", url.PathEscape(hyokID)), nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

func (h hyokConfigurations) Test(ctx context.Context, hyokID string) error {
	if !validStringID(&hyokID) {
		return ErrInvalidHYOK
	}

	req, err := h.client.NewRequest("POST", fmt.Sprintf("hyok-configurations/%s/actions/test", url.PathEscape(hyokID)), nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

func (h hyokConfigurations) Revoke(ctx context.Context, hyokID string) error {
	if !validStringID(&hyokID) {
		return ErrInvalidHYOK
	}

	req, err := h.client.NewRequest("POST", fmt.Sprintf("hyok-configurations/%s/actions/revoke", url.PathEscape(hyokID)), nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}
