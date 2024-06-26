// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidUnifiedID(t *testing.T) {
	type testCase struct {
		externalID    *string
		expectedValue bool
	}

	unifiedTeamID := "iam.group:kmpwhkwf6tkgWzgJKPcP"
	unifiedProjectID := "616f63c1-3ef5-46a5-b5e8-6d1d86c3f93f"
	nonUnifiedID := "prj-AywVvpbtLQTcwf8K"
	invalidID := "test/with-a-slash"

	cases := map[string]testCase{
		"external-id-is-nil":                {externalID: nil, expectedValue: false},
		"external-id-is-empty-string":       {externalID: new(string), expectedValue: false},
		"external-id-is-invalid-with-slash": {externalID: &invalidID, expectedValue: false},
		"external-id-is-unified-team-id":    {externalID: &unifiedTeamID, expectedValue: true},
		"external-id-is-unified-project-id": {externalID: &unifiedProjectID, expectedValue: true},
		"external-id-is-non-unified":        {externalID: &nonUnifiedID, expectedValue: true},
	}

	for name, tcase := range cases {
		t.Run(name, func(tt *testing.T) {
			actual := validUnifiedID(tcase.externalID)
			assert.Equal(tt, tcase.expectedValue, actual)
		})
	}
}
