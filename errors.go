package tfe

import (
	"errors"
)

// Generic errors applicable to all resources.
var (
	// ErrWorkspaceLocked is returned when trying to lock a
	// locked workspace.
	ErrWorkspaceLocked = errors.New("workspace already locked")

	// ErrWorkspaceNotLocked is returned when trying to unlock
	// a unlocked workspace.
	ErrWorkspaceNotLocked = errors.New("workspace already unlocked")

	// ErrUnauthorized is returned when a receiving a 401.
	ErrUnauthorized = errors.New("unauthorized")

	// ErrResourceNotFound is returned when a receiving a 404.
	ErrResourceNotFound = errors.New("resource not found")

	// ErrRequiredName is returned when a name option is not present.
	ErrRequiredName = errors.New("name is required")

	// ErrInvalidName is returned when the name option has invalid value.
	ErrInvalidName = errors.New("invalid value for name")

	// ErrMissingDirectory is returned when the path does not have an existing directory.
	ErrMissingDirectory = errors.New("path needs to be an existing directory")

	// ErrInvalidOrg is returned when the organization option has an invalid value.
	ErrInvalidOrg = errors.New("invalid value for organization")

	// ErrInvalidAgentPoolID is returned when the agent pool ID is invalid.
	ErrInvalidAgentPoolID = errors.New("invalid value for agent pool ID")

	// ErrInvalidWorkspaceID is returned when the workspace ID is invalid.
	ErrInvalidWorkspaceID = errors.New("invalid value for workspace ID")

	// ErrInvalidRunID is returned when the run ID is invalid.
	ErrInvalidRunID = errors.New("invalid value for run ID")

	// ErrInvalidApplyID is returned when the apply ID is invalid.
	ErrInvalidApplyID = errors.New("invalid value for apply ID")

	// ErrInvalidAgentTokenID is returned when the agent toek ID is invalid.
	ErrInvalidAgentTokenID = errors.New("invalid value for agent token ID")

	// ErrAgentTokenDescription is returned when the description is blank.
	ErrAgentTokenDescription = errors.New("agent token description can't be blank")

	// ErrInvalidConfigVersionID is returned when the configuration version ID is invalid.
	ErrInvalidConfigVersionID = errors.New("invalid value for configuration version ID")

	// ErrInvalidCostEstimateID is returned when the cost estimate ID is invalid.
	ErrInvalidCostEstimateID = errors.New("invalid value for cost estimate ID")
)
