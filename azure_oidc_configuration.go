package tfe

import (
	"context"
	"fmt"
	"net/url"
)

type AzureOidcConfigurations interface {
	Create(ctx context.Context, organization string, options AzureOidcConfigurationsCreateOptions) (*AzureOidcConfiguration, error)

	Read(ctx context.Context, oidcID string, options *AzureOidcConfigurationsReadOptions) (*AzureOidcConfiguration, error)

	Update(ctx context.Context, oidcID string, options AzureOidcConfigurationsUpdateOptions) (*AzureOidcConfiguration, error)

	Delete(ctx context.Context, oidcID string) error
}

type azureOidcConfigurations struct {
	client *Client
}

var _ AzureOidcConfigurations = &azureOidcConfigurations{}

type AzureOidcConfiguration struct {
	Type           string `jsonapi:"primary,azure-oidc-configurations"`
	ClientID       string `jsonapi:"attr,client-id"`
	SubscriptionID string `jsonapi:"attr,subscription-id"`
	TenantID       string `jsonapi:"attr,tenant-id"`

	Organization *Organization `jsonapi:"relation,organization"`
}

type AzureOidcConfigurationsCreateOptions struct {
	Type           string `jsonapi:"primary,azure-oidc-configurations"`
	ClientID       string `jsonapi:"attr,client-id"`
	SubscriptionID string `jsonapi:"attr,subscription-id"`
	TenantID       string `jsonapi:"attr,tenant-id"`

	Organization *Organization `jsonapi:"relation,organization"`
}

type AzureOidcConfigurationsIncludeOpt string

const (
	AzureOidcConfigurationsIncludeOrganization            AzureOidcConfigurationsIncludeOpt = "organization"
	AzureOidcConfigurationsIncludeProject                 AzureOidcConfigurationsIncludeOpt = "project"
	AzureOidcConfigurationsIncludeLatestHyokConfiguration AzureOidcConfigurationsIncludeOpt = "latest_hyok_configuration"
	AzureOidcConfigurationsIncludeHyokDiagnostics         AzureOidcConfigurationsIncludeOpt = "hyok_diagnostics"
)

type AzureOidcConfigurationsReadOptions struct {
	Include []AzureOidcConfigurationsIncludeOpt `url:"include,omitempty"`
}

type AzureOidcConfigurationsUpdateOptions struct {
	Type           string `jsonapi:"primary,azure-oidc-configurations"`
	ClientID       string `jsonapi:"attr,client-id"`
	SubscriptionID string `jsonapi:"attr,subscription-id"`
	TenantID       string `jsonapi:"attr,tenant-id"`

	Organization *Organization `jsonapi:"relation,organization"`
}

func (o *AzureOidcConfigurationsCreateOptions) valid() error {
	if o.ClientID == "" {
		return ErrRequiredClientID
	}

	if o.SubscriptionID == "" {
		return ErrRequiredSubscriptionID
	}

	if o.TenantID == "" {
		return ErrRequiredTenantID
	}

	return nil
}

func (aoc *azureOidcConfigurations) Create(ctx context.Context, organization string, options AzureOidcConfigurationsCreateOptions) (*AzureOidcConfiguration, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}

	if err := options.valid(); err != nil {
		return nil, err
	}

	req, err := aoc.client.NewRequest("POST", fmt.Sprintf("organizations/%s/oidc-configurations", organization), &options)
	if err != nil {
		return nil, err
	}

	azureOidcConfiguration := &AzureOidcConfiguration{}
	err = req.Do(ctx, azureOidcConfiguration)
	if err != nil {
		return nil, err
	}

	return azureOidcConfiguration, nil
}

func (o *AzureOidcConfigurationsReadOptions) valid() error {
	return nil
}

func (aoc *azureOidcConfigurations) Read(ctx context.Context, oidcID string, options *AzureOidcConfigurationsReadOptions) (*AzureOidcConfiguration, error) {
	if options != nil {
		if err := options.valid(); err != nil {
			return nil, err
		}
	}

	req, err := aoc.client.NewRequest("GET", fmt.Sprintf("oidc-configurations/%s", url.PathEscape(oidcID)), options)
	if err != nil {
		return nil, err
	}

	azureOidcConfiguration := &AzureOidcConfiguration{}
	err = req.Do(ctx, azureOidcConfiguration)
	if err != nil {
		return nil, err
	}

	return azureOidcConfiguration, nil
}

func (o *AzureOidcConfigurationsUpdateOptions) valid() error {
	if o.ClientID == "" {
		return ErrRequiredClientID
	}

	if o.SubscriptionID == "" {
		return ErrRequiredSubscriptionID
	}

	if o.TenantID == "" {
		return ErrRequiredTenantID
	}

	return nil
}

func (aoc *azureOidcConfigurations) Update(ctx context.Context, oidcID string, options AzureOidcConfigurationsUpdateOptions) (*AzureOidcConfiguration, error) {
	if !validStringID(&oidcID) {
		return nil, ErrInvalidOidc
	}

	if err := options.valid(); err != nil {
		return nil, err
	}

	req, err := aoc.client.NewRequest("PATCH", fmt.Sprintf("oidc-configurations/%s", url.PathEscape(oidcID)), &options)
	if err != nil {
		return nil, err
	}

	azureOidcConfiguration := &AzureOidcConfiguration{}
	err = req.Do(ctx, azureOidcConfiguration)
	if err != nil {
		return nil, err
	}

	return azureOidcConfiguration, nil
}

func (aoc *azureOidcConfigurations) Delete(ctx context.Context, oidcID string) error {
	if !validStringID(&oidcID) {
		return ErrInvalidOidc
	}

	req, err := aoc.client.NewRequest("DELETE", fmt.Sprintf("oidc-configurations/%s", url.PathEscape(oidcID)), nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}
