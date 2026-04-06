// Copyright IBM Corp. 2018, 2025
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"os"
	"testing"
)

func stackVCSRepoIdentifier(t *testing.T) string {
	t.Helper()

	githubIdentifier := os.Getenv("GITHUB_STACK_IDENTIFIER")
	if githubIdentifier == "" {
		t.Skip("Export a valid GITHUB_STACK_IDENTIFIER before running this test")
	}

	return githubIdentifier
}

func stackVCSRepoBranch() string {
	githubBranch := os.Getenv("GITHUB_STACK_REPO_BRANCH")
	if githubBranch == "" {
		return "main"
	}

	return githubBranch
}
