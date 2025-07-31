package tfe

import (
	"context"
	"fmt"
	"net/url"
)

type VaultOIDCConfigurations interface {
	Create(ctx context.Context, organization string, options VaultOIDCConfigurationsCreateOptions) (*VaultOIDCConfiguration, error)

	Read(ctx context.Context, oidcID string) (*VaultOIDCConfiguration, error)

	Update(ctx context.Context, oidcID string, options VaultOIDCConfigurationsUpdateOptions) (*VaultOIDCConfiguration, error)

	Delete(ctx context.Context, oidcID string) error
}

type vaultOIDCConfigurations struct {
	client *Client
}

var _ VaultOIDCConfigurations = &vaultOIDCConfigurations{}

type VaultOIDCConfiguration struct {
	ID               string `jsonapi:"primary,vault-oidc-configurations"`
	Address          string `jsonapi:"attr,address"`
	RoleName         string `jsonapi:"attr,role"`
	Namespace        string `jsonapi:"attr,namespace"`
	JWTAuthPath      string `jsonapi:"attr,auth_path"`
	TLSCACertificate string `jsonapi:"attr,encoded_cacert"`

	Organization *Organization `jsonapi:"relation,organization"`
}

type VaultOIDCConfigurationsCreateOptions struct {
	ID               string `jsonapi:"primary,vault-oidc-configurations"`
	Address          string `jsonapi:"attr,address"`
	RoleName         string `jsonapi:"attr,role"`
	Namespace        string `jsonapi:"attr,namespace"`
	JWTAuthPath      string `jsonapi:"attr,auth_path"`
	TLSCACertificate string `jsonapi:"attr,encoded_cacert"`

	Organization *Organization `jsonapi:"relation,organization"`
}

type VaultOIDCConfigurationsUpdateOptions struct {
	ID               string `jsonapi:"primary,vault-oidc-configurations"`
	Address          string `jsonapi:"attr,address"`
	RoleName         string `jsonapi:"attr,role"`
	Namespace        string `jsonapi:"attr,namespace"`
	JWTAuthPath      string `jsonapi:"attr,auth_path"`
	TLSCACertificate string `jsonapi:"attr,encoded_cacert"`

	Organization *Organization `jsonapi:"relation,organization"`
}

func (o *VaultOIDCConfigurationsCreateOptions) valid() error {
	if o.Address == "" {
		return ErrRequiredAddress
	}

	if o.RoleName == "" {
		return ErrRequiredRoleName
	}

	return nil
}

func (voc *vaultOIDCConfigurations) Create(ctx context.Context, organization string, options VaultOIDCConfigurationsCreateOptions) (*VaultOIDCConfiguration, error) {
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

	vaultOIDCConfiguration := &VaultOIDCConfiguration{}
	err = req.Do(ctx, vaultOIDCConfiguration)
	if err != nil {
		return nil, err
	}

	return vaultOIDCConfiguration, nil
}

func (voc *vaultOIDCConfigurations) Read(ctx context.Context, oidcID string) (*VaultOIDCConfiguration, error) {
	req, err := voc.client.NewRequest("GET", fmt.Sprintf("oidc-configurations/%s", url.PathEscape(oidcID)), nil)
	if err != nil {
		return nil, err
	}

	vaultOIDCConfiguration := &VaultOIDCConfiguration{}
	err = req.Do(ctx, vaultOIDCConfiguration)
	if err != nil {
		return nil, err
	}

	return vaultOIDCConfiguration, nil
}

func (o *VaultOIDCConfigurationsUpdateOptions) valid() error {
	if o.Address == "" {
		return ErrRequiredAddress
	}

	if o.RoleName == "" {
		return ErrRequiredRoleName
	}

	return nil
}

func (voc *vaultOIDCConfigurations) Update(ctx context.Context, oidcID string, options VaultOIDCConfigurationsUpdateOptions) (*VaultOIDCConfiguration, error) {
	if !validStringID(&oidcID) {
		return nil, ErrInvalidOIDC
	}

	if err := options.valid(); err != nil {
		return nil, err
	}

	req, err := voc.client.NewRequest("PATCH", fmt.Sprintf("oidc-configurations/%s", url.PathEscape(oidcID)), &options)
	if err != nil {
		return nil, err
	}

	vaultOIDCConfiguration := &VaultOIDCConfiguration{}
	err = req.Do(ctx, vaultOIDCConfiguration)
	if err != nil {
		return nil, err
	}

	return vaultOIDCConfiguration, nil
}

func (voc *vaultOIDCConfigurations) Delete(ctx context.Context, oidcID string) error {
	if !validStringID(&oidcID) {
		return ErrInvalidOIDC
	}

	req, err := voc.client.NewRequest("DELETE", fmt.Sprintf("oidc-configurations/%s", url.PathEscape(oidcID)), nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}
