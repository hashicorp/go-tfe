// Copyright IBM Corp. 2018, 2025
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"fmt"
	"net/url"
	"time"
)

var _ OrganizationAuditConfigurations = (*organizationAuditConfigurations)(nil)

// OrganizationAuditConfigurations describes the configuration for auditing events for the organization.
type OrganizationAuditConfigurations interface {
	// Read the audit configuration of an organization by its name.
	Read(ctx context.Context, organization string) (*OrganizationAuditConfiguration, error)

	// Send a test audit event for an organization by its name.
	Test(ctx context.Context, organization string) (*OrganizationAuditConfigurationTest, error)

	// Update the audit configuration of an organization by its name.
	Update(ctx context.Context, organization string, options OrganizationAuditConfigurationOptions) (*OrganizationAuditConfiguration, error)
}

// OrganizationAuditConfiguration represents the auditing configuration for a HCP Terraform Organization.
type OrganizationAuditConfiguration struct {
	AuditTrails          *OrganizationAuditConfigAuditTrails    `jsonapi:"attr,audit-trails,omitempty"`
	HCPAuditLogStreaming *OrganizationAuditConfigAuditStreaming `jsonapi:"attr,hcp-audit-log-streaming,omitempty"`
	ID                   string                                 `jsonapi:"primary,audit-configurations"`
	Permissions          *OrganizationAuditConfigPermissions    `jsonapi:"attr,permissions,omitempty"`
	Timestamps           *OrganizationAuditConfigTimestamps     `jsonapi:"attr,timestamps,omitempty"`
	UpdatedAt            time.Time                              `jsonapi:"attr,updated-at,iso8601"`

	Organization *Organization `jsonapi:"relation,organization"`
}

type OrganizationAuditConfigAuditTrails struct {
	Enabled bool `jsonapi:"attr,enabled"`
}

type OrganizationAuditConfigAuditStreaming struct {
	Enabled                bool   `jsonapi:"attr,enabled"`
	OrganizationID         string `jsonapi:"attr,organization-id"`
	UseDefaultOrganization bool   `jsonapi:"attr,use-default-organization"`
}

type OrganizationAuditConfigPermissions struct {
	CanEnableHCPAuditLogStreaming              bool `jsonapi:"attr,can-enable-hcp-audit-log-streaming"`
	CanSetHCPAuditLogStreamingOrganization     bool `jsonapi:"attr,can-set-hcp-audit-log-streaming-organization-id"`
	CanUseDefaultAuditLogStreamingOrganization bool `jsonapi:"attr,can-use-default-audit-log-streaming-organization"`
}

type OrganizationAuditConfigTimestamps struct {
	AuditTrailsDisabledAt           *time.Time `jsonapi:"attr,audit-trails-disabled-at,iso8601,omitempty"`
	AuditTrailsEnabledAt            *time.Time `jsonapi:"attr,audit-trails-enabled-at,iso8601,omitempty"`
	AuditTrailsLastFailure          *time.Time `jsonapi:"attr,audit-trails-last-failure,iso8601,omitempty"`
	AuditTrailsLastSuccess          *time.Time `jsonapi:"attr,audit-trails-last-success,iso8601,omitempty"`
	HCPAuditLogStreamingDisabledAt  *time.Time `jsonapi:"attr,hcp-audit-log-streaming-disabled-at,iso8601,omitempty"`
	HCPAuditLogStreamingEnabledAt   *time.Time `jsonapi:"attr,hcp-audit-log-streaming-enabled-at,iso8601,omitempty"`
	HCPAuditLogStreamingLastFailure *time.Time `jsonapi:"attr,hcp-audit-log-streaming-last-failure,iso8601,omitempty"`
	HCPAuditLogStreamingLastSuccess *time.Time `jsonapi:"attr,hcp-audit-log-streaming-last-success,iso8601,omitempty"`
}

type OrganizationAuditConfigurationTest struct {
	RequestID *string `json:"request-id,omitempty"`
}

type OrganizationAuditConfigurationOptions struct {
	AuditTrails          *OrganizationAuditConfigAuditTrails    `jsonapi:"attr,audit-trails,omitempty"`
	HCPAuditLogStreaming *OrganizationAuditConfigAuditStreaming `jsonapi:"attr,hcp-audit-log-streaming,omitempty"`
}

type organizationAuditConfigurations struct {
	client *Client
}

// Read the audit configuration of an organization by its name.
func (s *organizationAuditConfigurations) Read(ctx context.Context, organization string) (*OrganizationAuditConfiguration, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}

	u := fmt.Sprintf("organizations/%s/audit-configuration", url.PathEscape(organization))
	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	ac := &OrganizationAuditConfiguration{}
	err = req.Do(ctx, ac)
	if err != nil {
		return nil, err
	}

	return ac, err
}

// Send a test audit event for an organization by its name.
func (s *organizationAuditConfigurations) Test(ctx context.Context, organization string) (*OrganizationAuditConfigurationTest, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}

	u := fmt.Sprintf("organizations/%s/audit-configuration/test", url.PathEscape(organization))
	req, err := s.client.NewRequest("POST", u, nil)
	if err != nil {
		return nil, err
	}

	result := &OrganizationAuditConfigurationTest{}
	err = req.DoJSON(ctx, result)
	if err != nil {
		return nil, err
	}

	return result, err
}

// Update the audit configuration of an organization by its name.
func (s *organizationAuditConfigurations) Update(ctx context.Context, organization string, options OrganizationAuditConfigurationOptions) (*OrganizationAuditConfiguration, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}

	u := fmt.Sprintf("organizations/%s/audit-configuration", url.PathEscape(organization))
	req, err := s.client.NewRequest("PATCH", u, &options)
	if err != nil {
		return nil, err
	}

	ac := &OrganizationAuditConfiguration{}
	err = req.Do(ctx, ac)
	if err != nil {
		return nil, err
	}

	return ac, err
}
