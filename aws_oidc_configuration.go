package tfe

import (
	"context"
	"fmt"
	"net/url"
)

type AWSOIDCConfigurations interface {
	Create(ctx context.Context, organization string, options AWSOIDCConfigurationCreateOptions) (*AWSOIDCConfiguration, error)

	Read(ctx context.Context, hyokID string) (*AWSOIDCConfiguration, error)

	Update(ctx context.Context, hyokID string, options AWSOIDCConfigurationUpdateOptions) (*AWSOIDCConfiguration, error)

	Delete(ctx context.Context, oidcID string) error
}

type awsOIDCConfigurations struct {
	client *Client
}

var _ AWSOIDCConfigurations = &awsOIDCConfigurations{}

type AWSOIDCConfiguration struct {
	ID      string `jsonapi:"primary,aws-oidc-configurations"`
	RoleARN string `jsonapi:"attr,role-arn"`

	Organization *Organization `jsonapi:"relation,organization"`
}

type AWSOIDCConfigurationCreateOptions struct {
	ID      string `jsonapi:"primary,aws-oidc-configurations"`
	RoleARN string `jsonapi:"attr,role-arn"`

	Organization *Organization `jsonapi:"relation,organization"`
}

type AWSOIDCConfigurationUpdateOptions struct {
	ID      string `jsonapi:"primary,aws-oidc-configurations"`
	RoleARN string `jsonapi:"attr,role-arn"`

	Organization *Organization `jsonapi:"relation,organization"`
}

func (o *AWSOIDCConfigurationCreateOptions) valid() error {
	if o.RoleARN == "" {
		return ErrRequiredRoleARN
	}

	return nil
}

func (aoc *awsOIDCConfigurations) Create(ctx context.Context, organization string, options AWSOIDCConfigurationCreateOptions) (*AWSOIDCConfiguration, error) {
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

	awsOIDCConfiguration := &AWSOIDCConfiguration{}
	err = req.Do(ctx, awsOIDCConfiguration)
	if err != nil {
		return nil, err
	}

	return awsOIDCConfiguration, nil
}

func (aoc *awsOIDCConfigurations) Read(ctx context.Context, oidcID string) (*AWSOIDCConfiguration, error) {
	req, err := aoc.client.NewRequest("GET", fmt.Sprintf("oidc-configurations/%s", url.PathEscape(oidcID)), nil)
	if err != nil {
		return nil, err
	}

	awsOIDCConfiguration := &AWSOIDCConfiguration{}
	err = req.Do(ctx, awsOIDCConfiguration)
	if err != nil {
		return nil, err
	}

	return awsOIDCConfiguration, nil
}

func (o *AWSOIDCConfigurationUpdateOptions) valid() error {
	if o.RoleARN == "" {
		return ErrRequiredRoleARN
	}

	return nil
}

func (aoc *awsOIDCConfigurations) Update(ctx context.Context, oidcID string, options AWSOIDCConfigurationUpdateOptions) (*AWSOIDCConfiguration, error) {
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

	awsOIDCConfiguration := &AWSOIDCConfiguration{}
	err = req.Do(ctx, awsOIDCConfiguration)
	if err != nil {
		return nil, err
	}

	return awsOIDCConfiguration, nil
}

func (aoc *awsOIDCConfigurations) Delete(ctx context.Context, oidcID string) error {
	if !validStringID(&oidcID) {
		return ErrInvalidOIDC
	}

	req, err := aoc.client.NewRequest("DELETE", fmt.Sprintf("oidc-configurations/%s", url.PathEscape(oidcID)), nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}
