package tfe

import (
	"errors"
	"fmt"
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
func (s *SSHKeys) List(organization string, options SSHKeyListOptions) ([]*SSHKey, error) {
	if !validStringID(&organization) {
		return nil, errors.New("Invalid value for organization")
	}

	u := fmt.Sprintf("organizations/%s/ssh-keys", organization)
	req, err := s.client.newRequest("GET", u, &options)
	if err != nil {
		return nil, err
	}

	result, err := s.client.do(req, []*SSHKey{})
	if err != nil {
		return nil, err
	}

	var ks []*SSHKey
	for _, k := range result.([]interface{}) {
		ks = append(ks, k.(*SSHKey))
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
func (s *SSHKeys) Create(organization string, options SSHKeyCreateOptions) (*SSHKey, error) {
	if !validStringID(&organization) {
		return nil, errors.New("Invalid value for organization")
	}

	if err := options.valid(); err != nil {
		return nil, err
	}

	// Make sure we don't send a user provided ID.
	options.ID = ""

	u := fmt.Sprintf("organizations/%s/ssh-keys", organization)
	req, err := s.client.newRequest("POST", u, &options)
	if err != nil {
		return nil, err
	}

	k, err := s.client.do(req, &SSHKey{})
	if err != nil {
		return nil, err
	}

	return k.(*SSHKey), nil
}

// Retrieve an SSH key.
func (s *SSHKeys) Retrieve(sshKeyID string) (*SSHKey, error) {
	if !validStringID(&sshKeyID) {
		return nil, errors.New("Invalid value for SSH key ID")
	}

	req, err := s.client.newRequest("GET", "ssh-keys/"+sshKeyID, nil)
	if err != nil {
		return nil, err
	}

	k, err := s.client.do(req, &SSHKey{})
	if err != nil {
		return nil, err
	}

	return k.(*SSHKey), nil
}

// SSHKeyUpdateOptions represents the options for updating an SSH key.
type SSHKeyUpdateOptions struct {
	// For internal use only!
	ID string `jsonapi:"primary,ssh-keys"`

	// A new name to identify the SSH key.
	Name *string `jsonapi:"attr,name"`

	// Updated content of the SSH private key.
	Value *string `jsonapi:"attr,value"`
}

// Update an SSH key.
func (s *SSHKeys) Update(sshKeyID string, options SSHKeyUpdateOptions) (*SSHKey, error) {
	if !validStringID(&sshKeyID) {
		return nil, errors.New("Invalid value for SSH key ID")
	}

	// Make sure we don't send a user provided ID.
	options.ID = ""

	req, err := s.client.newRequest("POST", "ssh-keys/"+sshKeyID, &options)
	if err != nil {
		return nil, err
	}

	k, err := s.client.do(req, &SSHKey{})
	if err != nil {
		return nil, err
	}

	return k.(*SSHKey), nil
}

// Delete an SSH key.
func (s *SSHKeys) Delete(sshKeyID string) error {
	if !validStringID(&sshKeyID) {
		return errors.New("Invalid value for SSH key ID")
	}

	req, err := s.client.newRequest("DELETE", "ssh-keys/"+sshKeyID, nil)
	if err != nil {
		return err
	}

	_, err = s.client.do(req, nil)

	return err
}
