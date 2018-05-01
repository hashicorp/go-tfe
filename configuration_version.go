package tfe

import (
	"errors"
	"io/ioutil"
	"os"

	"github.com/hashicorp/go-tfe/slug"
)

// ConfigurationVersion is a representation of an uploaded or ingressed
// Terraform configuration in TFE. A workspace must have at least one
// configuration version before any runs may be queued on it.
type ConfigurationVersion struct {
	// The unique ID of the configuration version.
	ID *string `json:"id,omitempty"`

	// ID of the organization which owns this configuration version.
	OrganizationID *string `json:"-"`

	// If the configuration version failed to upload or ingress, this field
	// will contain the detailed error message indicating why.
	Error *string `json:"error_message,omitempty"`

	// Status indicates the current status of the configuration version. This
	// can be useful for determining whether the data is uploaded or not, or
	// if there has been an error.
	Status *string `json:"status,omitempty"`

	// The source of the configuration version. This indicates where the data
	// came from, which may be a manual upload, or a VCS integration, etc.
	Source *string `json:"source,omitempty"`

	// The URL to use for uploading configuration data. This field will only
	// be present until the configuration version has been uploaded once.
	// After that, if updating the configuration is needed, create a new
	// ConfigurationVersion and upload the updated data.
	UploadURL *string `json:"upload-url,omitempty"`
}

// ConfigurationVersion is used to look up a single configuration version.
func (c *Client) ConfigurationVersion(id string) (
	*ConfigurationVersion, error) {

	if !validStringID(&id) {
		return nil, errors.New("Invalid ID given")
	}

	var output jsonapiConfigurationVersion

	if _, err := c.do(&request{
		method: "GET",
		path:   "/api/v2/configuration-versions/" + id,
		output: &output,
	}); err != nil {
		return nil, err
	}

	return output.ConfigurationVersion, nil
}

// CreateConfigurationVersionInput holds the inputs to use when calling the
// configuration version creation API.
type CreateConfigurationVersionInput struct {
	// ID of the workspace the configuration version should belong to.
	WorkspaceID *string
}

func (i *CreateConfigurationVersionInput) valid() error {
	if !validStringID(i.WorkspaceID) {
		return errors.New("Invalid value for WorkspaceID")
	}
	return nil
}

// CreateConfigurationVersionOutput holds the return values from creating a
// configuration version.
type CreateConfigurationVersionOutput struct {
	// A reference to the newly-created configuration version.
	ConfigurationVersion *ConfigurationVersion
}

// CreateConfigurationVersion is used to create a new configuration version.
// The created configuration version will be usable once data is uploaded to
// it, which is done as a separate step. The created configuration version is
// applicable only to the given workspace ID.
func (c *Client) CreateConfigurationVersion(
	input *CreateConfigurationVersionInput) (
	*CreateConfigurationVersionOutput, error) {

	if err := input.valid(); err != nil {
		return nil, err
	}
	wsID := *input.WorkspaceID

	jsonapiParams := jsonapiConfigurationVersion{
		ConfigurationVersion: &ConfigurationVersion{},
	}

	var output jsonapiConfigurationVersion

	if _, err := c.do(&request{
		method: "POST",
		path:   "/api/v2/workspaces/" + wsID + "/configuration-versions",
		input:  jsonapiParams,
		output: &output,
	}); err != nil {
		return nil, err
	}

	return &CreateConfigurationVersionOutput{
		ConfigurationVersion: output.ConfigurationVersion,
	}, nil
}

// UploadConfigurationVersionInput holds the parameters used to upload the
// data of a configuration version.
type UploadConfigurationVersionInput struct {
	// The configuration version to upload data for.
	ConfigurationVersion *ConfigurationVersion

	// A path on the local filesystem which will be packaged and uploaded.
	// The full contents of this directory will be packed into a single file
	// and uploaded to Terraform Enterprise.
	Path *string
}

func (i *UploadConfigurationVersionInput) valid() error {
	if i.ConfigurationVersion == nil {
		return errors.New("Invalid value for ConfigurationVersion")
	}
	if !validString(i.ConfigurationVersion.UploadURL) {
		return errors.New("ConfigurationVersion has no UploadURL")
	}
	if !validString(i.Path) {
		return errors.New("Invalid value for Path")
	}
	return nil
}

// UploadConfigurationVersionOutput holds the outputs from uploading a
// configuration version.
type UploadConfigurationVersionOutput struct{}

// UploadConfigurationVersion packages and uploads Terraform configuration.
func (c *Client) UploadConfigurationVersion(
	input *UploadConfigurationVersionInput) (
	*UploadConfigurationVersionOutput, error) {

	if err := input.valid(); err != nil {
		return nil, err
	}

	fh, err := ioutil.TempFile("", "go-tfe")
	if err != nil {
		return nil, err
	}
	fh.Close()
	defer os.Remove(fh.Name())

	if _, err := slug.Pack(*input.Path, fh.Name()); err != nil {
		return nil, err
	}

	fh, err = os.Open(fh.Name())
	if err != nil {
		return nil, err
	}
	// Already have a defer os.Remove() on this.

	if err := c.upload(*input.ConfigurationVersion.UploadURL, fh); err != nil {
		return nil, err
	}

	return &UploadConfigurationVersionOutput{}, nil
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

func (j jsonapiConfigurationVersion) SetToOneReferenceID(
	name, id string) error {

	switch name {
	case "organization":
		j.OrganizationID = String(id)
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
