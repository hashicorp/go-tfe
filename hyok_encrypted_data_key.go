package tfe

import (
	"context"
	"fmt"
	"net/url"
	"time"
)

var _ HYOKEncryptedDataKeys = (*hyokEncryptedDataKeys)(nil)

// HYOKEncryptedDataKeys describes all the hyok customer key version related methods that the HCP Terraform API supports.
// HCP Terraform API docs:
// https://developer.hashicorp.com/terraform/cloud-docs/api-docs/hold-your-own-key/encrypted-data-keys
type HYOKEncryptedDataKeys interface {
	// Read a HYOK encrypted data key by its ID.
	Read(ctx context.Context, hyokEncryptedDataKeyID string) (*HYOKEncryptedDataKey, error)
}

// hyokEncryptedDataKeys implements HYOKEncryptedDataKeys
type hyokEncryptedDataKeys struct {
	client *Client
}

// HYOKEncryptedDataKey represents the resource
type HYOKEncryptedDataKey struct {
	// Attributes
	ID              string    `jsonapi:"primary,hyok-encrypted-data-keys"`
	EncryptedDEK    string    `jsonapi:"attr,encrypted-dek"`
	CustomerKeyName string    `jsonapi:"attr,customer-key-name"`
	CreatedAt       time.Time `jsonapi:"attr,created-at,iso8601"`

	// Relationships
	KeyVersion *HYOKCustomerKeyVersion `jsonapi:"relation,hyok-customer-key-versions"`
}

// Read a HYOK encrypted data key by its ID.
func (h hyokEncryptedDataKeys) Read(ctx context.Context, hyokEncryptedDataKeyID string) (*HYOKEncryptedDataKey, error) {
	if !validStringID(&hyokEncryptedDataKeyID) {
		return nil, ErrInvalidHYOKEncryptedDataKey
	}

	path := fmt.Sprintf("hyok-encrypted-data-keys/%s", url.PathEscape(hyokEncryptedDataKeyID))
	req, err := h.client.NewRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}

	dek := &HYOKEncryptedDataKey{}
	err = req.Do(ctx, dek)
	if err != nil {
		return nil, err
	}

	return dek, nil
}
