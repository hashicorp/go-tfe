package tfe

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"
)

// Compile-time proof of interface implementation.
var _ PolicySetVersions = (*policySetVersions)(nil)

// PolicySetVersions describes all the Policy Set Version related methods that the Terraform
// Enterprise API supports.
//
// TFE API docs: https://www.terraform.io/docs/cloud/api/policy-sets.html#create-a-policy-set-version
type PolicySetVersions interface {
	// Create is used to create a new Policy Set Version.
	Create(ctx context.Context, policySetID string) (*PolicySetVersion, error)

	// Read is used to read a Policy Set Version by its ID.
	Read(ctx context.Context, policySetVersionID string) (*PolicySetVersion, error)

	// Upload uploads policy files. It takes a Policy Set Version and a path
	// to the set of sentinel files, which will be packaged by hashicorp/go-slug
	// before being uploaded.
	Upload(ctx context.Context, psv PolicySetVersion, path string) error
}

// policySetVersions implements PolicySetVersions.
type policySetVersions struct {
	client *Client
}

// PolciySetVersionSource represents a source type of a policy set version.
type PolciySetVersionSource string

// List all available sources for a Policy Set Version.
const (
	PolciySetVersionSourceAPI       PolciySetVersionSource = "tfe-api"
	PolciySetVersionSourceADO       PolciySetVersionSource = "ado"
	PolciySetVersionSourceBitBucket PolciySetVersionSource = "bitbucket"
	PolciySetVersionSourceGitHub    PolciySetVersionSource = "github"
	PolciySetVersionSourceGitLab    PolciySetVersionSource = "gitlab"
)

// PolicySetVersionStatus represents a policy set version status.
type PolicySetVersionStatus string

//List all available policy set version statuses.
const (
	PolicySetVersionErrored PolicySetVersionStatus = "errored"
	PolicySetVersionPending PolicySetVersionStatus = "pending"
	PolicySetVersionReady   PolicySetVersionStatus = "ready"
)

// PolciySetVersionStatusTimestamps holds the timestamps for individual policy
// set version statuses.
type PolciySetVersionStatusTimestamps struct {
	PendingAt    time.Time `jsonapi:"attr,pending-at,rfc3339"`
	IngressingAt time.Time `jsonapi:"attr,ingressing-at,rfc3339"`
	ReadyAt      time.Time `jsonapi:"attr,ready-at,rfc3339"`
	ErroredAt    time.Time `jsonapi:"attr,errored-at,rfc3339"`
}

type PolicySetVersion struct {
	ID               string                           `jsonapi:"primary,policy-set-versions"`
	Source           PolciySetVersionSource           `jsonapi:"attr,source"`
	Status           PolicySetVersionStatus           `jsonapi:"attr,status"`
	StatusTimestamps PolciySetVersionStatusTimestamps `jsonapi:"attr,status-timestamps"`
	Error            string                           `jsonapi:"attr,error"`
	CreatedAt        time.Time                        `jsonapi:"attr,created-at,iso8601"`
	UpdatedAt        time.Time                        `jsonapi:"attr,updated-at,iso8601"`

	// Relations
	PolicySet *PolicySet `jsonapi:"relation,policy-set"`

	// Links
	Links map[string]interface{} `jsonapi:"links,omitempty"`
}

func (p PolicySetVersion) uploadURL() (string, error) {
	uploadURL, ok := p.Links["upload"].(string)
	if !ok {
		return uploadURL, fmt.Errorf("The Policy Set Version does not contain an upload link.")
	}

	if uploadURL == "" {
		return uploadURL, fmt.Errorf("The Policy Set Version upload URL is empty.")
	}

	return uploadURL, nil
}

// Create is used to create a new Policy Set Version.
func (p *policySetVersions) Create(ctx context.Context, policySetID string) (*PolicySetVersion, error) {
	if !validStringID(&policySetID) {
		return nil, errors.New("invalid value for policy set ID")
	}

	u := fmt.Sprintf("policy-sets/%s/versions", url.QueryEscape(policySetID))
	req, err := p.client.newRequest("POST", u, nil)
	if err != nil {
		return nil, err
	}

	psv := &PolicySetVersion{}
	err = p.client.do(ctx, req, psv)
	if err != nil {
		return nil, err
	}

	return psv, nil
}

// Read is used to read a Policy Set Version by its ID.
func (p *policySetVersions) Read(ctx context.Context, policySetVersionID string) (*PolicySetVersion, error) {
	if !validStringID(&policySetVersionID) {
		return nil, errors.New("invalid value for policy set ID")
	}

	u := fmt.Sprintf("policy-set-versions/%s", url.QueryEscape(policySetVersionID))
	req, err := p.client.newRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	psv := &PolicySetVersion{}
	err = p.client.do(ctx, req, psv)
	if err != nil {
		return nil, err
	}

	return psv, nil
}

// Upload uploads policy files. It takes a Policy Set Version and a path
// to the set of sentinel files, which will be packaged by hashicorp/go-slug
// before being uploaded.
func (p *policySetVersions) Upload(ctx context.Context, psv PolicySetVersion, path string) error {
	uploadURL, err := psv.uploadURL()
	if err != nil {
		return err
	}

	body, err := readFile(path)
	if err != nil {
		return err
	}

	req, err := p.client.newRequest("PUT", uploadURL, body)
	if err != nil {
		return err
	}

	return p.client.do(ctx, req, nil)
}
