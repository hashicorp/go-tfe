package tfe

import (
	"context"
	"fmt"
	"net/url"
	"time"
)

// HyokCustomerKeyVersions describes all the hyok customer key version related methods that the HCP Terraform API supports.
//
// TFE API docs: (TODO: ADD DOCS URL)
type HyokCustomerKeyVersions interface {
	// List all hyok customer key versions associated to a HYOK configuration.
	List(ctx context.Context, hyokConfiguration string, options *HyokCustomerKeyVersionListOptions) (*HyokCustomerKeyVersionList, error)

	// Read a hyok customer key version by its ID.
	Read(ctx context.Context, HyokCustomerKeyVersionID string) (*HyokCustomerKeyVersion, error)

	// Revoke a hyok customer key version.
	Revoke(ctx context.Context, HyokCustomerKeyVersionID string) error

	// Delete a hyok customer key version.
	Delete(ctx context.Context, HyokCustomerKeyVersionID string) error
}

// hyokCustomerKeyVersions implements HyokCustomerKeyVersions
type hyokCustomerKeyVersions struct {
	client *Client
}

// HyokCustomerKeyVersionList represents a list of hyok customer key versions
type HyokCustomerKeyVersionList struct {
	*Pagination
	Items []*HyokCustomerKeyVersion
}

// HyokCustomerKeyVersion represents a Terraform Enterprise $resource
type HyokCustomerKeyVersion struct {
	ID         string           `jsonapi:"primary,hyok-customer-key-version"`
	KeyVersion string           `jsonapi:"attr,key-version"`
	CreatedAt  time.Time        `jsonapi:"attr,created-at,iso8601"`
	UpdatedAt  time.Time        `jsonapi:"attr,updated-at,iso8601"`
	RevokedAt  time.Time        `jsonapi:"attr,revoked-at,iso8601"`
	Status     KeyVersionStatus `jsonapi:"attr,status"`
	Error      string           `jsonapi:"attr,error"`
}

// KeyVersionStatus represents a key version status.
type KeyVersionStatus string

// List all available configuration version statuses.
const (
	KeyVersionStatusAvailable        KeyVersionStatus = "available"
	KeyVersionStatusRevoking         KeyVersionStatus = "revoking"
	KeyVersionStatusRevoked          KeyVersionStatus = "revoked"
	KeyVersionStatusRevocationFailed KeyVersionStatus = "revocation_failed"
)

// HyokCustomerKeyVersionListOptions represents the options for listing hyok customer key versions
type HyokCustomerKeyVersionListOptions struct {
	ListOptions
	Refresh bool `query:"refresh"`
}

func (o *HyokCustomerKeyVersionListOptions) valid() error {
	return nil
}

// HyokCustomerKeyVersionCreateOptions represents the options for creating a hyok customer key version
type HyokCustomerKeyVersionCreateOptions struct {
	Type string `jsonapi:"primary,hyok-customer-key-version"`
	// Add more create options here
}

// HyokCustomerKeyVersionUpdateOptions represents the options for updating a hyok customer key version
type HyokCustomerKeyVersionUpdateOptions struct {
	ID string `jsonapi:"primary,hyok-customer-key-version"`

	// Add more update options here
}

// List all hyok customer key versions.
func (s *hyokCustomerKeyVersions) List(ctx context.Context, hyokConfiguration string, options *HyokCustomerKeyVersionListOptions) (*HyokCustomerKeyVersionList, error) {
	if !validStringID(&hyokConfiguration) {
		return nil, ErrInvalidHyokConfigID
	}

	// TODO: DO I NEED TO CHECK THIS?
	//if err := options.valid(); err != nil {
	//	return nil, err
	//}

	path := fmt.Sprintf("hyok-configurations/%s/hyok-customer-key-versions", url.PathEscape(hyokConfiguration))
	req, err := s.client.NewRequest("GET", path, options)
	if err != nil {
		return nil, err
	}

	kvs := &HyokCustomerKeyVersionList{}
	err = req.Do(ctx, kvs)
	if err != nil {
		return nil, err
	}

	return kvs, nil
}

// Read a hyok customer key version by its ID.
func (s *hyokCustomerKeyVersions) Read(ctx context.Context, HyokCustomerKeyVersionID string) (*HyokCustomerKeyVersion, error) {
	if !validStringID(&HyokCustomerKeyVersionID) {
		return nil, ErrInvalidHyokConfigID
	}

	path := fmt.Sprintf("hyok-customer-key-versions/%s", url.PathEscape(HyokCustomerKeyVersionID))
	req, err := s.client.NewRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}

	kv := &HyokCustomerKeyVersion{}
	err = req.Do(ctx, kv)
	if err != nil {
		return nil, err
	}

	return kv, nil

}

// Revoke a hyok customer key version.
func (s *hyokCustomerKeyVersions) Revoke(ctx context.Context, HyokCustomerKeyVersionID string) error {
	if !validStringID(&HyokCustomerKeyVersionID) {
		return ErrInvalidHyokConfigID
	}

	path := fmt.Sprintf("hyok-customer-key-versions/%s/actions/revoke", url.PathEscape(HyokCustomerKeyVersionID))
	req, err := s.client.NewRequest("POST", path, nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

// Delete a hyok customer key version.
func (s *hyokCustomerKeyVersions) Delete(ctx context.Context, HyokCustomerKeyVersionID string) error {
	if !validStringID(&HyokCustomerKeyVersionID) {
		return ErrInvalidHyokConfigID
	}

	path := fmt.Sprintf("hyok-customer-key-versions/%s", url.PathEscape(HyokCustomerKeyVersionID))
	req, err := s.client.NewRequest("DELETE", path, nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}
