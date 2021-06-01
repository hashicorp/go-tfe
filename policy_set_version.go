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

	// Upload a tarball to the Policy Set Version.
	Upload(ctx context.Context, psv PolicySetVersion, path string) error
}

// policySetVersions implements Policy Set Versions.
type policySetVersions struct {
	client *Client
}

// PolciySetVersionSource represents a source type of a policy set version.
type PolciySetVersionSource string

// List all available run sources.
const (
	PolciySetVersionSourceAPI       PolciySetVersionSource = "tfe-api"
	PolciySetVersionSourceADO       PolciySetVersionSource = "ado"
	PolciySetVersionSourceBitBucket PolciySetVersionSource = "bitbucket"
	PolciySetVersionSourceGitHub    PolciySetVersionSource = "github"
	PolciySetVersionSourceGitLab    PolciySetVersionSource = "gitlab"
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
	Source           string                           `jsonapi:"attr,source"`
	Status           PolciySetVersionSource           `jsonapi:"attr,status"`
	StatusTimestamps PolciySetVersionStatusTimestamps `jsonapi:"attr,status-timestamps"`
	Error            string                           `jsonapi:"attr,error"`
	CreatedAt        time.Time                        `jsonapi:"attr,created-at,iso8601"`
	UpdatedAt        time.Time                        `jsonapi:"attr,updated-at,iso8601"`

	// Relations
	PolicySet *PolicySet `jsonapi:"relation,policy-set"`

	// Links
	Links map[string]interface{} `jsonapi:"links,omitempty"`
}

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

func (p *policySetVersions) Upload(ctx context.Context, psv PolicySetVersion, path string) error {
	uploadURL, ok := psv.Links["upload"].(string)
	if !ok {
		return fmt.Errorf("The Policy Set Version does not contain an upload link.")
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
