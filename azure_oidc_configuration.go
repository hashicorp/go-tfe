package tfe

import (
	"context"
	"fmt"
	"net/url"
)

type AzureOIDCConfigurations interface {
	Create(ctx context.Context, organization string, options AzureOIDCConfigurationsCreateOptions) (*AzureOIDCConfiguration, error)

	Read(ctx context.Context, oidcID string) (*AzureOIDCConfiguration, error)

	Update(ctx context.Context, oidcID string, options AzureOIDCConfigurationsUpdateOptions) (*AzureOIDCConfiguration, error)

	Delete(ctx context.Context, oidcID string) error
}

type azureOIDCConfigurations struct {
	client *Client
}

var _ AzureOIDCConfigurations = &azureOIDCConfigurations{}

type AzureOIDCConfiguration struct {
	ID             string `jsonapi:"primary,azure-oidc-configurations"`
	ClientID       string `jsonapi:"attr,client-id"`
	SubscriptionID string `jsonapi:"attr,subscription-id"`
	TenantID       string `jsonapi:"attr,tenant-id"`

	Organization *Organization `jsonapi:"relation,organization"`
}

type AzureOIDCConfigurationsCreateOptions struct {
	ID             string `jsonapi:"primary,azure-oidc-configurations"`
	ClientID       string `jsonapi:"attr,client-id"`
	SubscriptionID string `jsonapi:"attr,subscription-id"`
	TenantID       string `jsonapi:"attr,tenant-id"`

	Organization *Organization `jsonapi:"relation,organization"`
}

type AzureOIDCConfigurationsUpdateOptions struct {
	ID             string `jsonapi:"primary,azure-oidc-configurations"`
	ClientID       string `jsonapi:"attr,client-id"`
	SubscriptionID string `jsonapi:"attr,subscription-id"`
	TenantID       string `jsonapi:"attr,tenant-id"`

	Organization *Organization `jsonapi:"relation,organization"`
}

func (o *AzureOIDCConfigurationsCreateOptions) valid() error {
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

func (aoc *azureOIDCConfigurations) Create(ctx context.Context, organization string, options AzureOIDCConfigurationsCreateOptions) (*AzureOIDCConfiguration, error) {
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

	azureOIDCConfiguration := &AzureOIDCConfiguration{}
	err = req.Do(ctx, azureOIDCConfiguration)
	if err != nil {
		return nil, err
	}

	return azureOIDCConfiguration, nil
}

func (aoc *azureOIDCConfigurations) Read(ctx context.Context, oidcID string) (*AzureOIDCConfiguration, error) {
	req, err := aoc.client.NewRequest("GET", fmt.Sprintf("oidc-configurations/%s", url.PathEscape(oidcID)), nil)
	if err != nil {
		return nil, err
	}

	azureOIDCConfiguration := &AzureOIDCConfiguration{}
	err = req.Do(ctx, azureOIDCConfiguration)
	if err != nil {
		return nil, err
	}

	return azureOIDCConfiguration, nil
}

func (o *AzureOIDCConfigurationsUpdateOptions) valid() error {
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

func (aoc *azureOIDCConfigurations) Update(ctx context.Context, oidcID string, options AzureOIDCConfigurationsUpdateOptions) (*AzureOIDCConfiguration, error) {
	if !validStringID(&oidcID) {
		return nil, ErrInvalidOIDC
	}

	if err := options.valid(); err != nil {
		return nil, err
	}

	req, err := aoc.client.NewRequest("PATCH", fmt.Sprintf("oidc-configurations/%s", url.PathEscape(oidcID)), &options)
	if err != nil {
		return nil, err
	}

	azureOIDCConfiguration := &AzureOIDCConfiguration{}
	err = req.Do(ctx, azureOIDCConfiguration)
	if err != nil {
		return nil, err
	}

	return azureOIDCConfiguration, nil
}

func (aoc *azureOIDCConfigurations) Delete(ctx context.Context, oidcID string) error {
	if !validStringID(&oidcID) {
		return ErrInvalidOIDC
	}

	req, err := aoc.client.NewRequest("DELETE", fmt.Sprintf("oidc-configurations/%s", url.PathEscape(oidcID)), nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}
