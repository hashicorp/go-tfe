package tfe

import (
	"context"
	"fmt"
	"net/url"
)

type HyokConfigurations interface {
	List(ctx context.Context, organization string, options *HyokConfigurationsListOptions) (*HyokConfigurationsList, error)

	Create(ctx context.Context, organization string, options HyokConfigurationsCreateOptions) (*HyokConfiguration, error)

	Read(ctx context.Context, hyokID string, options *HyokConfigurationsReadOptions) (*HyokConfiguration, error)

	Update(ctx context.Context, hyokID string, options HyokConfigurationsUpdateOptions) (*HyokConfiguration, error)
}

type hyokConfigurations struct {
	client *Client
}

var _ HyokConfigurations = &hyokConfigurations{}

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
	HyokConfigurationsIncludeOrganization            HyokConfigurationsIncludeOpt = "organization"
	HyokConfigurationsIncludeProject                 HyokConfigurationsIncludeOpt = "project"
	HyokConfigurationsIncludeLatestHyokConfiguration HyokConfigurationsIncludeOpt = "latest_hyokConfiguration"
	HyokConfigurationsIncludeHyokDiagnostics         HyokConfigurationsIncludeOpt = "hyok_diagnostics"
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
	Organization            *Organization                           `jsonapi:"relation,organization"`
	OIDCConfiguration       *OIDCConfiguration                      `jsonapi:"relation,oidc-configuration"`
	AgentPool               *AgentPool                              `jsonapi:"relation,agent-pool"`
	HyokCustomerKeyVersions []*HyokConfigurationsCustomerKeyVersion `jsonapi:"relation,hyok-customer-key-versions"`
}

type HyokConfigurationsUpdateOptions struct {
	KekID      string      `jsonapi:"attr,kek-id"`
	Name       string      `jsonapi:"attr,name"`
	KMSOptions *KMSOptions `jsonapi:"attr,kms-options"`
	Primary    bool        `jsonapi:"attr,primary"`
}

type OIDCConfiguration struct {
	ID   string `jsonapi:"attr,id"`
	Type string `jsonapi:"attr,type"`
}

type HyokConfiguration struct {
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
	Organization            *Organization                           `jsonapi:"relation,organization"`
	OIDCConfiguration       *OIDCConfiguration                      `jsonapi:"relation,oidc-configuration"`
	AgentPool               *AgentPool                              `jsonapi:"relation,agent-pool"`
	HyokCustomerKeyVersions []*HyokConfigurationsCustomerKeyVersion `jsonapi:"relation,hyok-customer-key-versions"`
}

type KMSOptions struct {
	KeyRegion   string `jsonapi:"attr,key-region"`   // AWS KMS
	KeyLocation string `jsonapi:"attr,key-location"` // GCP KMS
	KeyRingID   string `jsonapi:"attr,key-ring-id"`  // GCP KMS
}

type HyokConfigurationsCustomerKeyVersion struct {
	ID   string `jsonapi:"primary,hyok-customer-key-versions"`
	Type string `jsonapi:"attr,type"`
}

func (h *HyokConfigurationsListOptions) valid() error {
	return nil
}

func (h *HyokConfigurationsReadOptions) valid() error {
	return nil
}

func (h *HyokConfigurationsCreateOptions) valid() error {
	return nil
}

func (h *HyokConfigurationsUpdateOptions) valid() error {
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

func (h hyokConfigurations) Read(ctx context.Context, hyokID string, options *HyokConfigurationsReadOptions) (*HyokConfiguration, error) {
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

func (h hyokConfigurations) Update(ctx context.Context, hyokID string, options HyokConfigurationsUpdateOptions) (*HyokConfiguration, error) {
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
