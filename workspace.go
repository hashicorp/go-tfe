package tfe

// Workspace encapsulates all data fields of a workspace in TFE.
type Workspace struct {
	// The globally unique identifier for the workspace.
	ExternalID string `jsonapi:"primary,workspaces"`

	// Name of the workspace. This value is only guaranteed unique within
	// an organization.
	Name string `jsonapi:"attr,name"`

	// Creation time of the workspace.
	CreatedAt string `jsonapi:"attr,created-at"`

	// Indicates if plans will be automatically applied (without confirmation).
	AutoApply bool `jsonapi:"attr,auto-apply"`

	// The working directory used by Terraform during runs.
	WorkingDirectory string `jsonapi:"attr,working-directory"`

	// The version of Terraform which will be used to execute plan and
	// apply operations for this workspace.
	TerraformVersion string `jsonapi:"attr,terraform-version"`

	// VCSRepo holds the VCS settings for this workspace.
	VCSRepo WorkspaceVCSRepo `jsonapi:"attr,vcs-repo"`
}

// WorkspaceVCSRepo contains the configuration of a VCS integration as it
// pertains to a specific workspace.
type WorkspaceVCSRepo struct {
	// The ID of the VCS integration to use for cloning this workspace's
	// configuration.
	OauthTokenID string `json:"oauth-token-id,omitempty"`

	// The identifier of the VCS repository. The format of this field is
	// typically "<user or org>/<repo name>", depending on the VCS backend.
	Identifier string `json:"identifier,omitempty"`

	// Non-default branch to clone. Defaults to the default branch configured
	// at the VCS provider.
	Branch string `json:"branch,omitempty"`

	// Determines if submodules should be initialized and cloned on the
	// Terraform configuration repository when TFE clones the VCS repo.
	IncludeSubmodules bool `json:"ingress-submodules"`
}

// Workspaces returns all of the workspaces within an organization.
func (c *Client) Workspaces(organization string) ([]*Workspace, error) {
	var output []*Workspace

	if _, err := c.do(&request{
		method: "GET",
		path:   "/api/v2/organizations/" + organization + "/workspaces",
		output: &output,
	}); err != nil {
		return nil, err
	}

	return output, nil
}

// Workspace returns the workspace identified by the given org and name.
func (c *Client) Workspace(organization, workspace string) (*Workspace, error) {
	var ws Workspace

	if _, err := c.do(&request{
		method: "GET",
		path:   "/api/v2/organizations/" + organization + "/workspaces/" + workspace,
		output: &ws,
	}); err != nil {
		return nil, err
	}

	return &ws, nil
}

// CreateWorkspaceInput contains the parameters used for creating new
// new workspaces within an existing organization.
type CreateWorkspaceInput struct {
	// The organization name to create the workspace in.
	Organization string

	// The name of the workspace
	Name string

	// Determines if plans should automatically apply. Use this option with
	// caution - unexpected changes could be deployed to your infrastructure
	// if this is set to true.
	AutoApply bool

	// Allows creating a workspace in a locked state, such that no Terraform
	// runs can be executed until it is manually unlocked.
	Locked bool

	// The Terraform version number to run this workspace's configuration.
	// Setting this to "latest" will track the latest available version of
	// Terraform known to the TFE instance.
	TerraformVersion string

	// An optional subdirectory to use as the "root" of the Terraform
	// configuration. TFE will change to this directory before running any
	// Terraform CLI commands against the configuration.
	WorkingDirectory string

	// The ID of the VCS integration to use for cloning this workspace's
	// configuration.
	VCSOauthTokenID string

	// The identifier of the VCS repository. The format of this field is
	// typically "<user or org>/<repo name>", depending on the VCS backend.
	VCSRepo string

	// Non-default branch to clone. Defaults to the default branch configured
	// at the VCS provider.
	VCSBranch string

	// Determines if submodules should be initialized and cloned on the
	// Terraform configuration repository when TFE clones the VCS repo.
	VCSIncludeSubmodules bool
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

	return &CreateWorkspaceOutput{}, nil
}

// WorkspaceOrganizationSort provides sorting by the workspace name.
type WorkspaceNameSort []*Organization

func (w WorkspaceNameSort) Len() int           { return len(w) }
func (w WorkspaceNameSort) Less(a, b int) bool { return w[a].Name < w[b].Name }
func (w WorkspaceNameSort) Swap(a, b int)      { w[a], w[b] = w[b], w[a] }
