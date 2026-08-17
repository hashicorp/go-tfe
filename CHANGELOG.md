# Unreleased

* Enhancement: Workspace reads in the v2 client now include No-Code metadata: nullable
  `source-module-id` and `no-code-upgrade-available` attributes, plus a `no-code-module-version`
  relationship to a typed `no-code-module-versions` resource (`module-version`, `version-number`).
  These fields are omitted or null when the workspace is not No-Code. Prefer the relationship over
  `source-module-id` for a typed link to the current module version.

# v2.6.0

* The latest public endpoints are available

# v2.5.0

* The latest public endpoints are available

* Bug Fix: Set `Content-Type: application/vnd.api+json` on POST, PATCH, and DELETE requests without a body.

* Enhancement: NewClient configuration now supports `HTTPTransport` option, allowing you to customize many more aspects of every request round trip. 

# v2.4.0

The latest public endpoints are available

# v2.3.0

The latest public endpoints are available

# v2.2.0

The latest public endpoints are available

# v2.1.0

The latest public endpoints are available

# v2.0.0

The go-tfe v2 package has been added to this repository and contains substantial breaking changes
and improvements to the v1 package.

See [docs/UPGRADING.md](docs/UPGRADING.md) for more details.
