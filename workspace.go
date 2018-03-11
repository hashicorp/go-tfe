package tfe

import (
	"reflect"

	"github.com/google/jsonapi"
)

// The reflect type of a workspace. Used during deserialization.
var workspaceType = reflect.TypeOf(&Workspace{})

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
	resp, err := c.do(&request{
		method: "GET",
		path:   "/api/v2/organizations/" + organization + "/workspaces",
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	apiWorkspaces, err := jsonapi.UnmarshalManyPayload(
		resp.Body,
		workspaceType,
	)
	if err != nil {
		return nil, err
	}

	workspaces := make([]*Workspace, len(apiWorkspaces))
	for i, ws := range apiWorkspaces {
		workspaces[i] = ws.(*Workspace)
	}
	return workspaces, nil
}

// Workspace returns the workspace identified by the given org and name.
func (c *Client) Workspace(organization, workspace string) (*Workspace, error) {
	resp, err := c.do(&request{
		method: "GET",
		path:   "/api/v2/organizations/" + organization + "/workspaces/" + workspace,
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var ws Workspace
	if err := jsonapi.UnmarshalPayload(resp.Body, &ws); err != nil {
		return nil, err
	}
	return &ws, nil
}
