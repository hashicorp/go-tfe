package tfe

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"
)

// StateVersions handles communication with the state version related
// methods of the Terraform Enterprise API.
//
// TFE API docs:
// https://www.terraform.io/docs/enterprise/api/state-versions.html
type StateVersions struct {
	client *Client
}

// StateVersion represents a Terraform Enterprise state version.
type StateVersion struct {
	ID           string    `jsonapi:"primary,state-versions"`
	CreatedAt    time.Time `jsonapi:"attr,created-at,iso8601"`
	DownloadURL  string    `jsonapi:"attr,hosted-state-download-url"`
	Serial       int       `jsonapi:"attr,serial"`
	VCSCommitSHA string    `jsonapi:"attr,vcs-commit-sha"`
	VCSCommitURL string    `jsonapi:"attr,vcs-commit-url"`

	// Relations
	Run *Run `jsonapi:"relation,run"`
}

// StateVersionListOptions represents the options for listing state versions.
type StateVersionListOptions struct {
	ListOptions
	Organization *string `url:"filter[organization][name]"`
	Workspace    *string `url:"filter[workspace][name]"`
}

func (o StateVersionListOptions) valid() error {
	if !validString(o.Organization) {
		return errors.New("Organization is required")
	}
	if !validString(o.Workspace) {
		return errors.New("Workspace is required")
	}
	return nil
}

// List returns all the organizations visible to the current user.
func (s *StateVersions) List(ctx context.Context, options StateVersionListOptions) ([]*StateVersion, error) {
	if err := options.valid(); err != nil {
		return nil, err
	}

	req, err := s.client.newRequest("GET", "state-versions", &options)
	if err != nil {
		return nil, err
	}

	var svs []*StateVersion
	err = s.client.do(ctx, req, &svs)
	if err != nil {
		return nil, err
	}

	return svs, nil
}

// StateVersionCreateOptions represents the options for creating a state version.
type StateVersionCreateOptions struct {
	// For internal use only!
	ID string `jsonapi:"primary,state-versions"`

	// The lineage of the state.
	Lineage *string `jsonapi:"attr,lineage,omitempty"`

	// The MD5 hash of the state version.
	MD5 *string `jsonapi:"attr,md5"`

	// The serial of the state.
	Serial *int64 `jsonapi:"attr,serial"`

	// The base64 encoded state.
	State *string `jsonapi:"attr,state"`
}

func (o StateVersionCreateOptions) valid() error {
	if !validString(o.MD5) {
		return errors.New("MD5 is required")
	}
	if o.Serial == nil {
		return errors.New("Serial is required")
	}
	if !validString(o.State) {
		return errors.New("State is required")
	}
	return nil
}

// Create a new state version with the given name.
func (s *StateVersions) Create(ctx context.Context, workspaceID string, options StateVersionCreateOptions) (*StateVersion, error) {
	if !validStringID(&workspaceID) {
		return nil, errors.New("Invalid value for workspace ID")
	}
	if err := options.valid(); err != nil {
		return nil, err
	}

	// Make sure we don't send a user provided ID.
	options.ID = ""

	u := fmt.Sprintf("workspaces/%s/state-versions", url.QueryEscape(workspaceID))
	req, err := s.client.newRequest("POST", u, &options)
	if err != nil {
		return nil, err
	}

	sv := &StateVersion{}
	err = s.client.do(ctx, req, sv)
	if err != nil {
		return nil, err
	}

	return sv, nil
}

// Read a single state version by its ID.
func (s *StateVersions) Read(ctx context.Context, svID string) (*StateVersion, error) {
	if !validStringID(&svID) {
		return nil, errors.New("Invalid value for state version ID")
	}

	u := fmt.Sprintf("state-versions/%s", url.QueryEscape(svID))
	req, err := s.client.newRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	sv := &StateVersion{}
	err = s.client.do(ctx, req, sv)
	if err != nil {
		return nil, err
	}

	return sv, nil
}

// Download retrieves the actual stored state of a state version
func (s *StateVersions) Download(ctx context.Context, url string) ([]byte, error) {
	req, err := s.client.newRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	err = s.client.do(ctx, req, &buf)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
