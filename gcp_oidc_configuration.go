package tfe

import (
	"context"
	"fmt"
	"net/url"
)

// GCPOIDCConfigurations describes all the GCP OIDC configuration related methods that the HCP Terraform API supports.
// HCP Terraform API docs:
// https://developer.hashicorp.com/terraform/cloud-docs/api-docs/hold-your-own-key/oidc-configurations/gcp
type GCPOIDCConfigurations interface {
	Create(ctx context.Context, organization string, options GCPOIDCConfigurationCreateOptions) (*GCPOIDCConfiguration, error)

	Read(ctx context.Context, oidcID string) (*GCPOIDCConfiguration, error)

	Update(ctx context.Context, oidcID string, options GCPOIDCConfigurationUpdateOptions) (*GCPOIDCConfiguration, error)

	Delete(ctx context.Context, oidcID string) error
}

type gcpOIDCConfigurations struct {
	client *Client
}

var _ GCPOIDCConfigurations = &gcpOIDCConfigurations{}

type GCPOIDCConfiguration struct {
	ID                   string `jsonapi:"primary,gcp-oidc-configurations"`
	ServiceAccountEmail  string `jsonapi:"attr,service-account-email"`
	ProjectNumber        string `jsonapi:"attr,project-number"`
	WorkloadProviderName string `jsonapi:"attr,workload-provider-name"`

	Organization *Organization `jsonapi:"relation,organization"`
}

type GCPOIDCConfigurationCreateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,gcp-oidc-configurations"`

	// Attributes
	ServiceAccountEmail  string `jsonapi:"attr,service-account-email"`
	ProjectNumber        string `jsonapi:"attr,project-number"`
	WorkloadProviderName string `jsonapi:"attr,workload-provider-name"`
}

type GCPOIDCConfigurationUpdateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,gcp-oidc-configurations"`

	// Attributes
	ServiceAccountEmail  *string `jsonapi:"attr,service-account-email,omitempty"`
	ProjectNumber        *string `jsonapi:"attr,project-number,omitempty"`
	WorkloadProviderName *string `jsonapi:"attr,workload-provider-name,omitempty"`
}

func (o *GCPOIDCConfigurationCreateOptions) valid() error {
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

func (goc *gcpOIDCConfigurations) Create(ctx context.Context, organization string, options GCPOIDCConfigurationCreateOptions) (*GCPOIDCConfiguration, error) {
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

	gcpOIDCConfiguration := &GCPOIDCConfiguration{}
	err = req.Do(ctx, gcpOIDCConfiguration)
	if err != nil {
		return nil, err
	}

	return gcpOIDCConfiguration, nil
}

func (goc *gcpOIDCConfigurations) Read(ctx context.Context, oidcID string) (*GCPOIDCConfiguration, error) {
	req, err := goc.client.NewRequest("GET", fmt.Sprintf(OIDCConfigPathFormat, url.PathEscape(oidcID)), nil)
	if err != nil {
		return nil, err
	}

	gcpOIDCConfiguration := &GCPOIDCConfiguration{}
	err = req.Do(ctx, gcpOIDCConfiguration)
	if err != nil {
		return nil, err
	}

	return gcpOIDCConfiguration, nil
}

func (goc *gcpOIDCConfigurations) Update(ctx context.Context, oidcID string, options GCPOIDCConfigurationUpdateOptions) (*GCPOIDCConfiguration, error) {
	if !validStringID(&oidcID) {
		return nil, ErrInvalidOIDC
	}

	req, err := goc.client.NewRequest("PATCH", fmt.Sprintf(OIDCConfigPathFormat, url.PathEscape(oidcID)), &options)
	if err != nil {
		return nil, err
	}

	gcpOIDCConfiguration := &GCPOIDCConfiguration{}
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

	req, err := goc.client.NewRequest("DELETE", fmt.Sprintf(OIDCConfigPathFormat, url.PathEscape(oidcID)), nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}
