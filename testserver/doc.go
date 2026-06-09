// Copyright IBM Corp. 2018, 2026
// SPDX-License-Identifier: MPL-2.0

// Package testserver provides a local in-memory API server for go-tfe clients.
//
// This package is a steel-thread starting point for deployment-free acceptance
// style tests and local API exploration. It intentionally implements only a
// subset of endpoints today:
//
//   - GET /api/v2/ping
//   - GET|POST /api/v2/organizations
//   - GET|PATCH|DELETE /api/v2/organizations/{organization}
//   - GET|POST /api/v2/organizations/{organization}/workspaces
//   - GET|PATCH|DELETE /api/v2/organizations/{organization}/workspaces/{workspace}
//   - GET|PATCH|DELETE /api/v2/workspaces/{workspace_id}
//
// The package also exposes helper methods to seed state and to construct a
// ready-to-use go-tfe client configuration via Server.ClientConfig.
package testserver
