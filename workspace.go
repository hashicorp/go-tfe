package tfe

// Workspace encapsulates all data fields of a workspace in TFE.
type Workspace struct {
	ExternalID       string `jsonapi:"primary,workspaces"`
	Name             string `jsonapi:"attr,name"`
	CreatedAt        string `jsonapi:"attr,created-at"`
	AutoApply        bool   `jsonapi:"attr,auto-apply"`
	WorkingDirectory string `jsonapi:"attr,working-directory"`
	TerraformVersion string `jsonapi:"attr,terraform-version"`
}

// Workspaces returns all of the workspaces within an organization.
func (c *Client) Workspaces(organization string) ([]*Workspace, error) {
	var output []*Workspace

	if _, err := c.do(&request{
		method: "GET",
		path:   "/api/v2/organizations/" + organization + "/workspaces",
		output: output,
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
