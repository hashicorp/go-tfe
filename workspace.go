package tfe

import (
	"errors"
)

// Workspace encapsulates all data fields of a workspace in TFE.
type Workspace struct {
	// Unique ID of this workspace. This ID is guaranteed unique within the
	// context of the TFE instance.
	ID *string `json:"id,omitempty"`

	// ID of the organization which owns this workspace.
	OrganizationName *string `json:"-"`

	// Name of the workspace. This value is only guaranteed unique within
	// an organization.
	Name *string `json:"name,omitempty"`

	// Creation time of the workspace.
	CreatedAt *string `json:"created-at,omitempty"`

	// Indicates if plans will be automatically applied (without confirmation).
	AutoApply *bool `json:"auto-apply,omitempty"`

	// The working directory used by Terraform during runs.
	WorkingDirectory *string `json:"working-directory,omitempty"`

	// The version of Terraform which will be used to execute plan and
	// apply operations for this workspace.
	TerraformVersion *string `json:"terraform-version,omitempty"`

	// VCSRepo holds the VCS settings for this workspace.
	VCSRepo *VCSRepo `json:"vcs-repo,omitempty"`

	// Permissions the current API user has on the workspace.
	Permissions *Permissions `json:"permissions,omitempty"`
}

// Workspaces returns all of the workspaces within an organization.
func (c *Client) Workspaces(organization string) ([]*Workspace, error) {
	var result jsonapiWorkspaces

	if _, err := c.do(&request{
		method: "GET",
		path:   "/api/v2/organizations/" + organization + "/workspaces",
		output: &result,
	}); err != nil {
		return nil, err
	}

	output := make([]*Workspace, len(result))
	for i, ws := range result {
		output[i] = ws.Workspace
	}

	return output, nil
}

// Workspace returns the workspace identified by the given org and name.
func (c *Client) Workspace(organization, workspace string) (*Workspace, error) {
	var output jsonapiWorkspace

	if _, err := c.do(&request{
		method: "GET",
		path:   "/api/v2/organizations/" + organization + "/workspaces/" + workspace,
		output: &output,
	}); err != nil {
		return nil, err
	}

	return output.Workspace, nil
}

// CreateWorkspaceInput contains the parameters used for creating new
// new workspaces within an existing organization.
type CreateWorkspaceInput struct {
	// The organization name to create the workspace in.
	OrganizationName *string

	// The name of the workspace
	Name *string

	// Determines if plans should automatically apply. Use this option with
	// caution - unexpected changes could be deployed to your infrastructure
	// if this is set to true.
	AutoApply *bool

	// The Terraform version number to run this workspace's configuration.
	// Setting this to "latest" will track the latest available version of
	// Terraform known to the TFE instance.
	TerraformVersion *string

	// An optional subdirectory to use as the "root" of the Terraform
	// configuration. TFE will change to this directory before running any
	// Terraform CLI commands against the configuration.
	WorkingDirectory *string

	VCSRepo *VCSRepo
}

// Valid determines if the input is sufficiently filled.
func (i *CreateWorkspaceInput) Valid() error {
	if !isStringID(i.OrganizationName) {
		return errors.New("Invalid value for OrganizationName")
	}
	if !isStringID(i.Name) {
		return errors.New("Invalid value for Name")
	}
	return nil
}

// CreateWorkspaceOutput holds the return values from a workspace creation
// request.
type CreateWorkspaceOutput struct {
	// A reference to the newly created workspace.
	Workspace *Workspace
}

// CreateWorkspace is used to create a new workspace with the given parameters.
func (c *Client) CreateWorkspace(input *CreateWorkspaceInput) (
	*CreateWorkspaceOutput, error) {

	if err := input.Valid(); err != nil {
		return nil, err
	}
	orgName := *input.OrganizationName

	// Create the special JSONAPI payload.
	jsonapiParams := jsonapiWorkspace{
		Workspace: &Workspace{
			Name:             input.Name,
			AutoApply:        input.AutoApply,
			TerraformVersion: input.TerraformVersion,
			WorkingDirectory: input.WorkingDirectory,
			VCSRepo:          input.VCSRepo,
		},
	}

	var output jsonapiWorkspace

	// Send the request.
	if _, err := c.do(&request{
		method: "POST",
		path:   "/api/v2/organizations/" + orgName + "/workspaces",
		input:  jsonapiParams,
		output: &output,
	}); err != nil {
		return nil, err
	}

	return &CreateWorkspaceOutput{
		Workspace: output.Workspace,
	}, nil
}

// ModifyWorkspaceInput carries the adjustable values which can be modified
// on a workspace after its creation.
type ModifyWorkspaceInput struct {
	// The organization name the workspace belongs to. Required.
	OrganizationName *string

	// The current name of the workspace. Required.
	Name *string

	// A new name for the workspace. This changes the workspace name, which
	// may affect further API requests or Terraform configurations which refer
	// to the current workspace name in remote state references etc. Be
	// mindful when renaming workspaces!
	Rename *string

	// A new value for the auto-apply setting.
	AutoApply *bool

	// The Terraform version to use for runs in this workspace.
	TerraformVersion *string

	// The working directory to use when running Terraform commands. This is
	// relative to the root of the Terraform configuration.
	WorkingDirectory *string

	// VCS integration settings.
	VCSRepo *VCSRepo
}

// Valid determines if the input is sufficiently filled.
func (i *ModifyWorkspaceInput) Valid() error {
	if !isStringID(i.OrganizationName) {
		return errors.New("Invalid value for OrganizationName")
	}
	if !isStringID(i.Name) {
		return errors.New("Invalid value for Name")
	}
	return nil
}

// ModifyWorkspaceOutput is used to encapsulate the return values from a
// workspace modification command.
type ModifyWorkspaceOutput struct {
	// A reference to the modified workspace. All updated values are refelected
	// in this object.
	Workspace *Workspace
}

// ModifyWorkspace is used to adjust settings on an existing workspace.
func (c *Client) ModifyWorkspace(input *ModifyWorkspaceInput) (
	*ModifyWorkspaceOutput, error) {

	if err := input.Valid(); err != nil {
		return nil, err
	}
	orgName, wsName := *input.OrganizationName, *input.Name

	// Create the special JSONAPI payload.
	jsonapiParams := jsonapiWorkspace{
		Workspace: &Workspace{
			Name:             input.Rename,
			AutoApply:        input.AutoApply,
			TerraformVersion: input.TerraformVersion,
			WorkingDirectory: input.WorkingDirectory,
		},
	}

	var output jsonapiWorkspace

	// Send the request.
	if _, err := c.do(&request{
		method: "PATCH",
		path:   "/api/v2/organizations/" + orgName + "/workspaces/" + wsName,
		input:  jsonapiParams,
		output: &output,
	}); err != nil {
		return nil, err
	}

	return &ModifyWorkspaceOutput{
		Workspace: output.Workspace,
	}, nil
}

// DeleteWorkspaceInput carries the parameters used for deleting workspaces.
type DeleteWorkspaceInput struct {
	// Organization is the name of the organization in which the workspace
	// exists.
	OrganizationName *string

	// Name is the name of the workspace to delete.
	Name *string
}

// Valid determines if the input is filled sufficiently.
func (i *DeleteWorkspaceInput) Valid() error {
	if !isStringID(i.OrganizationName) {
		return errors.New("Invalid value for OrganizationName")
	}
	if !isStringID(i.Name) {
		return errors.New("Invalid value for Name")
	}
	return nil
}

// DeleteWorkspaceOutput holds the return values from deleting a workspace.
type DeleteWorkspaceOutput struct{}

// DeleteWorkspace is used to delete a single workspace.
func (c *Client) DeleteWorkspace(input *DeleteWorkspaceInput) (
	*DeleteWorkspaceOutput, error) {

	if err := input.Valid(); err != nil {
		return nil, err
	}
	orgName, wsName := *input.OrganizationName, *input.Name

	if _, err := c.do(&request{
		method: "DELETE",
		path:   "/api/v2/organizations/" + orgName + "/workspaces/" + wsName,
	}); err != nil {
		return nil, err
	}

	return &DeleteWorkspaceOutput{}, nil
}

// WorkspaceNameSort provides sorting by the workspace name.
type WorkspaceNameSort []*Workspace

func (w WorkspaceNameSort) Len() int           { return len(w) }
func (w WorkspaceNameSort) Less(a, b int) bool { return *w[a].Name < *w[b].Name }
func (w WorkspaceNameSort) Swap(a, b int)      { w[a], w[b] = w[b], w[a] }

// Internal type to satisfy the jsonapi interface for a single workspace.
type jsonapiWorkspace struct{ *Workspace }

func (w jsonapiWorkspace) GetName() string {
	return "workspaces"
}

func (w jsonapiWorkspace) GetID() string {
	if w.ID == nil {
		return ""
	}
	return *w.ID
}

func (w jsonapiWorkspace) SetID(id string) error {
	w.ID = String(id)
	return nil
}

func (w jsonapiWorkspace) SetToOneReferenceID(name, id string) error {
	switch name {
	case "organization":
		w.OrganizationName = String(id)
	}
	return nil
}

// Internal type to satisfy the jsonapi interface for workspace indexes.
type jsonapiWorkspaces []jsonapiWorkspace

func (jsonapiWorkspaces) GetName() string    { return "workspaces" }
func (jsonapiWorkspaces) GetID() string      { return "" }
func (jsonapiWorkspaces) SetID(string) error { return nil }
func (jsonapiWorkspaces) SetToOneReferenceID(a, b string) error {
	return nil
}
