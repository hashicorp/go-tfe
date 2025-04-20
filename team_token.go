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
var _ TeamTokens = (*teamTokens)(nil)

// TeamTokens describes all the team token related methods that the
// Terraform Enterprise API supports.
//
// TFE API docs:
// https://developer.hashicorp.com/terraform/cloud-docs/api-docs/team-tokens
type TeamTokens interface {
	// Create a new team token using the legacy creation behavior, which creates a token without a description
	// or regenerates the existing, descriptionless token.
	Create(ctx context.Context, teamID string) (*TeamToken, error)

	// CreateWithOptions creates a team token, with options. If no description is provided, it uses the legacy
	// creation behavior, which regenerates the descriptionless token if it already exists. Otherwise, it create
	//  a new token with the given unique description, allowing for the creation of multiple team tokens.
	CreateWithOptions(ctx context.Context, teamID string, options TeamTokenCreateOptions) (*TeamToken, error)

	// Read a team token by its team ID.
	Read(ctx context.Context, teamID string) (*TeamToken, error)

	// Read a team token by its token ID.
	ReadByID(ctx context.Context, teamID string) (*TeamToken, error)

	// Delete a team token by its team ID.
	Delete(ctx context.Context, teamID string) error

	// Delete a team token by its token ID.
	DeleteByID(ctx context.Context, tokenID string) error
}

// teamTokens implements TeamTokens.
type teamTokens struct {
	client *Client
}

// TeamToken represents a Terraform Enterprise team token.
type TeamToken struct {
	ID          string           `jsonapi:"primary,authentication-tokens"`
	CreatedAt   time.Time        `jsonapi:"attr,created-at,iso8601"`
	Description *string          `jsonapi:"attr,description"`
	LastUsedAt  time.Time        `jsonapi:"attr,last-used-at,iso8601"`
	Token       string           `jsonapi:"attr,token"`
	ExpiredAt   time.Time        `jsonapi:"attr,expired-at,iso8601"`
	CreatedBy   *CreatedByChoice `jsonapi:"polyrelation,created-by"`
	Team        *Team            `jsonapi:"relation,team"`
}

// TeamTokenCreateOptions contains the options for creating a team token.
type TeamTokenCreateOptions struct {
	// Optional: The token's expiration date.
	// This feature is available in TFE release v202305-1 and later
	ExpiredAt *time.Time `jsonapi:"attr,expired-at,iso8601,omitempty"`

	// Optional: The token's description, which must unique per team.
	// This feature is considered BETA, SUBJECT TO CHANGE, and likely unavailable to most users.
	Description *string `jsonapi:"attr,description,omitempty"`
}

// Create a new team token using the legacy creation behavior, which creates a token without a description
// or regenerates the existing, descriptionless token.
func (s *teamTokens) Create(ctx context.Context, teamID string) (*TeamToken, error) {
	return s.CreateWithOptions(ctx, teamID, TeamTokenCreateOptions{})
}

// CreateWithOptions creates a team token, with options. If no description is provided, it uses the legacy
// creation behavior, which regenerates the descriptionless token if it already exists. Otherwise, it create
// a new token with the given unique description, allowing for the creation of multiple team tokens.
func (s *teamTokens) CreateWithOptions(ctx context.Context, teamID string, options TeamTokenCreateOptions) (*TeamToken, error) {
	if !validStringID(&teamID) {
		return nil, ErrInvalidTeamID
	}

	var u string
	if options.Description != nil {
		u = fmt.Sprintf("teams/%s/authentication-tokens", url.PathEscape(teamID))
	} else {
		u = fmt.Sprintf("teams/%s/authentication-token", url.PathEscape(teamID))
	}

	req, err := s.client.NewRequest("POST", u, &options)
	if err != nil {
		return nil, err
	}

	tt := &TeamToken{}
	err = req.Do(ctx, tt)
	if err != nil {
		return nil, err
	}

	return tt, err
}

// Read a team token by its team ID.
func (s *teamTokens) Read(ctx context.Context, teamID string) (*TeamToken, error) {
	if !validStringID(&teamID) {
		return nil, ErrInvalidTeamID
	}

	u := fmt.Sprintf("teams/%s/authentication-token", url.PathEscape(teamID))
	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	tt := &TeamToken{}
	err = req.Do(ctx, tt)
	if err != nil {
		return nil, err
	}

	return tt, err
}

// Read a team token by its token ID.
func (s *teamTokens) ReadByID(ctx context.Context, tokenID string) (*TeamToken, error) {
	if !validStringID(&tokenID) {
		return nil, ErrInvalidTokenID
	}

	u := fmt.Sprintf(AuthenticationTokensPath, url.PathEscape(tokenID))
	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	tt := &TeamToken{}
	err = req.Do(ctx, tt)
	if err != nil {
		return nil, err
	}

	return tt, err
}

// Delete a team token by its team ID.
func (s *teamTokens) Delete(ctx context.Context, teamID string) error {
	if !validStringID(&teamID) {
		return ErrInvalidTeamID
	}

	u := fmt.Sprintf("teams/%s/authentication-token", url.PathEscape(teamID))
	req, err := s.client.NewRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}

// Delete a team token by its token ID.
func (s *teamTokens) DeleteByID(ctx context.Context, tokenID string) error {
	if !validStringID(&tokenID) {
		return ErrInvalidTokenID
	}

	u := fmt.Sprintf(AuthenticationTokensPath, url.PathEscape(tokenID))
	req, err := s.client.NewRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	return req.Do(ctx, nil)
}
