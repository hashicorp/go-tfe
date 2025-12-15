// Copyright IBM Corp. 2018, 2025
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_validateAdminRunFilterParams(t *testing.T) {
	// All RunStatus values - keep this in sync with run.go
	validRunStatuses := []string{
		"applied",
		"applying",
		"apply_queued",
		"canceled",
		"confirmed",
		"cost_estimated",
		"cost_estimating",
		"discarded",
		"errored",
		"fetching",
		"fetching_completed",
		"pending",
		"planned",
		"planned_and_finished",
		"planned_and_saved",
		"planning",
		"plan_queued",
		"policy_checked",
		"policy_checking",
		"policy_override",
		"policy_soft_failed",
		"post_plan_awaiting_decision",
		"post_plan_completed",
		"post_plan_running",
		"pre_apply_running",
		"pre_apply_completed",
		"pre_plan_completed",
		"pre_plan_running",
		"queuing",
		"queuing_apply",
	}
	for _, v := range validRunStatuses {
		t.Run(v, func(t *testing.T) {
			require.NoError(t, validateAdminRunFilterParams(v), fmt.Sprintf("'%s' should be valid", v))
		})
	}

	// empty string is allowed
	require.NoError(t, validateAdminRunFilterParams(""), "empty string should be valid")

	// comma-separated list, all valid
	require.NoError(t, validateAdminRunFilterParams("applied,planned,canceled"), "'applied,planned,canceled' should be valid)")

	// invalid values
	require.Error(t, validateAdminRunFilterParams("cost_estimate"), "invalid value: cost_estimate")

	// comma-separated list, some invalid
	require.Error(t, validateAdminRunFilterParams("applied,not-planned,canceled"), "'applied,not-planned,canceled' should be invalid)")
}
