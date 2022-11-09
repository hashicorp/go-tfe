package tfe

import (
	"context"
	"fmt"
	"net/url"
	"time"
)

// Compile-time proof of interface implementation.
var _ PolicyEvaluations = (*policyEvaluation)(nil)

type PolicyResultCount struct {
	AdvisoryFailed  int `jsonapi:"attr,advisory-failed"`
	MandatoryFailed int `jsonapi:"attr,mandatory-failed"`
	Passed          int `jsonapi:"attr,passed"`
	Errored         int `jsonapi:"attr,errored"`
}

type PolicyAttachable struct {
	ID   string `jsonapi:"attr,id"`
	Type string `jsonapi:"attr,type"`
}

// PolicyEvaluations represents the complete policy result
type PolicyEvaluation struct {
	ID               string                     `jsonapi:"primary,policy-evaluations"`
	Status           TaskResultStatus           `jsonapi:"attr,status"`
	PolicyKind       PolicyKind                 `jsonapi:"attr,policy-kind"`
	StatusTimestamps TaskResultStatusTimestamps `jsonapi:"attr,status-timestamps"`
	ResultCount      *PolicyResultCount         `jsonapi:"attr,result-count"`
	CreatedAt        time.Time                  `jsonapi:"attr,created-at,iso8601"`
	UpdatedAt        time.Time                  `jsonapi:"attr,updated-at,iso8601"`

	// The task stage this result belongs to
	TaskStage *PolicyAttachable `jsonapi:"relation,policy-attachable"`
}

// PolicyEvalutations describes all the policy evaluation related methods that the
// Terraform Enterprise API supports.
//
// TFE API docs:
// https://www.terraform.io/docs/cloud/api/policy-checks.html
type PolicyEvaluations interface {
	// List all policy evaluations in the task stage.
	List(ctx context.Context, taskStageID string, options *PolicyEvaluationListOptions) (*PolicyEvaluationList, error)
}

// policyEvaluations implements PolicyEvalutations.
type policyEvaluation struct {
	client *Client
}

// PolicyEvaluationListOptions represents the options for listing policy evaluations.
type PolicyEvaluationListOptions struct {
	ListOptions
}

// PolicyEvalutationList represents a list of policy checks.
type PolicyEvaluationList struct {
	*Pagination
	Items []*PolicyEvaluation
}

// List all policy checks of the given run.
func (s *policyEvaluation) List(ctx context.Context, taskStageID string, options *PolicyEvaluationListOptions) (*PolicyEvaluationList, error) {
	if !validStringID(&taskStageID) {
		return nil, ErrInvalidTaskStageID
	}

	u := fmt.Sprintf("task-stages/%s/policy-evaluations", url.QueryEscape(taskStageID))
	req, err := s.client.NewRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	pcl := &PolicyEvaluationList{}
	err = req.Do(ctx, pcl)
	if err != nil {
		return nil, err
	}

	return pcl, nil
}

// Compile-time proof of interface implementation.
var _ PolicyOutcomes = (*policyOutcome)(nil)

// PolicyOutcomes describes all the policy outcome related methods that the
// Terraform Enterprise API supports.
//
// TFE API docs:
// https://www.terraform.io/docs/cloud/api/policy-checks.html
type PolicyOutcomes interface {
	// List all policy outcomes in the policy evaluation.
	List(ctx context.Context, policyEvaluationID string, options *PolicyOutcomeListOptions) (*PolicyOutcomeList, error)

	// Read a policy outcome by its ID.
	Read(ctx context.Context, policy_set_outcome_id string) (*PolicyOutcome, error)
}

// policyOutcome implements PolicyOutcomes.
type policyOutcome struct {
	client *Client
}

type PolicyOutcomeListFilter struct {
	// Optional: A status string used to filter the results.
	// Must be either "passed", "failed", or "errored".
	Status string

	// Optional: The enforcement level used to filter the results.
	// Must be either "advisory" or "mandatory".
	EnforcementLevel string
}

// PolicyOutcomeListOptions represents the options for listing policy outcomes.
type PolicyOutcomeListOptions struct {
	ListOptions

	Filter map[string]PolicyOutcomeListFilter
}

// PolicyOutcomeList represents a list of policy outcomes.
type PolicyOutcomeList struct {
	*Pagination
	Items []*PolicyOutcome
}

type PolicySetOutcome struct {
	EnforcementLevel EnforcementLevel `jsonapi:"attr,enforcement_level"`
	Query            string           `jsonapi:"attr,query"`
	Status           string           `jsonapi:"attr,status"`
	PolicyName       string           `jsonapi:"attr,policy_name"`
	Description      string           `jsonapi:"attr,description"`
}

// PolicyOutcome represents policy set outcomes that are part of the policy evaluation
type PolicyOutcome struct {
	ID                   string             `jsonapi:"primary,policy-evaluations"`
	Outcomes             []PolicySetOutcome `jsonapi:"attr,outcomes"`
	Error                string             `jsonapi:"attr,error"`
	Overridable          *bool              `jsonapi:"attr,overridable"`
	PolicySetName        string             `jsonapi:"attr,policy-set-name"`
	PolicySetDescription string             `jsonapi:"attr,policy-set-description"`
	ResultCount          PolicyResultCount  `jsonapi:"attr,result_count"`

	// The policy evaluation that this outcome belongs to
	PolicyEvaluation *PolicyEvaluation `jsonapi:"relation,policy-evaluation"`
}

// List all policy checks of the given run.
func (s *policyOutcome) List(ctx context.Context, policyEvaluationID string, options *PolicyOutcomeListOptions) (*PolicyOutcomeList, error) {
	if !validStringID(&policyEvaluationID) {
		return nil, ErrInvalidPolicyEvaluationID
	}

	additionalQueryParams := options.buildQueryString()

	u := fmt.Sprintf("policy-evaluations/%s/policy-set-outcomes", url.QueryEscape(policyEvaluationID))

	req, err := s.client.NewRequestWithAdditionalQueryParams("GET", u, options.ListOptions, additionalQueryParams)
	if err != nil {
		return nil, err
	}

	pol := &PolicyOutcomeList{}
	err = req.Do(ctx, pol)
	if err != nil {
		return nil, err
	}

	return pol, nil
}

func (opts *PolicyOutcomeListOptions) buildQueryString() map[string][]string {
	result := make(map[string][]string)
	for k, v := range opts.Filter {
		if v.Status != "" {
			newKey := fmt.Sprintf("filter[%s][status]", k)
			result[newKey] = append(result[newKey], v.Status)
		}
		if v.EnforcementLevel != "" {
			newKey := fmt.Sprintf("filter[%s][enforcement_level]", k)
			result[newKey] = append(result[newKey], v.EnforcementLevel)
		}
	}
	return result
}

// Read reads a policy set outcome by its ID
func (s *policyOutcome) Read(ctx context.Context, policySetOutcomeID string) (*PolicyOutcome, error) {
	if !validStringID(&policySetOutcomeID) {
		return nil, ErrInvalidPolicySetOutcomeID
	}

	u := fmt.Sprintf("policy-set-outcomes/%s", url.QueryEscape(policySetOutcomeID))
	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	po := &PolicyOutcome{}
	err = req.Do(ctx, po)
	if err != nil {
		return nil, err
	}

	return po, err
}
