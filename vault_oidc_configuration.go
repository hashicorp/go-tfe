package tfe

import (
	"context"
	"fmt"
	"net/url"
)

type VaultOidcConfigurations interface {
	Create(ctx context.Context, organization string, options VaultOidcConfigurationsCreateOptions) (*VaultOidcConfiguration, error)

	Read(ctx context.Context, oidcID string, options *VaultOidcConfigurationsReadOptions) (*VaultOidcConfiguration, error)

	Update(ctx context.Context, oidcID string, options VaultOidcConfigurationsUpdateOptions) (*VaultOidcConfiguration, error)

	Delete(ctx context.Context, oidcID string) error
}

type vaultOidcConfigurations struct {
	client *Client
}

var _ VaultOidcConfigurations = &vaultOidcConfigurations{}

type VaultOidcConfiguration struct {
	Type             string `jsonapi:"primary,vault-oidc-configurations"`
	Address          string `jsonapi:"attr,address"`
	RoleName         string `jsonapi:"attr,role"`
	Namespace        string `jsonapi:"attr,namespace"`
	JWTAuthPath      string `jsonapi:"attr,auth_path"`
	TLSCACertificate string `jsonapi:"attr,encoded_cacert"`

	Organization *Organization `jsonapi:"relation,organization"`
}

type VaultOidcConfigurationsCreateOptions struct {
	Type             string `jsonapi:"primary,vault-oidc-configurations"`
	Address          string `jsonapi:"attr,address"`
	RoleName         string `jsonapi:"attr,role"`
	Namespace        string `jsonapi:"attr,namespace"`
	JWTAuthPath      string `jsonapi:"attr,auth_path"`
	TLSCACertificate string `jsonapi:"attr,encoded_cacert"`

	Organization *Organization `jsonapi:"relation,organization"`
}

type VaultOidcConfigurationsIncludeOpt string

const (
	VaultOidcConfigurationsIncludeOrganization            VaultOidcConfigurationsIncludeOpt = "organization"
	VaultOidcConfigurationsIncludeProject                 VaultOidcConfigurationsIncludeOpt = "project"
	VaultOidcConfigurationsIncludeLatestHyokConfiguration VaultOidcConfigurationsIncludeOpt = "latest_hyok_configuration"
	VaultOidcConfigurationsIncludeHyokDiagnostics         VaultOidcConfigurationsIncludeOpt = "hyok_diagnostics"
)

type VaultOidcConfigurationsReadOptions struct {
	Include []VaultOidcConfigurationsIncludeOpt `url:"include,omitempty"`
}

type VaultOidcConfigurationsUpdateOptions struct {
	Type             string `jsonapi:"primary,vault-oidc-configurations"`
	Address          string `jsonapi:"attr,address"`
	RoleName         string `jsonapi:"attr,role"`
	Namespace        string `jsonapi:"attr,namespace"`
	JWTAuthPath      string `jsonapi:"attr,auth_path"`
	TLSCACertificate string `jsonapi:"attr,encoded_cacert"`

	Organization *Organization `jsonapi:"relation,organization"`
}

func (o *VaultOidcConfigurationsCreateOptions) valid() error {
	if o.Address == "" {
		return ErrRequiredAddress
	}

	if o.RoleName == "" {
		return ErrRequiredRoleName
	}

	return nil
}

func (voc *vaultOidcConfigurations) Create(ctx context.Context, organization string, options VaultOidcConfigurationsCreateOptions) (*VaultOidcConfiguration, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}

	if err := options.valid(); err != nil {
		return nil, err
	}

	req, err := voc.client.NewRequest("POST", fmt.Sprintf("organizations/%s/oidc-configurations", organization), &options)
	if err != nil {
		return nil, err
	}

	vaultOidcConfiguration := &VaultOidcConfiguration{}
	err = req.Do(ctx, vaultOidcConfiguration)
	if err != nil {
		return nil, err
	}

	return vaultOidcConfiguration, nil
}

func (o *VaultOidcConfigurationsReadOptions) valid() error {
	return nil
}

func (voc *vaultOidcConfigurations) Read(ctx context.Context, oidcID string, options *VaultOidcConfigurationsReadOptions) (*VaultOidcConfiguration, error) {
	if options != nil {
		if err := options.valid(); err != nil {
			return nil, err
		}
	}

	req, err := voc.client.NewRequest("GET", fmt.Sprintf("oidc-configurations/%s", url.PathEscape(oidcID)), options)
	if err != nil {
		return nil, err
	}

	vaultOidcConfiguration := &VaultOidcConfiguration{}
	err = req.Do(ctx, vaultOidcConfiguration)
	if err != nil {
		return nil, err
	}

	return vaultOidcConfiguration, nil
}

func (o *VaultOidcConfigurationsUpdateOptions) valid() error {
	if o.Address == "" {
		return ErrRequiredAddress
	}

	if o.RoleName == "" {
		return ErrRequiredRoleName
	}

	return nil
}

func (voc *vaultOidcConfigurations) Update(ctx context.Context, oidcID string, options VaultOidcConfigurationsUpdateOptions) (*VaultOidcConfiguration, error) {
	if !validStringID(&oidcID) {
		return nil, ErrInvalidOidc
	}

	if err := options.valid(); err != nil {
		return nil, err
	}

	req, err := voc.client.NewRequest("PATCH", fmt.Sprintf("oidc-configurations/%s", url.PathEscape(oidcID)), &options)
	if err != nil {
		return nil, err
	}

	vaultOidcConfiguration := &VaultOidcConfiguration{}
	err = req.Do(ctx, vaultOidcConfiguration)
	if err != nil {
		return nil, err
	}

	return vaultOidcConfiguration, nil
}

func (voc *vaultOidcConfigurations) Delete(ctx context.Context, oidcID string) error {
	if !validStringID(&oidcID) {
		return ErrInvalidOidc
	}

	req, err := voc.client.NewRequest("DELETE", fmt.Sprintf("oidc-configurations/%s", url.PathEscape(oidcID)), nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}
