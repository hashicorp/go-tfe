package tfe

import (
	"context"
	"fmt"
	"net/url"
)

const OIDCConfigPathFormat = "oidc-configurations/%s"

type AWSOIDCConfigurations interface {
	Create(ctx context.Context, organization string, options AWSOIDCConfigurationCreateOptions) (*AWSOIDCConfiguration, error)

	Read(ctx context.Context, oidcID string) (*AWSOIDCConfiguration, error)

	Update(ctx context.Context, oidcID string, options AWSOIDCConfigurationUpdateOptions) (*AWSOIDCConfiguration, error)

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
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,aws-oidc-configurations"`

	// Attributes
	RoleARN string `jsonapi:"attr,role-arn"`
}

type AWSOIDCConfigurationUpdateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,aws-oidc-configurations"`

	// Attributes
	RoleARN string `jsonapi:"attr,role-arn"`
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
	req, err := aoc.client.NewRequest("GET", fmt.Sprintf(OIDCConfigPathFormat, url.PathEscape(oidcID)), nil)
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

	req, err := aoc.client.NewRequest("PATCH", fmt.Sprintf(OIDCConfigPathFormat, url.PathEscape(oidcID)), &options)
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

	req, err := aoc.client.NewRequest("DELETE", fmt.Sprintf(OIDCConfigPathFormat, url.PathEscape(oidcID)), nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}
