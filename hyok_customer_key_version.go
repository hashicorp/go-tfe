package tfe

import (
	"context"
	"fmt"
	"net/url"
	"time"
)

// HYOKCustomerKeyVersions describes all the hyok customer key version related methods that the HCP Terraform API supports.
type HYOKCustomerKeyVersions interface {
	// List all hyok customer key versions associated to a HYOK configuration.
	List(ctx context.Context, hyokConfigurationID string, options *HYOKCustomerKeyVersionListOptions) (*HYOKCustomerKeyVersionList, error)

	// Read a hyok customer key version by its ID.
	Read(ctx context.Context, hyokCustomerKeyVersionID string) (*HYOKCustomerKeyVersion, error)

	// Revoke a hyok customer key version.
	Revoke(ctx context.Context, hyokCustomerKeyVersionID string) error

	// Delete a hyok customer key version.
	Delete(ctx context.Context, hyokCustomerKeyVersionID string) error
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
	ID         string               `jsonapi:"primary,hyok-customer-key-version"`
	KeyVersion string               `jsonapi:"attr,key-version"`
	CreatedAt  time.Time            `jsonapi:"attr,created-at,iso8601"`
	UpdatedAt  time.Time            `jsonapi:"attr,updated-at,iso8601"`
	RevokedAt  time.Time            `jsonapi:"attr,revoked-at,iso8601"`
	Status     HYOKKeyVersionStatus `jsonapi:"attr,status"`
	Error      string               `jsonapi:"attr,error"`
}

// HYOKKeyVersionStatus represents a key version status.
type HYOKKeyVersionStatus string

// List all available configuration version statuses.
const (
	KeyVersionStatusAvailable        HYOKKeyVersionStatus = "available"
	KeyVersionStatusRevoking         HYOKKeyVersionStatus = "revoking"
	KeyVersionStatusRevoked          HYOKKeyVersionStatus = "revoked"
	KeyVersionStatusRevocationFailed HYOKKeyVersionStatus = "revocation_failed"
)

// HYOKCustomerKeyVersionListOptions represents the options for listing hyok customer key versions
type HYOKCustomerKeyVersionListOptions struct {
	ListOptions
	Refresh bool `query:"refresh"`
}

// List all hyok customer key versions.
func (s *hyokCustomerKeyVersions) List(ctx context.Context, hyokConfigurationID string, options *HYOKCustomerKeyVersionListOptions) (*HYOKCustomerKeyVersionList, error) {
	if !validStringID(&hyokConfigurationID) {
		return nil, ErrInvalidHYOK
	}

	path := fmt.Sprintf("hyok-configurations/%s/hyok-customer-key-versions", url.PathEscape(hyokConfigurationID))
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
func (s *hyokCustomerKeyVersions) Read(ctx context.Context, hyokCustomerKeyVersionID string) (*HYOKCustomerKeyVersion, error) {
	if !validStringID(&hyokCustomerKeyVersionID) {
		return nil, ErrInvalidHYOK
	}

	path := fmt.Sprintf("hyok-customer-key-versions/%s", url.PathEscape(hyokCustomerKeyVersionID))
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
func (s *hyokCustomerKeyVersions) Revoke(ctx context.Context, hyokCustomerKeyVersionID string) error {
	if !validStringID(&hyokCustomerKeyVersionID) {
		return ErrInvalidHYOK
	}

	path := fmt.Sprintf("hyok-customer-key-versions/%s/actions/revoke", url.PathEscape(hyokCustomerKeyVersionID))
	req, err := s.client.NewRequest("POST", path, nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

// Delete a hyok customer key version.
func (s *hyokCustomerKeyVersions) Delete(ctx context.Context, hyokCustomerKeyVersionID string) error {
	if !validStringID(&hyokCustomerKeyVersionID) {
		return ErrInvalidHYOK
	}

	path := fmt.Sprintf("hyok-customer-key-versions/%s", url.PathEscape(hyokCustomerKeyVersionID))
	req, err := s.client.NewRequest("DELETE", path, nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}
