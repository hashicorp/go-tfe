package tfe

// VCSRepo contains the configuration of a VCS integration.
type VCSRepo struct {
	// The ID of the VCS integration to use for cloning this workspace's
	// configuration.
	OauthTokenID string `json:"oauth-token-id,omitempty"`

	// The identifier of the VCS repository. The format of this field is
	// typically "<user or org>/<repo name>", depending on the VCS backend.
	Identifier string `json:"identifier,omitempty"`

	// Non-default branch to clone. Defaults to the default branch configured
	// at the VCS provider.
	Branch string `json:"branch,omitempty"`

	// Determines if submodules should be initialized and cloned on the
	// Terraform configuration repository when TFE clones the VCS repo.
	IncludeSubmodules bool `json:"ingress-submodules"`
}
