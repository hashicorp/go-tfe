package tfe

import (
	"context"
	"fmt"
)

// Compile-time proof of interface implementation.
var _ GHAInstallations = (*gHAInstallations)(nil)

// GHAInstallations describes all the Github App Installations related methods that the
// Terraform Enterprise API supports.
//
// TFE API docs:
// https://www.terraform.io/docs/cloud/api/gha-installations.html
type GHAInstallations interface {
	// List all the Github App for the user.
	List(ctx context.Context, options *GHAInstallationListOptions) (*GHAInstallationList, error)
}

// gHAInstallations implements GHAInstallations.
type gHAInstallations struct {
	client *Client
}

// GHAInstallationList represents a list of github installations.
type GHAInstallationList struct {
	*Pagination
	Items []*GHAInstallation
}

// GHAInstallation represents a github app installation
type GHAInstallation struct {
	ID             string `jsonapi:"primary,github-app-installations"`
	InstallationId int32  `jsonapi:"attr,installation-id"`
	Name           string `jsonapi:"attr,name"`
}

// GHAInstallationListOptions represents the options for listing.
type GHAInstallationListOptions struct {
	ListOptions
}

// List all the github app installations.
func (s *gHAInstallations) List(ctx context.Context, options *GHAInstallationListOptions) (*GHAInstallationList, error) {
	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("github-app/installations")
	req, err := s.client.NewRequest("GET", u, options)
	fmt.Println(u)
	if err != nil {
		return nil, err
	}

	otl := &GHAInstallationList{}

	err = req.Do(ctx, otl)
	if err != nil {
		return nil, err
	}

	fmt.Println(otl.Items[0])
	return otl, nil
}

func (o *GHAInstallationListOptions) valid() error {
	return nil
}
