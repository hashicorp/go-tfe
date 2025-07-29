package tfe

import (
	"context"
	"fmt"
	"net/url"
)

type AwsOidcConfigurations interface {
	Create(ctx context.Context, organization string, options AwsOidcConfigurationsCreateOptions) (*AwsOidcConfiguration, error)

	Read(ctx context.Context, hyokID string) (*AwsOidcConfiguration, error)

	Update(ctx context.Context, hyokID string, options AwsOidcConfigurationsUpdateOptions) (*AwsOidcConfiguration, error)

	Delete(ctx context.Context, oidcID string) error
}

type awsOidcConfigurations struct {
	client *Client
}

var _ AwsOidcConfigurations = &awsOidcConfigurations{}

type AwsOidcConfiguration struct {
	ID      string `jsonapi:"primary,aws-oidc-configurations"`
	RoleArn string `jsonapi:"attr,role-arn"`

	Organization *Organization `jsonapi:"relation,organization"`
}

type AwsOidcConfigurationsCreateOptions struct {
	ID      string `jsonapi:"primary,aws-oidc-configurations"`
	RoleArn string `jsonapi:"attr,role-arn"`

	Organization *Organization `jsonapi:"relation,organization"`
}

type AwsOidcConfigurationsUpdateOptions struct {
	ID      string `jsonapi:"primary,aws-oidc-configurations"`
	RoleArn string `jsonapi:"attr,role-arn"`

	Organization *Organization `jsonapi:"relation,organization"`
}

func (o *AwsOidcConfigurationsCreateOptions) valid() error {
	if o.RoleArn == "" {
		return ErrRequiredRoleArn
	}

	return nil
}

func (aoc *awsOidcConfigurations) Create(ctx context.Context, organization string, options AwsOidcConfigurationsCreateOptions) (*AwsOidcConfiguration, error) {
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

	awsOidcConfiguration := &AwsOidcConfiguration{}
	err = req.Do(ctx, awsOidcConfiguration)
	if err != nil {
		return nil, err
	}

	return awsOidcConfiguration, nil
}

func (aoc *awsOidcConfigurations) Read(ctx context.Context, oidcID string) (*AwsOidcConfiguration, error) {
	req, err := aoc.client.NewRequest("GET", fmt.Sprintf("oidc-configurations/%s", url.PathEscape(oidcID)), nil)
	if err != nil {
		return nil, err
	}

	awsOidcConfiguration := &AwsOidcConfiguration{}
	err = req.Do(ctx, awsOidcConfiguration)
	if err != nil {
		return nil, err
	}

	return awsOidcConfiguration, nil
}

func (o *AwsOidcConfigurationsUpdateOptions) valid() error {
	if o.RoleArn == "" {
		return ErrRequiredRoleArn
	}

	return nil
}

func (aoc *awsOidcConfigurations) Update(ctx context.Context, oidcID string, options AwsOidcConfigurationsUpdateOptions) (*AwsOidcConfiguration, error) {
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

	awsOidcConfiguration := &AwsOidcConfiguration{}
	err = req.Do(ctx, awsOidcConfiguration)
	if err != nil {
		return nil, err
	}

	return awsOidcConfiguration, nil
}

func (aoc *awsOidcConfigurations) Delete(ctx context.Context, oidcID string) error {
	if !validStringID(&oidcID) {
		return ErrInvalidOidc
	}

	req, err := aoc.client.NewRequest("DELETE", fmt.Sprintf("oidc-configurations/%s", url.PathEscape(oidcID)), nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}
