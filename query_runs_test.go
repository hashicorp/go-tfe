// Copyright IBM Corp. 2014, 2026

package tfe

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQueryRunCreateOptions_PolicyPaths(t *testing.T) {
	opts := QueryRunCreateOptions{
		PolicyPaths: []string{"policies/compliance", "policies/security"},
	}

	assert.Equal(t, []string{"policies/compliance", "policies/security"}, opts.PolicyPaths)
}
