package tfe

import (
	"context"
	"fmt"
	"net/url"
	"time"
)

// HYOKCustomerKeyVersions describes all the hyok customer key version related methods that the HCP Terraform API supports.
//
// TFE API docs: (TODO: ADD DOCS URL)
type HYOKCustomerKeyVersions interface {
	// List all HYOK customer key versions associated to a HYOK configuration.
	List(ctx context.Context, hyokConfiguration string, options *HYOKCustomerKeyVersionListOptions) (*HYOKCustomerKeyVersionList, error)

	// Read a HYOK customer key version by its ID.
	Read(ctx context.Context, HYOKCustomerKeyVersionID string) (*HYOKCustomerKeyVersion, error)

	// Revoke a HYOK customer key version.
	Revoke(ctx context.Context, HYOKCustomerKeyVersionID string) error

	// Delete a HYOK customer key version.
	Delete(ctx context.Context, HYOKCustomerKeyVersionID string) error
}

// hyokCustomerKeyVersions implements HYOKCustomerKeyVersions
type hyokCustomerKeyVersions struct {
	client *Client
}

// HYOKCustomerKeyVersionList represents a list of hyok customer key versions
type HYOKCustomerKeyVersionList struct {
	*Pagination
	Items []*HYOKCustomerKeyVersion
}

// HYOKCustomerKeyVersion represents a Terraform Enterprise $resource
type HYOKCustomerKeyVersion struct {
	ID         string           `jsonapi:"primary,hyok-customer-key-versions"`
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

// HYOKCustomerKeyVersionListOptions represents the options for listing hyok customer key versions
type HYOKCustomerKeyVersionListOptions struct {
	ListOptions
	Refresh bool `query:"refresh"`
}

func (o *HYOKCustomerKeyVersionListOptions) valid() error {
	return nil
}

// HYOKCustomerKeyVersionCreateOptions represents the options for creating a hyok customer key version
type HYOKCustomerKeyVersionCreateOptions struct {
	ID string `jsonapi:"primary,hyok-customer-key-versions"`
	// Add more create options here
}

// HYOKCustomerKeyVersionUpdateOptions represents the options for updating a hyok customer key version
type HYOKCustomerKeyVersionUpdateOptions struct {
	ID string `jsonapi:"primary,hyok-customer-key-versions"`

	// Add more update options here
}

// List all hyok customer key versions.
func (s *hyokCustomerKeyVersions) List(ctx context.Context, hyokConfiguration string, options *HYOKCustomerKeyVersionListOptions) (*HYOKCustomerKeyVersionList, error) {
	if !validStringID(&hyokConfiguration) {
		return nil, ErrInvalidHYOK
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

	kvs := &HYOKCustomerKeyVersionList{}
	err = req.Do(ctx, kvs)
	if err != nil {
		return nil, err
	}

	return kvs, nil
}

// Read a hyok customer key version by its ID.
func (s *hyokCustomerKeyVersions) Read(ctx context.Context, HYOKCustomerKeyVersionID string) (*HYOKCustomerKeyVersion, error) {
	if !validStringID(&HYOKCustomerKeyVersionID) {
		return nil, ErrInvalidHYOK
	}

	path := fmt.Sprintf("hyok-customer-key-versions/%s", url.PathEscape(HYOKCustomerKeyVersionID))
	req, err := s.client.NewRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}

	kv := &HYOKCustomerKeyVersion{}
	err = req.Do(ctx, kv)
	if err != nil {
		return nil, err
	}

	return kv, nil

}

// Revoke a hyok customer key version.
func (s *hyokCustomerKeyVersions) Revoke(ctx context.Context, HYOKCustomerKeyVersionID string) error {
	if !validStringID(&HYOKCustomerKeyVersionID) {
		return ErrInvalidHYOK
	}

	path := fmt.Sprintf("hyok-customer-key-versions/%s/actions/revoke", url.PathEscape(HYOKCustomerKeyVersionID))
	req, err := s.client.NewRequest("POST", path, nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

// Delete a hyok customer key version.
func (s *hyokCustomerKeyVersions) Delete(ctx context.Context, HYOKCustomerKeyVersionID string) error {
	if !validStringID(&HYOKCustomerKeyVersionID) {
		return ErrInvalidHYOK
	}

	path := fmt.Sprintf("hyok-customer-key-versions/%s", url.PathEscape(HYOKCustomerKeyVersionID))
	req, err := s.client.NewRequest("DELETE", path, nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}
