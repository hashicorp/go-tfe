// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

type TestConfig struct {
	TestsEnabled       bool    `jsonapi:"attr,tests-enabled"`
	AgentExecutionMode *string `jsonapi:"attr,agent-execution-mode,omitempty"`
	AgentPoolID        *string `jsonapi:"attr,agent-pool-id,omitempty"`
}
