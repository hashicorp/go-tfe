package tfe

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/hashicorp/go-tfe/slug"
)

// ConfigurationVersions handles communication with the configuration version
// related methods of the Terraform Enterprise API.
//
// TFE API docs:
// https://www.terraform.io/docs/enterprise/api/configuration-versions.html
type ConfigurationVersions struct {
	client *Client
}

// ConfigurationStatus represents a configuration version state.
type ConfigurationStatus string

//List all available configuration version statuses.
const (
	ConfigurationErrored  ConfigurationStatus = "errored"
	ConfigurationPending  ConfigurationStatus = "pending"
	ConfigurationUploaded ConfigurationStatus = "uploaded"
)

// ConfigurationSource represents a source of a configuration version.
type ConfigurationSource string

// List all available configuration version sources.
const (
	ConfigurationSourceAPI       ConfigurationSource = "tfe-api"
	ConfigurationSourceBitbucket ConfigurationSource = "bitbucket"
	ConfigurationSourceGithub    ConfigurationSource = "github"
	ConfigurationSourceGitlab    ConfigurationSource = "gitlab"
	ConfigurationSourceTerraform ConfigurationSource = "terraform"
)

// ConfigurationVersion is a representation of an uploaded or ingressed
// Terraform configuration in TFE. A workspace must have at least one
// configuration version before any runs may be queued on it.
type ConfigurationVersion struct {
	ID               string              `jsonapi:"primary,configuration-versions"`
	AutoQueueRuns    bool                `jsonapi:"attr,auto-queue-runs"`
	Error            string              `jsonapi:"attr,error"`
	ErrorMessage     string              `jsonapi:"attr,error-message"`
	Source           ConfigurationSource `jsonapi:"attr,source"`
	Status           ConfigurationStatus `jsonapi:"attr,status"`
	StatusTimestamps *CVStatusTimestamps `jsonapi:"attr,status-timestamps"`
	UploadURL        string              `jsonapi:"attr,upload-url"`
}

// CVStatusTimestamps holds the timestamps for individual configuration version
// statuses.
type CVStatusTimestamps struct {
	FinishedAt time.Time `json:"finished-at"`
	QueuedAt   time.Time `json:"queued-at"`
	StartedAt  time.Time `json:"started-at"`
}

// ListConfigurationVersionsOptions represents the options for listing
// configuration versions.
type ListConfigurationVersionsOptions struct {
	ListOptions
}

// List returns all configuration versions of a workspace.
func (s *ConfigurationVersions) List(workspaceID string, options *ListConfigurationVersionsOptions) ([]*ConfigurationVersion, error) {
	if !validStringID(&workspaceID) {
		return nil, errors.New("Invalid value for workspace ID")
	}

	u := fmt.Sprintf("workspaces/%s/configuration-versions", workspaceID)
	req, err := s.client.newRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	result, err := s.client.do(req, []*ConfigurationVersion{})
	if err != nil {
		return nil, err
	}

	var cvs []*ConfigurationVersion
	for _, cv := range result.([]interface{}) {
		cvs = append(cvs, cv.(*ConfigurationVersion))
	}

	return cvs, nil
}

// CreateConfigurationVersionOptions represents the options for creating a
// configuration version.
type CreateConfigurationVersionOptions struct {
	// For internal use only!
	ID string `jsonapi:"primary,organizations"`

	// When true, runs are queued automatically when the configuration version
	// is uploaded.
	AutoQueueRuns *bool `jsonapi:"attr,auto-queue-runs,omitempty"`
}

// Create is used to create a new configuration version. The created
// configuration version will be usable once data is uploaded to it.
func (s *ConfigurationVersions) Create(workspaceID string, options *CreateConfigurationVersionOptions) (*ConfigurationVersion, error) {
	if !validStringID(&workspaceID) {
		return nil, errors.New("Invalid value for workspace ID")
	}

	// TODO SvH: This shouldn't be needed right?
	if options == nil {
		options = &CreateConfigurationVersionOptions{}
	}

	u := fmt.Sprintf("workspaces/%s/configuration-versions", workspaceID)
	req, err := s.client.newRequest("POST", u, options)
	if err != nil {
		return nil, err
	}

	cv, err := s.client.do(req, &ConfigurationVersion{})
	if err != nil {
		return nil, err
	}

	return cv.(*ConfigurationVersion), nil
}

// Retrieve single configuration version by its ID.
func (s *ConfigurationVersions) Retrieve(cvID string) (*ConfigurationVersion, error) {
	if !validStringID(&cvID) {
		return nil, errors.New("Invalid value for configuration version ID")
	}

	req, err := s.client.newRequest("GET", "configuration-versions/"+cvID, nil)
	if err != nil {
		return nil, err
	}

	cv, err := s.client.do(req, &ConfigurationVersion{})
	if err != nil {
		return nil, err
	}

	return cv.(*ConfigurationVersion), nil
}

// Upload packages and uploads Terraform configuration files. It requires the
// upload URL from a configuration version and the path to the configuration
// files on disk.
func (s *ConfigurationVersions) Upload(url, path string) error {
	fh, err := ioutil.TempFile("", "go-tfe")
	if err != nil {
		return err
	}
	fh.Close()
	defer os.Remove(fh.Name())

	if _, err := slug.Pack(path, fh.Name()); err != nil {
		return err
	}

	fh, err = os.Open(fh.Name())
	if err != nil {
		return err
	}
	// Already have a defer os.Remove() on this.

	return s.client.upload(url, fh)
}
