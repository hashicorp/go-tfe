package tfe

import (
	"errors"
)

// ConfigurationVersion is a representation of an uploaded or ingressed
// Terraform configuration in TFE. A workspace must have at least one
// configuration version before any runs may be queued on it.
type ConfigurationVersion struct {
	// The unique ID of the configuration version.
	ID *string `json:"id,omitempty"`

	// ID of the organization which owns this configuration version.
	OrganizationID *string `json:"-"`

	// The workspace this configuration version is associated with.
	WorkspaceID *string `json:"-"`

	// If the configuration version failed to upload or ingress, this field
	// will contain the detailed error message indicating why.
	Error *string `json:"error_message,omitempty"`

	// Status indicates the current status of the configuration version. This
	// can be useful for determining whether the data is uploaded or not, or
	// if there has been an error.
	Status *string `json:"status,omitempty"`

	// Timestamp at which the configuration version was initially created.
	CreatedAt *string `json:"created-at,omitempty"`

	// Timestamp of the last update to the configuration version.
	UpdatedAt *string `json:"updated-at,omitempty"`

	// The source of the configuration version. This indicates where the data
	// came from, which may be a manual upload, or a VCS integration, etc.
	Source *string `json:"source,omitempty"`

	// The URL to use for uploading configuration data. This field will only
	// be present until the configuration version has been uploaded once.
	// After that, if updating the configuration is needed, create a new
	// ConfigurationVersion and upload the updated data.
	UploadURL *string `json:"upload_url,omitempty"`
}

// ListConfigurationVersions holds the parameters used to query a list of
// configuration versions associated with a particular workspace.
type ListConfigurationVersionsInput struct {
	// Options used for paging through results.
	ListOptions

	// The ID of the workspace to list configuration versions for.
	WorkspaceID *string
}

func (i *ListConfigurationVersionsInput) valid() error {
	if !validStringID(i.WorkspaceID) {
		return errors.New("Invalid value for WorkspaceID")
	}
	return nil
}

// ListConfigurationVersions is used to list the configuration versions
// associated with a given workspace or run.
func (c *Client) ListConfigurationVersions(
	input *ListConfigurationVersionsInput) ([]*ConfigurationVersion, error) {

	if err := input.valid(); err != nil {
		return nil, err
	}
	wsID := *input.WorkspaceID

	var result []jsonapiConfigurationVersion

	if _, err := c.do(&request{
		method: "GET",
		path:   "/api/v2/workspaces/" + wsID + "/configuration-versions",
		output: &result,
		lopt:   input.ListOptions,
	}); err != nil {
		return nil, err
	}

	output := make([]*ConfigurationVersion, len(result))
	for i, cv := range result {
		output[i] = cv.ConfigurationVersion
	}

	return output, nil
}

// Internal types to handle JSONAPI.
type jsonapiConfigurationVersion struct{ *ConfigurationVersion }

func (j jsonapiConfigurationVersion) GetName() string {
	return "configuration-versions"
}

func (j jsonapiConfigurationVersion) GetID() string {
	if j.ID == nil {
		return ""
	}
	return *j.ID
}

func (j jsonapiConfigurationVersion) SetID(id string) error {
	j.ID = String(id)
	return nil
}

func (j jsonapiConfigurationVersion) SetToOneReference(name, id string) error {
	switch name {
	case "organization":
		j.OrganizationID = String(id)
	case "workspace":
		j.WorkspaceID = String(id)
	}
	return nil
}

type jsonapiConfigurationVersions []jsonapiConfigurationVersion

func (jsonapiConfigurationVersions) GetName() string    { return "" }
func (jsonapiConfigurationVersions) GetID() string      { return "" }
func (jsonapiConfigurationVersions) SetID(string) error { return nil }
func (jsonapiConfigurationVersions) SetToOneReferenceID(a, b string) error {
	return nil
}
