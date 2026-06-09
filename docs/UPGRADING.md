# Upgrading from package v1 to v2

# Breaking Changes

All interfaces are completely new so it is necessary to replace all go-tfe v1 callers when
upgrading to the v2 package.

The v2 package is no longer sensitive to ANY environment variables. All configuration must be
done using the `NewClient` function.

The v2 package no longer makes an initial request in order to decode and store platform configuration.

The v2 package does not include any up-to-date mocks. Old mocks may be found in the v1 `mocks` package.

To upgrade, get the module

```bash
$ go get github.com/hashicorp/go-tfe/v2@v2.0.0-beta1
```

...and import as `github.com/hashicorp/go-tfe/v2`

Ree the [Reference Documentation in README.md](../README.md) for full usage.