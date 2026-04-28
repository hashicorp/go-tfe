// Copyright IBM Corp. 2018, 2025
// SPDX-License-Identifier: MPL-2.0
package tfe

import (
	"context"
	"fmt"
	"net/url"
	"time"
)

type TFPolicyEvaluationStatus string

const (
	TFPolicyEvaluationStatusCanceled    TFPolicyEvaluationStatus = "canceled"
	TFPolicyEvaluationStatusCreated     TFPolicyEvaluationStatus = "created"
	TFPolicyEvaluationStatusErrored     TFPolicyEvaluationStatus = "errored"
	TFPolicyEvaluationStatusFinished    TFPolicyEvaluationStatus = "finished"
	TFPolicyEvaluationStatusMFAWaiting  TFPolicyEvaluationStatus = "mfa_waiting"
	TFPolicyEvaluationStatusPending     TFPolicyEvaluationStatus = "pending"
	TFPolicyEvaluationStatusQueued      TFPolicyEvaluationStatus = "queued"
	TFPolicyEvaluationStatusRunning     TFPolicyEvaluationStatus = "running"
	TFPolicyEvaluationStatusUnreachable TFPolicyEvaluationStatus = "unreachable"
)

type TFPolicyEvaluationStageType string

const (
	TFPolicyEvaluationStageTypePlan  TFPolicyEvaluationStageType = "Plan"
	TFPolicyEvaluationStageTypeApply TFPolicyEvaluationStageType = "Apply"
)

type TFPolicyEvaluationStatusTimestamps struct {
	CanceledAt      time.Time `jsonapi:"attr,canceled-at,rfc3339"`
	ErroredAt       time.Time `jsonapi:"attr,errored-at,rfc3339"`
	FinishedAt      time.Time `jsonapi:"attr,finished-at,rfc3339"`
	ForceCanceledAt time.Time `jsonapi:"attr,force-canceled-at,rfc3339"`
	QueuedAt        time.Time `jsonapi:"attr,queued-at,rfc3339"`
	StartedAt       time.Time `jsonapi:"attr,started-at,rfc3339"`
}

type TFPolicyEvaluationResultCount struct {
	AdvisoryFailed int `jsonapi:"attr,advisory-failed"`
	MadatoryFailed int `jsonapi:"attr,mandatory-failed"`
	Passed         int `jsonapi:"attr,passed"`
	Errored        int `jsonapi:"attr,errored"`
	Unknown        int `jsonapi:"attr,unknown"`
}

type TFPolicyEvaluationErrorType string

const (
	TFPolicyEvaluationErrorTypeSetupError               TFPolicyEvaluationErrorType = "setup_error"
	TFPolicyEvaluationErrorTypeIncompaitbleAgentVersion TFPolicyEvaluationErrorType = "incompatible_agent_version"
)

type TFPolicyEvaluationError struct {
	Type    TFPolicyEvaluationErrorType `jsonapi:"attr,type"`
	Summary string                      `jsonapi:"attr,summary"`
	Details string                      `jsonapi:"attr,details"`
}

type TFPolicyEvaluationPermissions struct {
	CanOverride bool `jsonapi:"attr,can-override"`
}

type TFPolicyEvaluationActions struct {
	IsOverridable bool `jsonapi:"attr,is-overridable"`
}

type TFPolicyEvaluation struct {
	ID               string                              `jsonapi:"primary,tf-policy-evaluations"`
	Status           TFPolicyEvaluationStatus            `jsonapi:"attr,status"`
	StageType        TFPolicyEvaluationStageType         `jsonapi:"attr,stage-type"`
	StatusTimestamps *TFPolicyEvaluationStatusTimestamps `jsonapi:"attr,status-timestamps"`
	ResultCount      *TFPolicyEvaluationResultCount      `jsonapi:"attr,result-count"`
	Errors           *TFPolicyEvaluationError            `jsonapi:"relation,errors,omitempty"`
	OrganizedLog     bool                                `jsonapi:"attr,organized-log"`
	Permissions      *TFPolicyEvaluationPermissions      `jsonapi:"relation,permissions,omitempty"`
	Actions          *TFPolicyEvaluationActions          `jsonapi:"relation,actions,omitempty"`

	// Relations
	Run                        *Run                         `jsonapi:"relation,run,omitempty"`
	TFPolicyEvaluationOutcomes []*TFPolicyEvaluationOutcome `jsonapi:"relation,outcomes,omitempty"`
	// Links
	Links map[string]interface{} `jsonapi:"links,omitempty"`
}

type TFPolicyEvaluationOutcomes interface {
	List(ctx context.Context, tfPolicyEvaluationID string, options *TFPolicyEvaluationListOptions) (*TFPolicyEvaluationOutcomeList, error)
}

type TFPolicyEvaluationOutcomeEnforcementLevel string

const (
	TFPolicyEvaluationOutcomeEnforcementLevelAdvisory             TFPolicyEvaluationOutcomeEnforcementLevel = "advisory"
	TFPolicyEvaluationOutcomeEnforcementLevelMandatory            TFPolicyEvaluationOutcomeEnforcementLevel = "mandatory"
	TFPolicyEvaluationOutcomeEnforcementLevelMandatoryOverridable TFPolicyEvaluationOutcomeEnforcementLevel = "mandatory_overridable"
)

type TFPolicyEvaluationOutcomeStatus string

const (
	TFPolicyEvaluationOutcomeStatusPassed  TFPolicyEvaluationOutcomeStatus = "passed"
	TFPolicyEvaluationOutcomeStatusFailed  TFPolicyEvaluationOutcomeStatus = "failed"
	TFPolicyEvaluationOutcomeStatusErrored TFPolicyEvaluationOutcomeStatus = "errored"
	TFPolicyEvaluationOutcomeStatusUnknown TFPolicyEvaluationOutcomeStatus = "unknown"
)

type TFPolicyEvaluationOutcomeDiagnostic struct {
	Code            string                               `jsonapi:"attr,code"`
	Context         string                               `jsonapi:"attr,context"`
	StartLine       int                                  `jsonapi:"attr,start_line"`
	Summary         string                               `jsonapi:"attr,summary"`
	Resource        *[]TFPolicyEvaluationOutcomeResource `jsonapi:"relation,resources,omitempty"`
	PassedResources []*TFPolicyEvaluationOutcomeResource `jsonapi:"relation,passed_resources,omitempty"`
}

type TFPolicyEvaluationOutcomeResource struct {
	ResourceName string   `jsonapi:"attr,resource_name"`
	InfoMessage  string   `jsonapi:"attr,info_message"`
	InfoMessages []string `jsonapi:"attr,info_messages,omitempty"`
	Code         string   `jsonapi:"attr,code,omitempty"`
	FileName     string   `jsonapi:"attr,file_name,omitempty"`
	StartLine    int      `jsonapi:"attr,start_line,omitempty"`
	Context      string   `jsonapi:"attr,context,omitempty"`
	Start        int      `jsonapi:"attr,start,omitempty"`
	Values       []*struct {
		Traversal string `jsonapi:"attr,traversal"`
		Statement string `jsonapi:"attr,statement"`
	}
}

type TFPolicyEvaluationOutcomeOutputPolicy struct {
	Code    string `jsonapi:"attr,code"`
	Context string `jsonapi:"attr,context"`
	Start   int    `jsonapi:"attr,start"`
	Values  any    `jsonapi:"attr,values"`
}

type TFPolicyEvaluationOutcomeOutput struct {
	Message  string                                `jsonapi:"attr,message"`
	Policy   TFPolicyEvaluationOutcomeOutputPolicy `jsonapi:"attr,policy"`
	Resource TFPolicyEvaluationOutcomeResource     `jsonapi:"attr,resource,omitempty"`
	Severity string                                `jsonapi:"attr,severity,omitempty"`
}

type TFPolicyEvaluationOutcomeOutcome struct {
	EnforcementLevel TFPolicyEvaluationOutcomeEnforcementLevel `jsonapi:"attr,enforcement_level"`
	Status           TFPolicyEvaluationOutcomeStatus           `jsonapi:"attr,status"`
	Description      string                                    `jsonapi:"attr,description"`
	FileName         string                                    `jsonapi:"attr,file_name"`
	Output           []*TFPolicyEvaluationOutcomeOutput        `jsonapi:"attr,output,omitempty"`
	Diagnostics      []*TFPolicyEvaluationOutcomeDiagnostic    `jsonapi:"attr,diagnostics,omitempty"`
}

type TFPolicyEvaluationOutcome struct {
	ID                   string                              `jsonapi:"primary,tf-policy-set-outcomes"`
	Outcomes             []*TFPolicyEvaluationOutcomeOutcome `jsonapi:"attr,outcomes,omitempty"`
	Error                *TFPolicyEvaluationError            `jsonapi:"attr,error,omitempty"`
	Overriable           bool                                `jsonapi:"attr,overridable"`
	PolicySetName        string                              `jsonapi:"attr,policy-set-name"`
	PolicySetDescription string                              `jsonapi:"attr,policy-set-description"`
	ResultCount          *TFPolicyEvaluationResultCount      `jsonapi:"attr,result-count,omitempty"`

	// Relations
	TFPolicyEvaluation *TFPolicyEvaluation `jsonapi:"relation,tf-policy-evaluation,omitempty"`
}

type TFPolicyEvaluationListOptions struct {
	ListOptions

	Status           string `url:"filter[status],omitempty"`
	EnforcementLevel string `url:"filter[enforcement-level],omitempty"`
}

type TFPolicyEvaluationOutcomeList struct {
	*Pagination
	Items []*TFPolicyEvaluationOutcome
}

type tfPolicyEvaluationOutcomes struct {
	client *Client
}

func (s *tfPolicyEvaluationOutcomes) List(ctx context.Context, tfPolicyEvaluationID string, options *TFPolicyEvaluationListOptions) (*TFPolicyEvaluationOutcomeList, error) {
	if !validStringID(&tfPolicyEvaluationID) {
		return nil, ErrInvalidTFPolicyEvaluationID
	}

	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("tf-policy-evaluations/%s/tf-policy-set-outcomes", url.PathEscape(tfPolicyEvaluationID))
	req, err := s.client.NewRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	tfpo := &TFPolicyEvaluationOutcomeList{}
	err = req.Do(ctx, tfpo)
	if err != nil {
		return nil, err
	}

	return tfpo, nil
}

func (s *TFPolicyEvaluationListOptions) valid() error {
	return nil
}
