// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

// DataRetentionPolicyChoice is a choice type struct that represents the possible types
// of a drp returned by a polymorphic relationship. If a value is available, exactly one field
// will be non-nil.
type DataRetentionPolicyChoice struct {
	DataRetentionPolicy            *DataRetentionPolicy
	DataRetentionPolicyDeleteOlder *DataRetentionPolicyDeleteOlder
	DataRetentionPolicyDontDelete  *DataRetentionPolicyDontDelete
}

// Returns whether one of the choices is populated
func (d DataRetentionPolicyChoice) IsPopulated() bool {
	return d.DataRetentionPolicy != nil ||
		d.DataRetentionPolicyDeleteOlder != nil ||
		d.DataRetentionPolicyDontDelete != nil
}

// DEPRECATED: Use DataRetentionPolicyDeleteOlder instead. This is the original representation of a
// data retention policy, only present in TFE v202311-1 and v202312-1
type DataRetentionPolicy struct {
	ID                   string `jsonapi:"primary,data-retention-policies"`
	DeleteOlderThanNDays int    `jsonapi:"attr,delete-older-than-n-days"`
}

// DEPRECATED: Use DataRetentionPolicyDeleteOlder variations instead
type DataRetentionPolicySetOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,data-retention-policies"`

	DeleteOlderThanNDays int `jsonapi:"attr,delete-older-than-n-days"`
}

type DataRetentionPolicyDeleteOlder struct {
	ID                   string `jsonapi:"primary,data-retention-policy-delete-olders"`
	DeleteOlderThanNDays int    `jsonapi:"attr,delete-older-than-n-days"`
}

type DataRetentionPolicyDontDelete struct {
	ID string `jsonapi:"primary,data-retention-policy-dont-deletes"`
}

type DataRetentionPolicyDeleteOlderSetOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,data-retention-policy-delete-olders"`

	DeleteOlderThanNDays int `jsonapi:"attr,delete-older-than-n-days"`
}

type DataRetentionPolicyDontDeleteSetOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,data-retention-policy-dont-deletes"`
}
