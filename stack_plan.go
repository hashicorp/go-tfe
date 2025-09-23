package tfe

import "time"

type StackPlanStatus string

const (
	StackPlanStatusCreated           StackPlanStatus = "created"
	StackPlanStatusRunning           StackPlanStatus = "running"
	StackPlanStatusRunningQueued     StackPlanStatus = "running_queued"
	StackPlanStatusRunningPlanning   StackPlanStatus = "running_planning"
	StackPlanStatusRunningApplying   StackPlanStatus = "running_applying"
	StackPlanStatusFinished          StackPlanStatus = "finished"
	StackPlanStatusFinishedNoChanges StackPlanStatus = "finished_no_changes"
	StackPlanStatusFinishedPlanned   StackPlanStatus = "finished_planned"
	StackPlanStatusFinishedApplied   StackPlanStatus = "finished_applied"
	StackPlanStatusDiscarded         StackPlanStatus = "discarded"
	StackPlanStatusErrored           StackPlanStatus = "errored"
	StackPlanStatusCanceled          StackPlanStatus = "canceled"
)

// StackPlanStatusTimestamps are the timestamps of the status changes for a stack
type StackPlanStatusTimestamps struct {
	CreatedAt  time.Time `jsonapi:"attr,created-at,rfc3339"`
	RunningAt  time.Time `jsonapi:"attr,running-at,rfc3339"`
	PausedAt   time.Time `jsonapi:"attr,paused-at,rfc3339"`
	FinishedAt time.Time `jsonapi:"attr,finished-at,rfc3339"`
}

// PlanChanges is the summary of the planned changes
type PlanChanges struct {
	Add    int `jsonapi:"attr,add"`
	Total  int `jsonapi:"attr,total"`
	Change int `jsonapi:"attr,change"`
	Import int `jsonapi:"attr,import"`
	Remove int `jsonapi:"attr,remove"`
}

// StackPlan represents a plan for a stack.
type StackPlan struct {
	ID               string                     `jsonapi:"primary,stack-plans"`
	PlanMode         string                     `jsonapi:"attr,plan-mode"`
	PlanNumber       string                     `jsonapi:"attr,plan-number"`
	Status           StackPlanStatus            `jsonapi:"attr,status"`
	StatusTimestamps *StackPlanStatusTimestamps `jsonapi:"attr,status-timestamps"`
	IsPlanned        bool                       `jsonapi:"attr,is-planned"`
	Changes          *PlanChanges               `jsonapi:"attr,changes"`
	Deployment       string                     `jsonapi:"attr,deployment"`

	// Relationships
	StackConfiguration *StackConfiguration `jsonapi:"relation,stack-configuration"`
	Stack              *Stack              `jsonapi:"relation,stack"`
}
