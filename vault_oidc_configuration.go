package tfe

import (
	"context"
	"fmt"
	"net/url"
)

type VaultOIDCConfigurations interface {
	Create(ctx context.Context, organization string, options VaultOIDCConfigurationCreateOptions) (*VaultOIDCConfiguration, error)

	Read(ctx context.Context, oidcID string) (*VaultOIDCConfiguration, error)

	Update(ctx context.Context, oidcID string, options VaultOIDCConfigurationUpdateOptions) (*VaultOIDCConfiguration, error)

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
	JWTAuthPath      string `jsonapi:"attr,auth-path"`
	TLSCACertificate string `jsonapi:"attr,encoded-cacert"`

	Organization *Organization `jsonapi:"relation,organization"`
}

type VaultOIDCConfigurationCreateOptions struct {
	ID               string `jsonapi:"primary,vault-oidc-configurations"`
	Address          string `jsonapi:"attr,address"`
	RoleName         string `jsonapi:"attr,role"`
	Namespace        string `jsonapi:"attr,namespace"`
	JWTAuthPath      string `jsonapi:"attr,auth-path"`
	TLSCACertificate string `jsonapi:"attr,encoded-cacert"`
}

type VaultOIDCConfigurationUpdateOptions struct {
	ID               string `jsonapi:"primary,vault-oidc-configurations"`
	Address          string `jsonapi:"attr,address"`
	RoleName         string `jsonapi:"attr,role"`
	Namespace        string `jsonapi:"attr,namespace"`
	JWTAuthPath      string `jsonapi:"attr,auth-path"`
	TLSCACertificate string `jsonapi:"attr,encoded-cacert"`
}

func (o *VaultOIDCConfigurationCreateOptions) valid() error {
	if o.Address == "" {
		return ErrRequiredVaultAddress
	}

	if o.RoleName == "" {
		return ErrRequiredRoleName
	}

	return nil
}

func (voc *vaultOIDCConfigurations) Create(ctx context.Context, organization string, options VaultOIDCConfigurationCreateOptions) (*VaultOIDCConfiguration, error) {
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
	req, err := voc.client.NewRequest("GET", fmt.Sprintf(OIDCConfigPathFormat, url.PathEscape(oidcID)), nil)
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

func (o *VaultOIDCConfigurationUpdateOptions) valid() error {
	if o.Address == "" {
		return ErrRequiredVaultAddress
	}

	if o.RoleName == "" {
		return ErrRequiredRoleName
	}

	return nil
}

func (voc *vaultOIDCConfigurations) Update(ctx context.Context, oidcID string, options VaultOIDCConfigurationUpdateOptions) (*VaultOIDCConfiguration, error) {
	if !validStringID(&oidcID) {
		return nil, ErrInvalidOIDC
	}

	if err := options.valid(); err != nil {
		return nil, err
	}

	req, err := voc.client.NewRequest("PATCH", fmt.Sprintf(OIDCConfigPathFormat, url.PathEscape(oidcID)), &options)
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

	req, err := voc.client.NewRequest("DELETE", fmt.Sprintf(OIDCConfigPathFormat, url.PathEscape(oidcID)), nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}
