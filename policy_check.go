package tfe

import (
	"errors"
	"fmt"
	"time"
)

// PolicyChecks handles communication with the policy checks related methods
// of the Terraform Enterprise API.
//
// TFE API docs:
// https://www.terraform.io/docs/enterprise/api/policy-checks.html
type PolicyChecks struct {
	client *Client
}

// PolicyScope represents a policy scope.
type PolicyScope string

// List all available policy scopes.
const (
	PolicyScopeOrganization PolicyScope = "organization"
	PolicyScopeWorkspace    PolicyScope = "workspace"
)

// PolicyStatus represents a policy check state.
type PolicyStatus string

//List all available policy check statuses.
const (
	PolicyErrored    PolicyStatus = "errored"
	PolicyHardFailed PolicyStatus = "hard_failed"
	PolicyOverridden PolicyStatus = "overridden"
	PolicyPasses     PolicyStatus = "passes"
	PolicyPending    PolicyStatus = "pending"
	PolicyQueued     PolicyStatus = "queued"
	PolicySoftFailed PolicyStatus = "soft_failed"
)

// PolicyCheck represents a Terraform Enterprise policy check..
type PolicyCheck struct {
	ID               string                  `jsonapi:"primary,policy-checks"`
	Actions          *PolicyActions          `jsonapi:"attr,actions"`
	Permissions      *PolicyPermissions      `jsonapi:"attr,permissions"`
	Result           *PolicyResult           `jsonapi:"attr,result"`
	Scope            PolicyScope             `jsonapi:"attr,source"`
	Status           PolicyStatus            `jsonapi:"attr,status"`
	StatusTimestamps *PolicyStatusTimestamps `jsonapi:"attr,status-timestamps"`
}

// PolicyActions represents the policy check actions.
type PolicyActions struct {
	IsOverridable bool `json:"is-overridable"`
}

// PolicyPermissions represents the policy check permissions.
type PolicyPermissions struct {
	CanOverride bool `json:"can-override"`
}

// PolicyResult represents the complete policy check result,
type PolicyResult struct {
	AdvisoryFailed int  `json:"advisory-failed"`
	Duration       int  `json:"duration"`
	HardFailed     int  `json:"hard-failed"`
	Passed         int  `json:"passed"`
	Result         bool `json:"result"`
	// Sentinel       *sentinel.EvalResult `json:"sentinel"`
	SoftFailed  int `json:"soft-failed"`
	TotalFailed int `json:"total-failed"`
}

// PolicyStatusTimestamps holds the timestamps for individual policy check
// statuses.
type PolicyStatusTimestamps struct {
	ErroredAt    time.Time `json:"errored-at"`
	HardFailedAt time.Time `json:"hard-failed-at"`
	PassedAt     time.Time `json:"passed-at"`
	QueuedAt     time.Time `json:"queued-at"`
	SoftFailedAt time.Time `json:"soft-failed-at"`
}

// ListPolicyCheckOptions represents the options for listing policy checks.
type ListPolicyCheckOptions struct {
	ListOptions
}

// List all policy checks of the given run.
func (s *PolicyChecks) List(runID string, options ListPolicyCheckOptions) ([]*PolicyCheck, error) {
	if !validStringID(&runID) {
		return nil, errors.New("Invalid value for run ID")
	}

	u := fmt.Sprintf("runs/%s/policy-checks", runID)
	req, err := s.client.newRequest("GET", u, &options)
	if err != nil {
		return nil, err
	}

	result, err := s.client.do(req, []*PolicyCheck{})
	if err != nil {
		return nil, err
	}

	var ps []*PolicyCheck
	for _, p := range result.([]interface{}) {
		ps = append(ps, p.(*PolicyCheck))
	}

	return ps, nil
}

// Override a soft-mandatory or warning policy.
func (s *PolicyChecks) Override(policyID string) (*PolicyCheck, error) {
	if !validStringID(&policyID) {
		return nil, errors.New("Invalid value for policy ID")
	}

	u := fmt.Sprintf("policy-checks/%s/actions/override", policyID)
	req, err := s.client.newRequest("POST", u, nil)
	if err != nil {
		return nil, err
	}

	p, err := s.client.do(req, &PolicyCheck{})
	if err != nil {
		return nil, err
	}

	return p.(*PolicyCheck), nil
}
