# Unreleased

## Enhancements
* Adds `Identifier`, `OAuthTokenID`, and `GHAInstallationID` fields to `RegistryModuleVCSRepoUpdateOptions` so callers can update a registry module's VCS repository identifier and connection (OAuth token or GitHub App installation) via the `Update` method; setting both `OAuthTokenID` and `GHAInstallationID` in the same call returns `ErrMutuallyExclusiveOAuthTokenAndGHAInstallation` [#1376](https://github.com/hashicorp/go-tfe/pull/1376)

# v2.0.0

The go-tfe v2 package has been added to this repository and contains substantial breaking changes
and improvements to the v1 package.

See [docs/UPGRADING.md](docs/UPGRADING.md) for more details.