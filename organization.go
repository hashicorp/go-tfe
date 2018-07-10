package tfe

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"
)

// Organizations handles communication with the organization related methods
// of the Terraform Enterprise API.
//
// TFE API docs:
// https://www.terraform.io/docs/enterprise/api/organizations.html
type Organizations struct {
	client *Client
}

// AuthPolicyType represents an authentication policy type.
type AuthPolicyType string

// List of available authentication policies.
const (
	AuthPolicyPassword  AuthPolicyType = "password"
	AuthPolicyTwoFactor AuthPolicyType = "two_factor_mandatory"
)

// EnterprisePlanType represents an enterprise plan type.
type EnterprisePlanType string

// List of available enterprise plan types.
const (
	EnterprisePlanDisabled EnterprisePlanType = "disabled"
	EnterprisePlanPremium  EnterprisePlanType = "premium"
	EnterprisePlanPro      EnterprisePlanType = "pro"
	EnterprisePlanTrial    EnterprisePlanType = "trial"
)

// Organization represents a Terraform Enterprise organization.
type Organization struct {
	Name                   string                   `jsonapi:"primary,organizations"`
	CollaboratorAuthPolicy AuthPolicyType           `jsonapi:"attr,collaborator-auth-policy"`
	CreatedAt              time.Time                `jsonapi:"attr,created-at,iso8601"`
	Email                  string                   `jsonapi:"attr,email"`
	EnterprisePlan         EnterprisePlanType       `jsonapi:"attr,enterprise-plan"`
	OwnersTeamSamlRoleID   string                   `jsonapi:"attr,owners-team-saml-role-id"`
	Permissions            *OrganizationPermissions `jsonapi:"attr,permissions"`
	SAMLEnabled            bool                     `jsonapi:"attr,saml-enabled"`
	SessionRemember        int                      `jsonapi:"attr,session-remember"`
	SessionTimeout         int                      `jsonapi:"attr,session-timeout"`
	TrialExpiresAt         time.Time                `jsonapi:"attr,trial-expires-at,iso8601"`
	TwoFactorConformant    bool                     `jsonapi:"attr,two-factor-conformant"`
}

// OrganizationPermissions represents the organization permissions.
type OrganizationPermissions struct {
	CanCreateTeam               bool `json:"can-create-team"`
	CanCreateWorkspace          bool `json:"can-create-workspace"`
	CanCreateWorkspaceMigration bool `json:"can-create-workspace-migration"`
	CanDestroy                  bool `json:"can-destroy"`
	CanTraverse                 bool `json:"can-traverse"`
	CanUpdate                   bool `json:"can-update"`
	CanUpdateAPIToken           bool `json:"can-update-api-token"`
	CanUpdateOAuth              bool `json:"can-update-oauth"`
	CanUpdateSentinel           bool `json:"can-update-sentinel"`
}

// OrganizationListOptions represents the options for listing organizations.
type OrganizationListOptions struct {
	ListOptions
}

// List returns all the organizations visible to the current user.
func (s *Organizations) List(ctx context.Context, options OrganizationListOptions) ([]*Organization, error) {
	req, err := s.client.newRequest("GET", "organizations", &options)
	if err != nil {
		return nil, err
	}

	var orgs []*Organization
	err = s.client.do(ctx, req, &orgs)
	if err != nil {
		return nil, err
	}

	return orgs, nil
}

// OrganizationCreateOptions represents the options for creating an organization.
type OrganizationCreateOptions struct {
	// For internal use only!
	ID string `jsonapi:"primary,organizations"`

	// Name of the organization.
	Name *string `jsonapi:"attr,name"`

	// Admin email address.
	Email *string `jsonapi:"attr,email"`
}

func (o OrganizationCreateOptions) valid() error {
	if !validString(o.Name) {
		return errors.New("Name is required")
	}
	if !validStringID(o.Name) {
		return errors.New("Invalid value for name")
	}
	if !validString(o.Email) {
		return errors.New("Email is required")
	}
	return nil
}

// Create a new organization with the given name and email.
func (s *Organizations) Create(ctx context.Context, options OrganizationCreateOptions) (*Organization, error) {
	if err := options.valid(); err != nil {
		return nil, err
	}

	// Make sure we don't send a user provided ID.
	options.ID = ""

	req, err := s.client.newRequest("POST", "organizations", &options)
	if err != nil {
		return nil, err
	}

	org := &Organization{}
	err = s.client.do(ctx, req, org)
	if err != nil {
		return nil, err
	}

	return org, nil
}

// Read single organization by its name.
func (s *Organizations) Read(ctx context.Context, name string) (*Organization, error) {
	if !validStringID(&name) {
		return nil, errors.New("Invalid value for name")
	}

	u := fmt.Sprintf("organizations/%s", url.QueryEscape(name))
	req, err := s.client.newRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	org := &Organization{}
	err = s.client.do(ctx, req, org)
	if err != nil {
		return nil, err
	}

	return org, nil
}

// OrganizationUpdateOptions represents the options for updating an organization.
type OrganizationUpdateOptions struct {
	// For internal use only!
	ID string `jsonapi:"primary,organizations"`

	// New name for the organization.
	Name *string `jsonapi:"attr,name,omitempty"`

	// New admin email address.
	Email *string `jsonapi:"attr,email,omitempty"`

	// Session expiration (minutes).
	SessionRemember *int `jsonapi:"attr,session-remember,omitempty"`

	// Session timeout after inactivity (minutes).
	SessionTimeout *int `jsonapi:"attr,session-timeout,omitempty"`

	// Authentication policy.
	CollaboratorAuthPolicy *AuthPolicyType `jsonapi:"attr,collaborator-auth-policy,omitempty"`
}

// Update attributes of an existing organization.
func (s *Organizations) Update(ctx context.Context, name string, options OrganizationUpdateOptions) (*Organization, error) {
	if !validStringID(&name) {
		return nil, errors.New("Invalid value for name")
	}

	// Make sure we don't send a user provided ID.
	options.ID = ""

	u := fmt.Sprintf("organizations/%s", url.QueryEscape(name))
	req, err := s.client.newRequest("PATCH", u, &options)
	if err != nil {
		return nil, err
	}

	org := &Organization{}
	err = s.client.do(ctx, req, org)
	if err != nil {
		return nil, err
	}

	return org, nil
}

// Delete an organization by its name.
func (s *Organizations) Delete(ctx context.Context, name string) error {
	if !validStringID(&name) {
		return errors.New("Invalid value for name")
	}

	u := fmt.Sprintf("organizations/%s", url.QueryEscape(name))
	req, err := s.client.newRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	return s.client.do(ctx, req, nil)
}
