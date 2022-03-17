# v1.0.0

## Breaking Changes
* Renamed methods named Generate to Create for `AgentTokens`, `OrganizationTokens`, `TeamTokens`, `UserTokens` [#327](https://github.com/hashicorp/go-tfe/pull/327)
* Methods that express an action on a relationship have been prefixed with a verb, e.g `Current()` is now `ReadCurrent()` [#327](https://github.com/hashicorp/go-tfe/pull/327)
* All list option structs are now pointers [#309](https://github.com/hashicorp/go-tfe/pull/309)
* All errors have been refactored into constants in `errors.go` [#310](https://github.com/hashicorp/go-tfe/pull/310)
* The `ID` field in Create/Update option structs has been renamed to `Type` in accordance with the JSON:API spec [#190](https://github.com/hashicorp/go-tfe/pull/190), [#323](https://github.com/hashicorp/go-tfe/pull/323), [#332](https://github.com/hashicorp/go-tfe/pull/332)
* Nested URL params (consisting of an organization, module and provider name) used to identify a `RegistryModule` have been refactored into a struct `RegistryModuleID`. [#337](https://github.com/hashicorp/go-tfe/pull/337)


## Enhancements
* Added missing include fields for `AdminRuns`, `AgentPools`, `ConfigurationVersions`, `OAuthClients`, `Organizations`, `PolicyChecks`, `PolicySets`, `Policies` and `RunTriggers` [#334](https://github.com/hashicorp/go-tfe/pull/334)
* Cleanup documentation and improve consistency [#331](https://github.com/hashicorp/go-tfe/pull/331)
* Add more linters to our CI pipeline [#326](https://github.com/hashicorp/go-tfe/pull/326)
* Resolve `TFE_HOSTNAME` as fallback for `TFE_ADDRESS` [#340](https://github.com/hashicorp/go-tfe/pull/326)
* Adds a`fetching` status to `RunStatus` and adds the `Archive` method to the ConfigurationVersions interface [#338](https://github.com/hashicorp/go-tfe/pull/338)
* Added a `Download` method to the `ConfigurationVersions` interface [#358](https://github.com/hashicorp/go-tfe/pull/358)

## Bug Fixes
* Fixed invalid memory address error when `AdminSMTPSettingsUpdateOptions.Auth` field is empty and accessed [#335](https://github.com/hashicorp/go-tfe/pull/335) 

