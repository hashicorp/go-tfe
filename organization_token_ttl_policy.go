// Copyright IBM Corp. 2018, 2025
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"
)

var _ OrganizationTokenTTLPolicies = (*organizationTokenTTLPolicies)(nil)

const (
	TokenTypeOrganization = "organization"
	TokenTypeTeam         = "team"
	TokenTypeUser         = "user"
	TokenTypeAuditTrails  = "audit_trails"
)

type OrganizationTokenTTLPolicies interface {
	List(ctx context.Context, organization string, options *OrganizationTokenTTLPolicyListOptions) (*OrganizationTokenTTLPolicyList, error)
	Update(ctx context.Context, organization string, options OrganizationTokenTTLPolicyUpdateOptions) ([]*OrganizationTokenTTLPolicy, error)
}

type organizationTokenTTLPolicies struct {
	client *Client
}

type OrganizationTokenTTLPolicy struct {
	ID        string    `jsonapi:"primary,organization-token-ttl-policies"`
	TokenType string    `jsonapi:"attr,token-type"`
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
	TokenType string `json:"token_type"`
	MaxTTLMs  int64  `json:"max_ttl_ms"`
}

type OrganizationTokenTTLPolicyUpdateOptions struct {
	Policies []OrganizationTokenTTLPolicyUpdateItem `json:"token_ttl_policies"`
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

	body, err := json.Marshal(options)
	if err != nil {
		return nil, err
	}

	u := fmt.Sprintf("organizations/%s/token-ttl-policies", url.PathEscape(organization))
	req, err := s.client.NewRequest("PUT", u, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", ContentTypeJSONAPI)
	req.Header.Set("Accept", ContentTypeJSONAPI)

	policyList := &OrganizationTokenTTLPolicyList{}
	err = req.Do(ctx, policyList)
	if err != nil {
		return nil, err
	}

	return policyList.Items, nil
}
