// Copyright IBM Corp. 2018, 2025
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"fmt"
	"net/url"
	"time"
)

var _ OrganizationTokenTTLPolicies = (*organizationTokenTTLPolicies)(nil)

type OrganizationTokenTTLPolicies interface {
	List(ctx context.Context, organization string, options *OrganizationTokenTTLPolicyListOptions) (*OrganizationTokenTTLPolicyList, error)
	Update(ctx context.Context, organization string, options OrganizationTokenTTLPolicyUpdateOptions) ([]*OrganizationTokenTTLPolicy, error)
}

type organizationTokenTTLPolicies struct {
	client *Client
}

type OrganizationTokenTTLPolicy struct {
	ID        string    `jsonapi:"primary,organization-token-ttl-policies"`
	TokenType TokenType `jsonapi:"attr,token-type"`
	MaxTTLMs  int64     `jsonapi:"attr,max-ttl-ms"`
	CreatedAt time.Time `jsonapi:"attr,created-at,iso8601"`
	UpdatedAt time.Time `jsonapi:"attr,updated-at,iso8601"`
}

type OrganizationTokenTTLPolicyList struct {
	*Pagination
	Items []*OrganizationTokenTTLPolicy
}

type OrganizationTokenTTLPolicyListOptions struct {
	ListOptions
}

type OrganizationTokenTTLPolicyUpdateItem struct {
	TokenType TokenType `jsonapi:"attr,token-type"`
	MaxTTLMs  int64     `jsonapi:"attr,max-ttl-ms"`
}

type OrganizationTokenTTLPolicyUpdateOptions struct {
	Type     string                                 `jsonapi:"primary,organization-token-ttl-policies"`
	Policies []OrganizationTokenTTLPolicyUpdateItem `jsonapi:"attr,token-ttl-policies"`
}

func (s *organizationTokenTTLPolicies) List(ctx context.Context, organization string, options *OrganizationTokenTTLPolicyListOptions) (*OrganizationTokenTTLPolicyList, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}

	u := fmt.Sprintf("organizations/%s/token-ttl-policies", url.PathEscape(organization))
	req, err := s.client.NewRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	policyList := &OrganizationTokenTTLPolicyList{}
	err = req.Do(ctx, policyList)
	if err != nil {
		return nil, err
	}

	return policyList, nil
}

func (s *organizationTokenTTLPolicies) Update(ctx context.Context, organization string, options OrganizationTokenTTLPolicyUpdateOptions) ([]*OrganizationTokenTTLPolicy, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}

	if len(options.Policies) == 0 {
		return nil, ErrRequiredPolicies
	}

	u := fmt.Sprintf("organizations/%s/token-ttl-policies", url.PathEscape(organization))
	req, err := s.client.NewRequest("PATCH", u, &options)
	if err != nil {
		return nil, err
	}

	policyList := &OrganizationTokenTTLPolicyList{}
	err = req.Do(ctx, policyList)
	if err != nil {
		return nil, err
	}

	return policyList.Items, nil
}
