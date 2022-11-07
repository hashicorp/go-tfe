package tfe

import "time"

type PolicyResultCount struct {
	AdvisoryFailed  int `jsonapi:"attr,advisory-failed"`
	MandatoryFailed int `jsonapi:"attr,mandatory-failed"`
	Passed          int `jsonapi:"attr,passed"`
}

type PolicyAttachable struct {
	ID   string `jsonapi:"attr,id"`
	Type string `jsonapi:"attr,type"`
}

// PolicyEvaluations represents the complete policy result
type PolicyEvaluations struct {
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
