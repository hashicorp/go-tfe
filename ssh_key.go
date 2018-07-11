package tfe

import (
	"context"
	"errors"
	"fmt"
	"net/url"
)

// SSHKeys handles communication with the SSH key related methods of the
// Terraform Enterprise API.
//
// TFE API docs:
// https://www.terraform.io/docs/enterprise/api/ssh-keys.html
type SSHKeys struct {
	client *Client
}

// SSHKey represents a SSH key.
type SSHKey struct {
	ID   string `jsonapi:"primary,ssh-keys"`
	Name string `jsonapi:"attr,name"`
}

// SSHKeyListOptions represents the options for listing SSH keys.
type SSHKeyListOptions struct {
	ListOptions
}

// List returns all the organizations visible to the current user.
func (s *SSHKeys) List(ctx context.Context, organization string, options SSHKeyListOptions) ([]*SSHKey, error) {
	if !validStringID(&organization) {
		return nil, errors.New("Invalid value for organization")
	}

	u := fmt.Sprintf("organizations/%s/ssh-keys", url.QueryEscape(organization))
	req, err := s.client.newRequest("GET", u, &options)
	if err != nil {
		return nil, err
	}

	var ks []*SSHKey
	err = s.client.do(ctx, req, &ks)
	if err != nil {
		return nil, err
	}

	return ks, nil
}

// SSHKeyCreateOptions represents the options for creating an SSH key.
type SSHKeyCreateOptions struct {
	// For internal use only!
	ID string `jsonapi:"primary,ssh-keys"`

	// A name to identify the SSH key.
	Name *string `jsonapi:"attr,name"`

	// The content of the SSH private key.
	Value *string `jsonapi:"attr,value"`
}

func (o SSHKeyCreateOptions) valid() error {
	if !validString(o.Name) {
		return errors.New("Name is required")
	}
	if !validString(o.Value) {
		return errors.New("Value is required")
	}
	return nil
}

// Create an SSH key and associate it with an organization.
func (s *SSHKeys) Create(ctx context.Context, organization string, options SSHKeyCreateOptions) (*SSHKey, error) {
	if !validStringID(&organization) {
		return nil, errors.New("Invalid value for organization")
	}

	if err := options.valid(); err != nil {
		return nil, err
	}

	// Make sure we don't send a user provided ID.
	options.ID = ""

	u := fmt.Sprintf("organizations/%s/ssh-keys", url.QueryEscape(organization))
	req, err := s.client.newRequest("POST", u, &options)
	if err != nil {
		return nil, err
	}

	k := &SSHKey{}
	err = s.client.do(ctx, req, k)
	if err != nil {
		return nil, err
	}

	return k, nil
}

// Read an SSH key.
func (s *SSHKeys) Read(ctx context.Context, sshKeyID string) (*SSHKey, error) {
	if !validStringID(&sshKeyID) {
		return nil, errors.New("Invalid value for SSH key ID")
	}

	u := fmt.Sprintf("ssh-keys/%s", url.QueryEscape(sshKeyID))
	req, err := s.client.newRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	k := &SSHKey{}
	err = s.client.do(ctx, req, k)
	if err != nil {
		return nil, err
	}

	return k, nil
}

// SSHKeyUpdateOptions represents the options for updating an SSH key.
type SSHKeyUpdateOptions struct {
	// For internal use only!
	ID string `jsonapi:"primary,ssh-keys"`

	// A new name to identify the SSH key.
	Name *string `jsonapi:"attr,name,omitempty"`

	// Updated content of the SSH private key.
	Value *string `jsonapi:"attr,value,omitempty"`
}

// Update an SSH key.
func (s *SSHKeys) Update(ctx context.Context, sshKeyID string, options SSHKeyUpdateOptions) (*SSHKey, error) {
	if !validStringID(&sshKeyID) {
		return nil, errors.New("Invalid value for SSH key ID")
	}

	// Make sure we don't send a user provided ID.
	options.ID = ""

	u := fmt.Sprintf("ssh-keys/%s", url.QueryEscape(sshKeyID))
	req, err := s.client.newRequest("PATCH", u, &options)
	if err != nil {
		return nil, err
	}

	k := &SSHKey{}
	err = s.client.do(ctx, req, k)
	if err != nil {
		return nil, err
	}

	return k, nil
}

// Delete an SSH key.
func (s *SSHKeys) Delete(ctx context.Context, sshKeyID string) error {
	if !validStringID(&sshKeyID) {
		return errors.New("Invalid value for SSH key ID")
	}

	u := fmt.Sprintf("ssh-keys/%s", url.QueryEscape(sshKeyID))
	req, err := s.client.newRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	return s.client.do(ctx, req, nil)
}
