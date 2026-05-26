// Copyright IBM Corp. 2018, 2026
// SPDX-License-Identifier: MPL-2.0

// These helpers represent JSON:API relationship linkage data for custom request payloads.
// They are shared by endpoints that need plain `json` payload structs instead of
// the standard `jsonapi` request models.

package tfe

type relationshipData struct {
	Data []relationshipItem `json:"data"`
}

type relationshipItem struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

func (p *Project) relationshipItem() relationshipItem {
	return relationshipItem{
		Type: "projects",
		ID:   p.ID,
	}
}

func (w *Workspace) relationshipItem() relationshipItem {
	return relationshipItem{
		Type: "workspaces",
		ID:   w.ID,
	}
}
