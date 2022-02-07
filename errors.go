package tfe

import (
	"errors"
)

// Generic errors applicable to all resources.
var (
	// ErrUnauthorized is returned when receiving a 401.
	ErrUnauthorized = errors.New("unauthorized")

	// ErrResourceNotFound is returned when receiving a 404.
	ErrResourceNotFound = errors.New("resource not found")

	// ErrRequiredName is returned when a name option is not present.
	ErrRequiredName = errors.New("name is required")

	// ErrInvalidName is returned when the name option has invalid value.
	ErrInvalidName = errors.New("invalid value for name")

	// ErrMissingDirectory is returned when the path does not have an existing directory.
	ErrMissingDirectory = errors.New("path needs to be an existing directory")
)

// Resource Errors
var (
	// ErrWorkspaceLocked is returned when trying to lock a
	// locked workspace.
	ErrWorkspaceLocked = errors.New("workspace already locked")

	// ErrWorkspaceNotLocked is returned when trying to unlock
	// a unlocked workspace.
	ErrWorkspaceNotLocked = errors.New("workspace already unlocked")

	// ErrWorkspaceLockedByRun is returned when trying to unlock a
	// workspace locked by a run
	ErrWorkspaceLockedByRun = errors.New("unable to unlock workspace locked by run")

	// ErrInvalidWorkspaceID is returned when the workspace ID is invalid.
	ErrInvalidWorkspaceID = errors.New("invalid value for workspace ID")

	// ErrInvalidWorkspaceValue is returned when workspace value is invalid.
	ErrInvalidWorkspaceValue = errors.New("invalid value for workspace")

	// ErrWorkspacesRequired is returned when the Workspaces are not present.
	ErrWorkspacesRequired = errors.New("workspaces is required")

	// ErrWorkspaceMinLimit is returned when the length of Workspaces is 0.
	ErrWorkspaceMinLimit = errors.New("must provide at least one workspace")

	// ErrMissingTagIdentifier is returned when tag resource identifiers are invalid
	ErrMissingTagIdentifier = errors.New("must specify at least one tag by ID or name")

	// Run/Apply errors

	// ErrInvalidRunID is returned when the run ID is invalid.
	ErrInvalidRunID = errors.New("invalid value for run ID")

	// Run Task errors

	// ErrInvalidRunTaskCategory is returned when a run task has a category other than "task"
	ErrInvalidRunTaskCategory = errors.New(`category must be "task"`)

	// ErrInvalidRunTaskID is returned when the run task ID is invalid
	ErrInvalidRunTaskID = errors.New("invalid value for run task ID")

	// ErrInvalidRunTaskURL is returned when the run task URL is invalid
	ErrInvalidRunTaskURL = errors.New("invalid url for run task URL")

	// Workspace Run Task errors

	//ErrInvalidWorkspaceRunTaskID is returned when the workspace run task ID is invalid
	ErrInvalidWorkspaceRunTaskID = errors.New("invalid value for workspace run task ID")

	//ErrInvalidWorkspaceRunTaskType is returned when Type is not "workspace-tasks"
	ErrInvalidWorkspaceRunTaskType = errors.New(`invalid value for type, please use "workspace-tasks"`)

	// Task Result errrors

	// ErrInvalidTaskResultID is returned when the task result ID is invalid
	ErrInvalidTaskResultID = errors.New("invalid value for task result ID")

	// Task Stage errors

	// ErrInvalidTaskStageID is returned when the task stage ID is invalid.
	ErrInvalidTaskStageID = errors.New("invalid value for task stage ID")

	// ErrInvalidApplyID is returned when the apply ID is invalid.
	ErrInvalidApplyID = errors.New("invalid value for apply ID")

	// Organzation errors

	// ErrInvalidOrg is returned when the organization option has an invalid value.
	ErrInvalidOrg = errors.New("invalid value for organization")

	// Agent errors

	// ErrInvalidAgentPoolID is returned when the agent pool ID is invalid.
	ErrInvalidAgentPoolID = errors.New("invalid value for agent pool ID")

	// ErrInvalidAgentTokenID is returned when the agent toek ID is invalid.
	ErrInvalidAgentTokenID = errors.New("invalid value for agent token ID")

	// Token errors

	// ErrAgentTokenDescription is returned when the description is blank.
	ErrAgentTokenDescription = errors.New("agent token description can't be blank")

	// Config errors

	// ErrInvalidConfigVersionID is returned when the configuration version ID is invalid.
	ErrInvalidConfigVersionID = errors.New("invalid value for configuration version ID")

	// Cost Esimation Errors

	// ErrInvalidCostEstimateID is returned when the cost estimate ID is invalid.
	ErrInvalidCostEstimateID = errors.New("invalid value for cost estimate ID")

	// User

	// ErrInvalidUservalue is invalid.
	ErrInvalidUserValue = errors.New("invalid value for user")

	// Settings

	// ErrInvalidSMTPAuth is returned when the smtp auth type is not valid.
	ErrInvalidSMTPAuth = errors.New("invalid smtp auth type")

	// Terraform Versions

	// ErrInvalidTerraformVersionID is returned when the ID for a terraform
	// version is invalid.
	ErrInvalidTerraformVersionID = errors.New("invalid value for terraform version ID")

	// ErrInvalidTerraformVersionType is returned when the type is not valid.
	ErrInvalidTerraformVersionType = errors.New("invalid type for terraform version. Please use 'terraform-version'")
)
