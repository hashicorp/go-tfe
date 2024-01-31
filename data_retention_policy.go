// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

import "regexp"

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

// convert to legacy DataRetentionPolicy struct
func (d *DataRetentionPolicyChoice) ConvertToLegacyStruct() *DataRetentionPolicy {
	if d.DataRetentionPolicy != nil {
		// TFE v202311-1 and v202312-1 will return a deprecated DataRetentionPolicy in the DataRetentionPolicyChoice struct
		return d.DataRetentionPolicy
	} else if d.DataRetentionPolicyDeleteOlder != nil {
		// DataRetentionPolicy was functionally replaced by DataRetentionPolicyDeleteOlder in TFE v202401
		return &DataRetentionPolicy{
			ID:                   d.DataRetentionPolicyDeleteOlder.ID,
			DeleteOlderThanNDays: d.DataRetentionPolicyDeleteOlder.DeleteOlderThanNDays,
		}
	}
	return nil
}

// Deprecated: Use DataRetentionPolicyDeleteOlder instead. This is the original representation of a
// data retention policy, only present in TFE v202311-1 and v202312-1
type DataRetentionPolicy struct {
	ID                   string `jsonapi:"primary,data-retention-policies"`
	DeleteOlderThanNDays int    `jsonapi:"attr,delete-older-than-n-days"`
}

// Deprecated: Use DataRetentionPolicyDeleteOlder variations instead
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

// error we get when trying to unmarshal a data retention policy from TFE v202401+ into the deprecated DataRetentionPolicy struct
var drpUnmarshalEr = regexp.MustCompile(`Trying to Unmarshal an object of type \".+\", but \"data-retention-policies\" does not match`)
