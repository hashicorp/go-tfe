package tfe

import (
	"github.com/manyminds/api2go/jsonapi"
)

// Run is an abstraction which wraps the flow of a Terraform plan and apply.
type Run struct {
	// The unique ID of this specific run.
	ID *string `json:"id,omitempty"`

	// The ID of the workspace associated with the run.
	// TODO: Fix this. The var name in the JSONAPI payload shouldn't be
	// _external_id, it should just be _id like everything else.
	WorkspaceID *string `json:"workspace_external_id"`

	// The ID of the configuration version the run was created with.
	ConfigurationVersionID *string `json:"configuration_version_id"`

	// Message is the description of the run, given at creation time.
	Message *string `json:"message"`

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

// Runs returns a list of runs present in the given workspace.
func (c *Client) Runs(workspaceID string) ([]*Run, error) {
	var output jsonapiRuns

	if _, err := c.do(&request{
		method: "GET",
		path:   "/api/v2/workspaces/" + workspaceID + "/runs",
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

// CreateRun creates a new run in TFE. The run automatically enters the queue
// and begins executing the plan phase.
func (c *Client) CreateRun(input *CreateRunInput) (*Run, error) {
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

	return output.Run, nil
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
	return ""
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
	}
}

func (r jsonapiRun) GetReferencedIDs() []jsonapi.ReferenceID {
	return []jsonapi.ReferenceID{
		jsonapi.ReferenceID{
			ID:           *r.Run.WorkspaceID,
			Type:         "workspaces",
			Name:         "workspace",
			Relationship: jsonapi.ToOneRelationship,
		},
	}
}

func (r jsonapiRun) SetToOneReferenceID(name, id string) error {
	if name == "workspace" {
		r.Run.WorkspaceID = String(id)
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
