// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"fmt"
	"net/url"
	"time"
)

// Compile-time proof of interface implementation.
var _ TeamNotificationConfigurations = (*teamNotificationConfigurations)(nil)

// TeamNotificationConfigurations describes all the Team Notification Configuration
// related methods that the Terraform Enterprise API supports.
//
// TFE API docs:
// https://developer.hashicorp.com/terraform/cloud-docs/api-docs/notification-configurations#team-notification-configuration
type TeamNotificationConfigurations interface {
	// List all the notification configurations within a team.
	List(ctx context.Context, teamID string, options *TeamNotificationConfigurationListOptions) (*TeamNotificationConfigurationList, error)

	// Create a new team notification configuration with the given options.
	Create(ctx context.Context, teamID string, options TeamNotificationConfigurationCreateOptions) (*TeamNotificationConfiguration, error)

	// Read a notification configuration by its ID.
	Read(ctx context.Context, teamNotificationConfigurationID string) (*TeamNotificationConfiguration, error)

	// Update an existing team notification configuration.
	Update(ctx context.Context, teamNotificationConfigurationID string, options TeamNotificationConfigurationUpdateOptions) (*TeamNotificationConfiguration, error)

	// Delete a team notification configuration by its ID.
	Delete(ctx context.Context, teamNotificationConfigurationID string) error

	// Verify a team notification configuration by its ID.
	Verify(ctx context.Context, teamNotificationConfigurationID string) (*TeamNotificationConfiguration, error)
}

// teamNotificationConfigurations implements TeamNotificationConfigurations.
type teamNotificationConfigurations struct {
	client *Client
}

// TeamNotificationConfigurationList represents a list of team notification
// configurations.
type TeamNotificationConfigurationList struct {
	*Pagination
	Items []*TeamNotificationConfiguration
}

// TeamNotificationConfiguration represents a team notification configuration.
type TeamNotificationConfiguration struct {
	ID                string                      `jsonapi:"primary,notification-configurations"`
	CreatedAt         time.Time                   `jsonapi:"attr,created-at,iso8601"`
	DeliveryResponses []*DeliveryResponse         `jsonapi:"attr,delivery-responses"`
	DestinationType   NotificationDestinationType `jsonapi:"attr,destination-type"`
	Enabled           bool                        `jsonapi:"attr,enabled"`
	Name              string                      `jsonapi:"attr,name"`
	Token             string                      `jsonapi:"attr,token"`
	Triggers          []string                    `jsonapi:"attr,triggers"`
	UpdatedAt         time.Time                   `jsonapi:"attr,updated-at,iso8601"`
	URL               string                      `jsonapi:"attr,url"`

	// EmailAddresses is only available for TFE users. It is not available in HCP Terraform.
	EmailAddresses []string `jsonapi:"attr,email-addresses"`

	// Relations
	Subscribable *Team   `jsonapi:"relation,subscribable"`
	EmailUsers   []*User `jsonapi:"relation,users"`
}

// TeamNotificationConfigurationListOptions represents the options for listing
// notification configurations.
type TeamNotificationConfigurationListOptions struct {
	ListOptions
}

// TeamNotificationConfigurationCreateOptions represents the options for
// creating a new team notification configuration.
type TeamNotificationConfigurationCreateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,notification-configurations"`

	// Required: The destination type of the team notification configuration
	DestinationType *NotificationDestinationType `jsonapi:"attr,destination-type"`

	// Required: Whether the team notification configuration should be enabled or not
	Enabled *bool `jsonapi:"attr,enabled"`

	// Required: The name of the team notification configuration
	Name *string `jsonapi:"attr,name"`

	// Optional: The token of the team notification configuration
	Token *string `jsonapi:"attr,token,omitempty"`

	// Optional: The list of events that will trigger team notifications
	Triggers []NotificationTriggerType `jsonapi:"attr,triggers,omitempty"`

	// Optional: The URL of the team notification configuration
	URL *string `jsonapi:"attr,url,omitempty"`

	// Optional: The list of email addresses that will receive team notification emails.
	// EmailAddresses is only available for TFE users. It is not available in HCP Terraform.
	EmailAddresses []string `jsonapi:"attr,email-addresses,omitempty"`

	// Optional: The list of users belonging to the organization that will receive
	// team notification emails.
	EmailUsers []*User `jsonapi:"relation,users,omitempty"`
}

// TeamNotificationConfigurationUpdateOptions represents the options for
// updating a existing team notification configuration.
type TeamNotificationConfigurationUpdateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,notification-configurations"`

	// Optional: Whether the team notification configuration should be enabled or not
	Enabled *bool `jsonapi:"attr,enabled,omitempty"`

	// Optional: The name of the team notification configuration
	Name *string `jsonapi:"attr,name,omitempty"`

	// Optional: The token of the team notification configuration
	Token *string `jsonapi:"attr,token,omitempty"`

	// Optional: The list of events that will trigger team notifications
	Triggers []NotificationTriggerType `jsonapi:"attr,triggers,omitempty"`

	// Optional: The URL of the team notification configuration
	URL *string `jsonapi:"attr,url,omitempty"`

	// Optional: The list of email addresses that will receive team notification emails.
	// EmailAddresses is only available for TFE users. It is not available in HCP Terraform.
	EmailAddresses []string `jsonapi:"attr,email-addresses,omitempty"`

	// Optional: The list of users belonging to the organization that will receive
	// team notification emails.
	EmailUsers []*User `jsonapi:"relation,users,omitempty"`
}

// List all the notification configurations associated with a team.
func (s *teamNotificationConfigurations) List(ctx context.Context, teamID string, options *TeamNotificationConfigurationListOptions) (*TeamNotificationConfigurationList, error) {
	if !validStringID(&teamID) {
		return nil, ErrInvalidTeamID
	}

	u := fmt.Sprintf("teams/%s/notification-configurations", url.PathEscape(teamID))
	req, err := s.client.NewRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	ncl := &TeamNotificationConfigurationList{}
	err = req.Do(ctx, ncl)
	if err != nil {
		return nil, err
	}

	return ncl, nil
}

// Create a team notification configuration with the given options.
func (s *teamNotificationConfigurations) Create(ctx context.Context, teamID string, options TeamNotificationConfigurationCreateOptions) (*TeamNotificationConfiguration, error) {
	if !validStringID(&teamID) {
		return nil, ErrInvalidTeamID
	}
	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("teams/%s/notification-configurations", url.PathEscape(teamID))
	req, err := s.client.NewRequest("POST", u, &options)
	if err != nil {
		return nil, err
	}

	nc := &TeamNotificationConfiguration{}
	err = req.Do(ctx, nc)
	if err != nil {
		return nil, err
	}

	return nc, nil
}

// Read a team notification configuration by its ID.
func (s *teamNotificationConfigurations) Read(ctx context.Context, teamNotificationConfigurationID string) (*TeamNotificationConfiguration, error) {
	if !validStringID(&teamNotificationConfigurationID) {
		return nil, ErrInvalidNotificationConfigID
	}

	u := fmt.Sprintf("notification-configurations/%s", url.PathEscape(teamNotificationConfigurationID))
	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	nc := &TeamNotificationConfiguration{}
	err = req.Do(ctx, nc)
	if err != nil {
		return nil, err
	}

	return nc, nil
}

// Updates a team notification configuration with the given options.
func (s *teamNotificationConfigurations) Update(ctx context.Context, teamNotificationConfigurationID string, options TeamNotificationConfigurationUpdateOptions) (*TeamNotificationConfiguration, error) {
	if !validStringID(&teamNotificationConfigurationID) {
		return nil, ErrInvalidNotificationConfigID
	}

	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("notification-configurations/%s", url.PathEscape(teamNotificationConfigurationID))
	req, err := s.client.NewRequest("PATCH", u, &options)
	if err != nil {
		return nil, err
	}

	nc := &TeamNotificationConfiguration{}
	err = req.Do(ctx, nc)
	if err != nil {
		return nil, err
	}

	return nc, nil
}

// Delete a team notification configuration by its ID.
func (s *teamNotificationConfigurations) Delete(ctx context.Context, teamNotificationConfigurationID string) error {
	if !validStringID(&teamNotificationConfigurationID) {
		return ErrInvalidNotificationConfigID
	}

	u := fmt.Sprintf("notification-configurations/%s", url.PathEscape(teamNotificationConfigurationID))
	req, err := s.client.NewRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

// Verify a team notification configuration by delivering a verification payload
// to the configured URL.
func (s *teamNotificationConfigurations) Verify(ctx context.Context, teamNotificationConfigurationID string) (*TeamNotificationConfiguration, error) {
	if !validStringID(&teamNotificationConfigurationID) {
		return nil, ErrInvalidNotificationConfigID
	}

	u := fmt.Sprintf(
		"notification-configurations/%s/actions/verify", url.PathEscape(teamNotificationConfigurationID))
	req, err := s.client.NewRequest("POST", u, nil)
	if err != nil {
		return nil, err
	}

	nc := &TeamNotificationConfiguration{}
	err = req.Do(ctx, nc)
	if err != nil {
		return nil, err
	}

	return nc, nil
}

func (o TeamNotificationConfigurationCreateOptions) valid() error {
	if o.DestinationType == nil {
		return ErrRequiredDestinationType
	}
	if o.Enabled == nil {
		return ErrRequiredEnabled
	}
	if !validString(o.Name) {
		return ErrRequiredName
	}

	if !validTeamNotificationTriggerType(o.Triggers) {
		return ErrInvalidNotificationTrigger
	}

	if *o.DestinationType == NotificationDestinationTypeGeneric ||
		*o.DestinationType == NotificationDestinationTypeSlack ||
		*o.DestinationType == NotificationDestinationTypeMicrosoftTeams {
		if o.URL == nil {
			return ErrRequiredURL
		}
	}
	return nil
}

func (o TeamNotificationConfigurationUpdateOptions) valid() error {
	if o.Name != nil && !validString(o.Name) {
		return ErrRequiredName
	}

	if !validTeamNotificationTriggerType(o.Triggers) {
		return ErrInvalidNotificationTrigger
	}

	return nil
}

func validTeamNotificationTriggerType(triggers []NotificationTriggerType) bool {
	for _, t := range triggers {
		switch t {
		case
			NotificationTriggerChangeRequestCreated:
			continue
		default:
			return false
		}
	}

	return true
}
