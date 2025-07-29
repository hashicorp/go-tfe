package tfe

import (
	"context"
	"fmt"
	"net/url"
)

type GcpOidcConfigurations interface {
	Create(ctx context.Context, organization string, options GcpOidcConfigurationsCreateOptions) (*GcpOidcConfiguration, error)

	Read(ctx context.Context, oidcID string) (*GcpOidcConfiguration, error)

	Update(ctx context.Context, oidcID string, options GcpOidcConfigurationsUpdateOptions) (*GcpOidcConfiguration, error)

	Delete(ctx context.Context, oidcID string) error
}

type gcpOidcConfigurations struct {
	client *Client
}

var _ GcpOidcConfigurations = &gcpOidcConfigurations{}

type GcpOidcConfiguration struct {
	ID                   string `jsonapi:"primary,gcp-oidc-configurations"`
	ServiceAccountEmail  string `jsonapi:"attr,service-account-email"`
	ProjectNumber        string `jsonapi:"attr,project-number"`
	WorkloadProviderName string `jsonapi:"attr,workload-provider-name"`

	Organization *Organization `jsonapi:"relation,organization"`
}

type GcpOidcConfigurationsCreateOptions struct {
	ID                   string `jsonapi:"primary,gcp-oidc-configurations"`
	ServiceAccountEmail  string `jsonapi:"attr,service-account-email"`
	ProjectNumber        string `jsonapi:"attr,project-number"`
	WorkloadProviderName string `jsonapi:"attr,workload-provider-name"`

	Organization *Organization `jsonapi:"relation,organization"`
}

type GcpOidcConfigurationsUpdateOptions struct {
	ID                   string `jsonapi:"primary,gcp-oidc-configurations"`
	ServiceAccountEmail  string `jsonapi:"attr,service-account-email"`
	ProjectNumber        string `jsonapi:"attr,project-number"`
	WorkloadProviderName string `jsonapi:"attr,workload-provider-name"`

	Organization *Organization `jsonapi:"relation,organization"`
}

func (o *GcpOidcConfigurationsCreateOptions) valid() error {
	if o.ServiceAccountEmail == "" {
		return ErrRequiredServiceAccountEmail
	}

	if o.ProjectNumber == "" {
		return ErrRequiredProjectNumber
	}

	if o.WorkloadProviderName == "" {
		return ErrRequiredWorkloadProviderName
	}

	return nil
}

func (goc *gcpOidcConfigurations) Create(ctx context.Context, organization string, options GcpOidcConfigurationsCreateOptions) (*GcpOidcConfiguration, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}

	if err := options.valid(); err != nil {
		return nil, err
	}

	req, err := goc.client.NewRequest("POST", fmt.Sprintf("organizations/%s/oidc-configurations", organization), &options)
	if err != nil {
		return nil, err
	}

	gcpOidcConfiguration := &GcpOidcConfiguration{}
	err = req.Do(ctx, gcpOidcConfiguration)
	if err != nil {
		return nil, err
	}

	return gcpOidcConfiguration, nil
}

func (goc *gcpOidcConfigurations) Read(ctx context.Context, oidcID string) (*GcpOidcConfiguration, error) {
	req, err := goc.client.NewRequest("GET", fmt.Sprintf("oidc-configurations/%s", url.PathEscape(oidcID)), nil)
	if err != nil {
		return nil, err
	}

	gcpOidcConfiguration := &GcpOidcConfiguration{}
	err = req.Do(ctx, gcpOidcConfiguration)
	if err != nil {
		return nil, err
	}

	return gcpOidcConfiguration, nil
}

func (o *GcpOidcConfigurationsUpdateOptions) valid() error {
	if o.ServiceAccountEmail == "" {
		return ErrRequiredServiceAccountEmail
	}

	if o.ProjectNumber == "" {
		return ErrRequiredProjectNumber
	}

	if o.WorkloadProviderName == "" {
		return ErrRequiredWorkloadProviderName
	}

	return nil
}

func (goc *gcpOidcConfigurations) Update(ctx context.Context, oidcID string, options GcpOidcConfigurationsUpdateOptions) (*GcpOidcConfiguration, error) {
	if !validStringID(&oidcID) {
		return nil, ErrInvalidOidc
	}

	if err := options.valid(); err != nil {
		return nil, err
	}

	req, err := goc.client.NewRequest("PATCH", fmt.Sprintf("oidc-configurations/%s", url.PathEscape(oidcID)), &options)
	if err != nil {
		return nil, err
	}

	gcpOidcConfiguration := &GcpOidcConfiguration{}
	err = req.Do(ctx, gcpOidcConfiguration)
	if err != nil {
		return nil, err
	}

	return gcpOidcConfiguration, nil
}

func (goc *gcpOidcConfigurations) Delete(ctx context.Context, oidcID string) error {
	if !validStringID(&oidcID) {
		return ErrInvalidOidc
	}

	req, err := goc.client.NewRequest("DELETE", fmt.Sprintf("oidc-configurations/%s", url.PathEscape(oidcID)), nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}
