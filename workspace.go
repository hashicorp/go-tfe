package tfe

import (
	"errors"
	"fmt"
	"time"
)

// Workspaces handles communication with the workspace related methods of the
// Terraform Enterprise API.
//
// TFE API docs: https://www.terraform.io/docs/enterprise/api/workspaces.html
type Workspaces struct {
	client *Client
}

// Workspace represents a Terraform Enterprise workspace.
type Workspace struct {
	ID                   string                `jsonapi:"primary,workspaces"`
	Actions              *WorkspaceActions     `jsonapi:"attr,actions"`
	AutoApply            bool                  `jsonapi:"attr,auto-apply"`
	CanQueueDestroyPlan  bool                  `jsonapi:"attr,can-queue-destroy-plan"`
	CreatedAt            time.Time             `jsonapi:"attr,created-at,iso8601"`
	Environment          string                `jsonapi:"attr,environment"`
	Locked               bool                  `jsonapi:"attr,locked"`
	MigrationEnvironment string                `jsonapi:"attr,migration-environment"`
	Name                 string                `jsonapi:"attr,name"`
	Permissions          *WorkspacePermissions `jsonapi:"attr,permissions"`
	TerraformVersion     string                `jsonapi:"attr,terraform-version"`
	VCSRepo              *VCSRepo              `jsonapi:"attr,vcs-repo"`
	WorkingDirectory     string                `jsonapi:"attr,working-directory"`

	// Relations
	Organization *Organization `jsonapi:"relation,organization"`
	// SSHKey *SSHKey `jsonapi:"relation,ssh-key"
}

// VCSRepo contains the configuration of a VCS integration.
type VCSRepo struct {
	Branch            string `json:"branch"`
	Identifier        string `json:"identifier"`
	IncludeSubmodules bool   `json:"ingress-submodules"`
	OAuthTokenID      string `json:"oauth-token-id"`
}

// WorkspaceActions represents the workspace actions.
type WorkspaceActions struct {
	IsDestroyable bool `json:"is-destroyable"`
}

// WorkspacePermissions represents the workspace permissions.
type WorkspacePermissions struct {
	CanDestroy        bool `json:"can-destroy"`
	CanLock           bool `json:"can-lock"`
	CanQueueDestroy   bool `json:"can-queue-destroy"`
	CanQueueRun       bool `json:"can-queue-run"`
	CanReadSettings   bool `json:"can-read-settings"`
	CanUpdate         bool `json:"can-update"`
	CanUpdateVariable bool `json:"can-update-variable"`
}

// ListWorkspacesOptions represents the options for listing workspaces.
type ListWorkspacesOptions struct {
	ListOptions
}

// List returns all of the workspaces within an organization.
func (s *Workspaces) List(organization string, options ListWorkspacesOptions) ([]*Workspace, error) {
	if !validStringID(&organization) {
		return nil, errors.New("Invalid value for organization")
	}

	u := fmt.Sprintf("organizations/%s/workspaces", organization)
	req, err := s.client.newRequest("GET", u, &options)
	if err != nil {
		return nil, err
	}

	result, err := s.client.do(req, []*Workspace{})
	if err != nil {
		return nil, err
	}

	var ws []*Workspace
	for _, w := range result.([]interface{}) {
		ws = append(ws, w.(*Workspace))
	}

	return ws, nil
}

// CreateWorkspaceOptions represents the options for creating a new workspace.
type CreateWorkspaceOptions struct {
	// For internal use only!
	ID string `jsonapi:"primary,workspaces"`

	// Whether to automatically apply changes when a Terraform plan is successful.
	AutoApply *bool `jsonapi:"attr,auto-apply,omitempty"`

	// The legacy TFE environment to use as the source of the migration, in the
	// form organization/environment. Omit this unless you are migrating a legacy
	// environment.
	MigrationEnvironment *string `jsonapi:"attr,migration-environment,omitempty"`

	// The name of the workspace, which can only include letters, numbers, -,
	// and _. This will be used as an identifier and must be unique in the
	// organization.
	Name *string `jsonapi:"attr,name,omitempty"`

	// The version of Terraform to use for this workspace. Upon creating a
	// workspace, the latest version is selected unless otherwise specified.
	TerraformVersion *string `jsonapi:"attr,terraform-version,omitempty"`

	// Settings for the workspace's VCS repository. If omitted, the workspace is
	// created without a VCS repo. If included, you must specify at least the
	// oauth-token-id and identifier keys below.
	VCSRepo *VCSRepoOptions `jsonapi:"attr,vcs-repo,omitempty"`

	// A relative path that Terraform will execute within. This defaults to the
	// root of your repository and is typically set to a subdirectory matching the
	// environment when multiple environments exist within the same repository.
	WorkingDirectory *string `jsonapi:"attr,working-directory,omitempty"`
}

// VCSRepoOptions represents the configuration options of a VCS integration.
type VCSRepoOptions struct {
	Branch            *string `json:"branch,omitempty"`
	Identifier        *string `json:"identifier,omitempty"`
	IncludeSubmodules *bool   `json:"ingress-submodules,omitempty"`
	OAuthTokenID      *string `json:"oauth-token-id,omitempty"`
}

func (o CreateWorkspaceOptions) valid() error {
	if !validStringID(o.Name) {
		return errors.New("Invalid value for name")
	}
	return nil
}

// Create is used to create a new workspace.
func (s *Workspaces) Create(organization string, options CreateWorkspaceOptions) (*Workspace, error) {
	if !validStringID(&organization) {
		return nil, errors.New("Invalid value for organization")
	}
	if err := options.valid(); err != nil {
		return nil, err
	}

	// Make sure we don't send a user provided ID.
	options.ID = ""

	u := fmt.Sprintf("organizations/%s/workspaces", organization)
	req, err := s.client.newRequest("POST", u, &options)
	if err != nil {
		return nil, err
	}

	w, err := s.client.do(req, &Workspace{})
	if err != nil {
		return nil, err
	}

	return w.(*Workspace), nil
}

// Retrieve a single workspace by its name.
func (s *Workspaces) Retrieve(organization, workspace string) (*Workspace, error) {
	if !validStringID(&organization) {
		return nil, errors.New("Invalid value for organization")
	}
	if !validStringID(&workspace) {
		return nil, errors.New("Invalid value for workspace")
	}

	u := fmt.Sprintf("organizations/%s/workspaces/%s", organization, workspace)
	req, err := s.client.newRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	w, err := s.client.do(req, &Workspace{})
	if err != nil {
		return nil, err
	}

	return w.(*Workspace), nil
}

// UpdateWorkspaceOptions represents the options for updating a workspace.
type UpdateWorkspaceOptions struct {
	// For internal use only!
	ID string `jsonapi:"primary,workspaces"`

	// Whether to automatically apply changes when a Terraform plan is successful.
	AutoApply *bool `jsonapi:"attr,auto-apply,omitempty"`

	// A new name for the workspace, which can only include letters, numbers, -,
	// and _. This will be used as an identifier and must be unique in the
	// organization. Warning: Changing a workspace's name changes its URL in the
	// API and UI.
	Name *string `jsonapi:"attr,name,omitempty"`

	// The version of Terraform to use for this workspace.
	TerraformVersion *string `jsonapi:"attr,terraform-version,omitempty"`

	// To delete a workspace's existing VCS repo, specify null instead of an
	// object. To modify a workspace's existing VCS repo, include whichever of
	// the keys below you wish to modify. To add a new VCS repo to a workspace
	// that didn't previously have one, include at least the oauth-token-id and
	// identifier keys.  VCSRepo *VCSRepo `jsonapi:"relation,vcs-repo,om-tempty"`
	VCSRepo *VCSRepoOptions `jsonapi:"attr,vcs-repo,omitempty"`

	// A relative path that Terraform will execute within. This defaults to the
	// root of your repository and is typically set to a subdirectory matching
	// the environment when multiple environments exist within the same
	// repository.
	WorkingDirectory *string `jsonapi:"attr,working-directory,omitempty"`
}

// valid determines if the input is sufficiently filled.
func (o UpdateWorkspaceOptions) valid() error {
	if !validStringID(o.Name) {
		return errors.New("Invalid value for name")
	}
	return nil
}

// Update settings of an existing workspace.
func (s *Workspaces) Update(organization, workspace string, options UpdateWorkspaceOptions) (*Workspace, error) {
	if !validStringID(&organization) {
		return nil, errors.New("Invalid value for organization")
	}
	if !validStringID(&workspace) {
		return nil, errors.New("Invalid value for workspace")
	}
	if err := options.valid(); err != nil {
		return nil, err
	}

	// Make sure we don't send a user provided ID.
	options.ID = ""

	u := fmt.Sprintf("organizations/%s/workspaces/%s", organization, workspace)
	req, err := s.client.newRequest("PATCH", u, &options)
	if err != nil {
		return nil, err
	}

	w, err := s.client.do(req, &Workspace{})
	if err != nil {
		return nil, err
	}

	return w.(*Workspace), nil
}

// Delete a workspace by its name.
func (s *Workspaces) Delete(organization, workspace string) error {
	if !validStringID(&organization) {
		return errors.New("Invalid value for organization")
	}
	if !validStringID(&workspace) {
		return errors.New("Invalid value for workspace")
	}

	u := fmt.Sprintf("organizations/%s/workspaces/%s", organization, workspace)
	req, err := s.client.newRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	_, err = s.client.do(req, nil)

	return err
}

// LockWorkspaceOptions represents the options for locking a workspace.
type LockWorkspaceOptions struct {
	// Specifies the reason for locking the workspace.
	Reason *string `json:"reason,omitempty"`
}

// Lock a workspace.
func (s *Workspaces) Lock(workspaceID string, options LockWorkspaceOptions) (*Workspace, error) {
	if !validStringID(&workspaceID) {
		return nil, errors.New("Invalid value for workspace ID")
	}

	u := fmt.Sprintf("workspaces/%s/actions/lock", workspaceID)
	req, err := s.client.newRequest("POST", u, &options)
	if err != nil {
		return nil, err
	}

	w, err := s.client.do(req, &Workspace{})
	if err != nil {
		return nil, err
	}

	return w.(*Workspace), nil
}

// Unlock a workspace.
func (s *Workspaces) Unlock(workspaceID string) (*Workspace, error) {
	if !validStringID(&workspaceID) {
		return nil, errors.New("Invalid value for workspace ID")
	}

	u := fmt.Sprintf("workspaces/%s/actions/unlock", workspaceID)
	req, err := s.client.newRequest("POST", u, nil)
	if err != nil {
		return nil, err
	}

	w, err := s.client.do(req, &Workspace{})
	if err != nil {
		return nil, err
	}

	return w.(*Workspace), nil
}

// AssignSSHKeyoptions represents the options to assign an SSH key to a
// workspace.
type AssignSSHKeyoptions struct {
	// For internal use only!
	ID string `jsonapi:"primary,workspaces"`

	// The SSH key to assign.
	SSHKey *SSHKey `jsonapi:"relation,ssh-key,omitempty"`
}

// AssignSSHKey to a workspace.
func (s *Workspaces) AssignSSHKey(workspaceID string, options AssignSSHKeyoptions) (*Workspace, error) {
	if !validStringID(&workspaceID) {
		return nil, errors.New("Invalid value for workspace ID")
	}

	// Make sure we don't send a user provided ID.
	options.ID = ""

	u := fmt.Sprintf("workspaces/%s/relationships/ssh-key", workspaceID)
	req, err := s.client.newRequest("PATCH", u, &options)
	if err != nil {
		return nil, err
	}

	w, err := s.client.do(req, &Workspace{})
	if err != nil {
		return nil, err
	}

	return w.(*Workspace), nil
}

// unassignSSHKeyoptions represents the options to unassign an SSH key to a
// workspace.
type unassignSSHKeyoptions struct {
	// For internal use only!
	ID string `jsonapi:"primary,workspaces"`

	// Must be nil to unset the currently assigned SSH key.
	SSHKey *SSHKey `jsonapi:"relation,ssh-key,omitempty"`
}

// UnassignSSHKey from a workspace.
func (s *Workspaces) UnassignSSHKey(workspaceID string) (*Workspace, error) {
	if !validStringID(&workspaceID) {
		return nil, errors.New("Invalid value for workspace ID")
	}

	u := fmt.Sprintf("workspaces/%s/relationships/ssh-key", workspaceID)
	req, err := s.client.newRequest("PATCH", u, &unassignSSHKeyoptions{})
	if err != nil {
		return nil, err
	}

	w, err := s.client.do(req, &Workspace{})
	if err != nil {
		return nil, err
	}

	return w.(*Workspace), nil
}
