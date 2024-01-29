// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

type DataRetentionPolicy struct {
	ID                   string `jsonapi:"primary,data-retention-policies"`
	DeleteOlderThanNDays int    `jsonapi:"attr,delete-older-than-n-days"`
}

type DataRetentionPolicySetOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,data-retention-policies"`

	DeleteOlderThanNDays int `jsonapi:"attr,delete-older-than-n-days"`
}
