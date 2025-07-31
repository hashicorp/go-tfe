package tfe

import (
	"context"
	"fmt"
	"net/url"
)

type GCPOIDCConfigurations interface {
	Create(ctx context.Context, organization string, options GCPOIDCConfigurationsCreateOptions) (*GcpOIDCConfiguration, error)

	Read(ctx context.Context, oidcID string) (*GcpOIDCConfiguration, error)

	Update(ctx context.Context, oidcID string, options GCPOIDCConfigurationsUpdateOptions) (*GcpOIDCConfiguration, error)

	Delete(ctx context.Context, oidcID string) error
}

type gcpOIDCConfigurations struct {
	client *Client
}

var _ GCPOIDCConfigurations = &gcpOIDCConfigurations{}

type GcpOIDCConfiguration struct {
	ID                   string `jsonapi:"primary,gcp-oidc-configurations"`
	ServiceAccountEmail  string `jsonapi:"attr,service-account-email"`
	ProjectNumber        string `jsonapi:"attr,project-number"`
	WorkloadProviderName string `jsonapi:"attr,workload-provider-name"`

	Organization *Organization `jsonapi:"relation,organization"`
}

type GCPOIDCConfigurationsCreateOptions struct {
	ID                   string `jsonapi:"primary,gcp-oidc-configurations"`
	ServiceAccountEmail  string `jsonapi:"attr,service-account-email"`
	ProjectNumber        string `jsonapi:"attr,project-number"`
	WorkloadProviderName string `jsonapi:"attr,workload-provider-name"`

	Organization *Organization `jsonapi:"relation,organization"`
}

type GCPOIDCConfigurationsUpdateOptions struct {
	ID                   string `jsonapi:"primary,gcp-oidc-configurations"`
	ServiceAccountEmail  string `jsonapi:"attr,service-account-email"`
	ProjectNumber        string `jsonapi:"attr,project-number"`
	WorkloadProviderName string `jsonapi:"attr,workload-provider-name"`

	Organization *Organization `jsonapi:"relation,organization"`
}

func (o *GCPOIDCConfigurationsCreateOptions) valid() error {
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

func (goc *gcpOIDCConfigurations) Create(ctx context.Context, organization string, options GCPOIDCConfigurationsCreateOptions) (*GcpOIDCConfiguration, error) {
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

	gcpOIDCConfiguration := &GcpOIDCConfiguration{}
	err = req.Do(ctx, gcpOIDCConfiguration)
	if err != nil {
		return nil, err
	}

	return gcpOIDCConfiguration, nil
}

func (goc *gcpOIDCConfigurations) Read(ctx context.Context, oidcID string) (*GcpOIDCConfiguration, error) {
	req, err := goc.client.NewRequest("GET", fmt.Sprintf("oidc-configurations/%s", url.PathEscape(oidcID)), nil)
	if err != nil {
		return nil, err
	}

	gcpOIDCConfiguration := &GcpOIDCConfiguration{}
	err = req.Do(ctx, gcpOIDCConfiguration)
	if err != nil {
		return nil, err
	}

	return gcpOIDCConfiguration, nil
}

func (o *GCPOIDCConfigurationsUpdateOptions) valid() error {
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

func (goc *gcpOIDCConfigurations) Update(ctx context.Context, oidcID string, options GCPOIDCConfigurationsUpdateOptions) (*GcpOIDCConfiguration, error) {
	if !validStringID(&oidcID) {
		return nil, ErrInvalidOIDC
	}

	if err := options.valid(); err != nil {
		return nil, err
	}

	req, err := goc.client.NewRequest("PATCH", fmt.Sprintf("oidc-configurations/%s", url.PathEscape(oidcID)), &options)
	if err != nil {
		return nil, err
	}

	gcpOIDCConfiguration := &GcpOIDCConfiguration{}
	err = req.Do(ctx, gcpOIDCConfiguration)
	if err != nil {
		return nil, err
	}

	return gcpOIDCConfiguration, nil
}

func (goc *gcpOIDCConfigurations) Delete(ctx context.Context, oidcID string) error {
	if !validStringID(&oidcID) {
		return ErrInvalidOIDC
	}

	req, err := goc.client.NewRequest("DELETE", fmt.Sprintf("oidc-configurations/%s", url.PathEscape(oidcID)), nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}
