package tfe

import (
	"errors"
	"time"

	"github.com/manyminds/api2go/jsonapi"
)

// Run is an abstraction which wraps the flow of a Terraform plan and apply.
type Run struct {
	// The unique ID of this specific run.
	ID *string `json:"id,omitempty"`

	// Timestamp of when the run was created.
	CreatedAt *time.Time `json:"created-at,omitempty"`

	// The ID of the workspace associated with the run.
	WorkspaceID *string `json:"-"`

	// The ID of the configuration version the run was created with.
	// TODO: Make this actually use a JSONAPI relationship. Currently this
	//       is a plain old attribute.
	ConfigurationVersionID *string `json:"configuration_version_id,omitempty"`

	// Message is the description of the run, given at creation time.
	Message *string `json:"message,omitempty"`

	// Flag indicating if the run should destroy infrastructure (rather than
	// creating or changing it).
	Destroy *bool `json:"is-destroy,omitempty"`

	// True if the plan has completed successfully and has changes which can
	// be applied.
	HasChanges *bool `json:"has-changes,omitempty"`

	// Permissions the current API user has on the run.
	Permissions *Permissions `json:"permissions,omitempty"`

	// The source of the run. This reflects how the run was created (via the
	// UI, API, triggered from VCS, etc.).
	Source *string `json:"source,omitempty"`

	// Current status of the run (planning, applying, etc.).
	Status *string `json:"status,omitempty"`
}

// ListRunsInput holds the input values for listing runs.
type ListRunsInput struct {
	// Options used for paging through results.
	ListOptions

	// The workspace ID to list runs for.
	WorkspaceID *string
}

func (i *ListRunsInput) valid() error {
	if !validStringID(i.WorkspaceID) {
		return errors.New("Invalid value for WorkspaceID")
	}
	return nil
}

// ListRuns returns a list of runs present in the given workspace.
func (c *Client) ListRuns(input *ListRunsInput) ([]*Run, error) {
	if err := input.valid(); err != nil {
		return nil, err
	}
	wsID := *input.WorkspaceID

	var output jsonapiRuns

	if _, err := c.do(&request{
		method: "GET",
		path:   "/api/v2/workspaces/" + wsID + "/runs",
		output: &output,
	}); err != nil {
		return nil, err
	}

	runs := make([]*Run, len(output))
	for i, run := range output {
		runs[i] = run.Run
	}

	return runs, nil
}

// CreateRunInput holds all of the request fields for creating a new run.
type CreateRunInput struct {
	// The workspace ID to create the run for.
	WorkspaceID *string

	// The ID of the configuration version to use when creating the run.
	ConfigurationVersionID *string

	// Optional message to display with the run in TFE.
	Message *string

	// If provided, creates a run comment with the given text as part of the
	// run creation.
	Comment *string

	// When true, Terraform will attempt to destroy infrastructure (as
	// opposed to creating it otherwise).
	Destroy *bool
}

func (i *CreateRunInput) valid() error {
	if !validStringID(i.WorkspaceID) {
		return errors.New("Invalid value for WorkspaceID")
	}
	if v := i.ConfigurationVersionID; v != nil && !validStringID(v) {
		return errors.New("Invalid valud for ConfigurationVersionID")
	}
	return nil
}

// CreateRunOutput holds the return values from creating a run.
type CreateRunOutput struct {
	// A reference to the newly created Run.
	Run *Run
}

// CreateRun creates a new run in TFE. The run automatically enters the queue
// and begins executing the plan phase.
func (c *Client) CreateRun(input *CreateRunInput) (*CreateRunOutput, error) {
	if err := input.valid(); err != nil {
		return nil, err
	}

	// Create the special JSONAPI params.
	jsonapiParams := jsonapiRun{
		Run: &Run{
			WorkspaceID:            input.WorkspaceID,
			ConfigurationVersionID: input.ConfigurationVersionID,
			Message:                input.Message,
			Destroy:                input.Destroy,
		},
		Comment: input.Comment,
	}

	var output jsonapiRun

	if _, err := c.do(&request{
		method: "POST",
		path:   "/api/v2/runs",
		input:  jsonapiParams,
		output: &output,
	}); err != nil {
		return nil, err
	}

	return &CreateRunOutput{
		Run: output.Run,
	}, nil
}

type jsonapiRun struct {
	*Run

	// An optional comment passed at run creation time which will be used to
	// create a new Run comment. Only used during run creation (there will
	// never be a comment returned when listing or fetching runs), hence why
	// it exists here instead of in the Run struct.
	Comment *string `json:"comment,omitempty"`
}

func (r jsonapiRun) GetName() string {
	return "runs"
}

func (r jsonapiRun) GetID() string {
	if r.ID == nil {
		return ""
	}
	return *r.ID
}

func (r jsonapiRun) SetID(id string) (err error) {
	r.ID = String(id)
	return nil
}

func (r jsonapiRun) GetReferences() []jsonapi.Reference {
	return []jsonapi.Reference{
		jsonapi.Reference{
			Type: "workspaces",
			Name: "workspace",
		},
		jsonapi.Reference{
			Type: "configuration-versions",
			Name: "configuration-version",
		},
	}
}

func (r jsonapiRun) GetReferencedIDs() (result []jsonapi.ReferenceID) {
	if r.WorkspaceID != nil {
		result = append(result, jsonapi.ReferenceID{
			ID:           *r.WorkspaceID,
			Type:         "workspaces",
			Name:         "workspace",
			Relationship: jsonapi.ToOneRelationship,
		})
	}
	if r.ConfigurationVersionID != nil {
		result = append(result, jsonapi.ReferenceID{
			ID:           *r.ConfigurationVersionID,
			Type:         "configuration-versions",
			Name:         "configuration-version",
			Relationship: jsonapi.ToOneRelationship,
		})
	}
	return
}

func (r jsonapiRun) SetToOneReferenceID(name, id string) error {
	switch name {
	case "workspace":
		r.WorkspaceID = String(id)
	case "configuration-version":
		r.ConfigurationVersionID = String(id)
	}
	return nil
}

func (r jsonapiRun) SetToManyReferenceIDs(string, []string) error {
	return nil
}

type jsonapiRuns []jsonapiRun

func (jsonapiRuns) GetName() string                       { return "runs" }
func (jsonapiRuns) GetID() string                         { return "" }
func (jsonapiRuns) SetID(string) error                    { return nil }
func (jsonapiRuns) SetToOneReferenceID(a, b string) error { return nil }
