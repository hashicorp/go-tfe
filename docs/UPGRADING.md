# Upgrading from package v1 to v2

All interfaces in the v2 module are completely new but can be imported alongside the v1 module.

- The v2 module is no longer sensitive to ANY environment variables. All configuration must be
done using the `NewClient` function.

- The v2 module no longer makes an initial request in order to decode and store platform configuration.

- The v2 module does not include any up-to-date mocks. Old mocks may be found in the v1 `mocks` module.

To upgrade, get the module and import as `github.com/hashicorp/go-tfe/v2`

```bash
$ go get github.com/hashicorp/go-tfe/v2@VERSION
```

See the [Reference Documentation in README.md](../README.md) for full usage.
