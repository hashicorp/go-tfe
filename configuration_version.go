package tfe

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	slug "github.com/hashicorp/go-slug"
)

// ConfigurationVersions handles communication with the configuration version
// related methods of the Terraform Enterprise API.
//
// TFE API docs:
// https://www.terraform.io/docs/enterprise/api/configuration-versions.html
type ConfigurationVersions struct {
	client *Client
}

// ConfigurationStatus represents a configuration version status.
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

// ConfigurationVersionListOptions represents the options for listing
// configuration versions.
type ConfigurationVersionListOptions struct {
	ListOptions
}

// List returns all configuration versions of a workspace.
func (s *ConfigurationVersions) List(ctx context.Context, workspaceID string, options ConfigurationVersionListOptions) ([]*ConfigurationVersion, error) {
	if !validStringID(&workspaceID) {
		return nil, errors.New("Invalid value for workspace ID")
	}

	u := fmt.Sprintf("workspaces/%s/configuration-versions", url.QueryEscape(workspaceID))
	req, err := s.client.newRequest("GET", u, &options)
	if err != nil {
		return nil, err
	}

	result, err := s.client.do(ctx, req, []*ConfigurationVersion{})
	if err != nil {
		return nil, err
	}

	var cvs []*ConfigurationVersion
	for _, cv := range result.([]interface{}) {
		cvs = append(cvs, cv.(*ConfigurationVersion))
	}

	return cvs, nil
}

// ConfigurationVersionCreateOptions represents the options for creating a
// configuration version.
type ConfigurationVersionCreateOptions struct {
	// For internal use only!
	ID string `jsonapi:"primary,configuration-versions"`

	// When true, runs are queued automatically when the configuration version
	// is uploaded.
	AutoQueueRuns *bool `jsonapi:"attr,auto-queue-runs,omitempty"`
}

// Create is used to create a new configuration version. The created
// configuration version will be usable once data is uploaded to it.
func (s *ConfigurationVersions) Create(ctx context.Context, workspaceID string, options ConfigurationVersionCreateOptions) (*ConfigurationVersion, error) {
	if !validStringID(&workspaceID) {
		return nil, errors.New("Invalid value for workspace ID")
	}

	// Make sure we don't send a user provided ID.
	options.ID = ""

	u := fmt.Sprintf("workspaces/%s/configuration-versions", url.QueryEscape(workspaceID))
	req, err := s.client.newRequest("POST", u, &options)
	if err != nil {
		return nil, err
	}

	cv, err := s.client.do(ctx, req, &ConfigurationVersion{})
	if err != nil {
		return nil, err
	}

	return cv.(*ConfigurationVersion), nil
}

// Read single configuration version by its ID.
func (s *ConfigurationVersions) Read(ctx context.Context, cvID string) (*ConfigurationVersion, error) {
	if !validStringID(&cvID) {
		return nil, errors.New("Invalid value for configuration version ID")
	}

	u := fmt.Sprintf("configuration-versions/%s", url.QueryEscape(cvID))
	req, err := s.client.newRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	cv, err := s.client.do(ctx, req, &ConfigurationVersion{})
	if err != nil {
		return nil, err
	}

	return cv.(*ConfigurationVersion), nil
}

// Upload packages and uploads Terraform configuration files. It requires the
// upload URL from a configuration version and the path to the configuration
// files on disk.
func (s *ConfigurationVersions) Upload(ctx context.Context, url, path string) error {
	body := bytes.NewBuffer(nil)

	_, err := slug.Pack(path, body)
	if err != nil {
		return err
	}

	req, err := s.client.newRequest("PUT", url, body)
	if err != nil {
		return err
	}

	_, err = s.client.do(ctx, req, nil)

	return err
}
