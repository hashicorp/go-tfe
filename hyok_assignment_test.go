// Copyright IBM Corp. 2018, 2026
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/hashicorp/jsonapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkspace_HYOKReencryptionStatusDecode(t *testing.T) {
	t.Parallel()

	decode := func(t *testing.T, attrs string) *Workspace {
		t.Helper()
		payload := `{"data":{"id":"ws-1","type":"workspaces","attributes":{"name":"ws"` + attrs + `}}}`
		w := &Workspace{}
		require.NoError(t, jsonapi.UnmarshalPayload(strings.NewReader(payload), w))
		return w
	}

	// An omitted attribute and an explicit null both decode as unset, which
	// callers treat as "no re-encryption outstanding".
	t.Run("omitted", func(t *testing.T) {
		w := decode(t, ``)
		assert.False(t, w.HYOKReencryptionStatus.IsSpecified())
		assert.Nil(t, w.HYOKReencryptionError)
	})

	t.Run("null", func(t *testing.T) {
		w := decode(t, `,"hyok-reencryption-status":null,"hyok-reencryption-error":null`)
		assert.False(t, w.HYOKReencryptionStatus.IsSpecified())
		assert.Nil(t, w.HYOKReencryptionError)
	})

	t.Run("pending", func(t *testing.T) {
		w := decode(t, `,"hyok-reencryption-status":"pending","hyok-reencryption-error":null`)
		status, err := w.HYOKReencryptionStatus.Get()
		require.NoError(t, err)
		assert.Equal(t, "pending", status)
		assert.Nil(t, w.HYOKReencryptionError)
	})

	t.Run("failed", func(t *testing.T) {
		w := decode(t, `,"hyok-reencryption-status":"failed","hyok-reencryption-error":"kms unavailable"`)
		status, err := w.HYOKReencryptionStatus.Get()
		require.NoError(t, err)
		assert.Equal(t, "failed", status)
		require.NotNil(t, w.HYOKReencryptionError)
		assert.Equal(t, "kms unavailable", *w.HYOKReencryptionError)
	})
}

func TestHYOKConfigurationRelationsDecode(t *testing.T) {
	t.Parallel()

	t.Run("project default", func(t *testing.T) {
		payload := `{"data":{"id":"prj-1","type":"projects","attributes":{"name":"p"},
			"relationships":{"default-hyok-configuration":{"data":{"id":"hyokc-1","type":"hyok-configurations"}}}}}`
		p := &Project{}
		require.NoError(t, jsonapi.UnmarshalPayload(strings.NewReader(payload), p))
		require.NotNil(t, p.DefaultHYOKConfiguration)
		assert.Equal(t, "hyokc-1", p.DefaultHYOKConfiguration.ID)
	})

	t.Run("project default null", func(t *testing.T) {
		payload := `{"data":{"id":"prj-1","type":"projects","attributes":{"name":"p"},
			"relationships":{"default-hyok-configuration":{"data":null}}}}`
		p := &Project{}
		require.NoError(t, jsonapi.UnmarshalPayload(strings.NewReader(payload), p))
		assert.Nil(t, p.DefaultHYOKConfiguration)
	})

	t.Run("workspace override", func(t *testing.T) {
		payload := `{"data":{"id":"ws-1","type":"workspaces","attributes":{"name":"ws"},
			"relationships":{"hyok-configuration":{"data":{"id":"hyokc-2","type":"hyok-configurations"}}}}}`
		w := &Workspace{}
		require.NoError(t, jsonapi.UnmarshalPayload(strings.NewReader(payload), w))
		require.NotNil(t, w.HYOKConfiguration)
		assert.Equal(t, "hyokc-2", w.HYOKConfiguration.ID)
	})
}

// encodedAttributes marshals options the same way the client does and returns
// the JSON:API attributes object.
func encodedAttributes(t *testing.T, options any) map[string]any {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, jsonapi.MarshalPayloadWithoutIncluded(&buf, options))

	var doc struct {
		Data struct {
			Attributes map[string]any `json:"attributes"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &doc))
	return doc.Data.Attributes
}

func TestHYOKConfigurationUpdateOptionsEncode(t *testing.T) {
	t.Parallel()

	check := func(t *testing.T, attr string, opts any, wantPresent bool, want any) {
		t.Helper()
		attrs := encodedAttributes(t, opts)
		v, ok := attrs[attr]
		assert.Equal(t, wantPresent, ok, "attribute %q presence", attr)
		if wantPresent {
			assert.Equal(t, want, v)
		}
	}

	const attrProject = "default-hyok-configuration-id"
	const attrWorkspace = "hyok-configuration-id"

	t.Run("project set", func(t *testing.T) {
		check(t, attrProject, &ProjectUpdateOptions{
			DefaultHYOKConfigurationID: jsonapi.NewNullableAttrWithValue("hyokc-1"),
		}, true, "hyokc-1")
	})
	t.Run("project explicit null", func(t *testing.T) {
		check(t, attrProject, &ProjectUpdateOptions{
			DefaultHYOKConfigurationID: jsonapi.NewNullNullableAttr[string](),
		}, true, nil)
	})
	t.Run("project unset", func(t *testing.T) {
		check(t, attrProject, &ProjectUpdateOptions{Name: String("p")}, false, nil)
	})

	t.Run("workspace set", func(t *testing.T) {
		check(t, attrWorkspace, &WorkspaceUpdateOptions{
			HYOKConfigurationID: jsonapi.NewNullableAttrWithValue("hyokc-2"),
		}, true, "hyokc-2")
	})
	t.Run("workspace explicit null", func(t *testing.T) {
		check(t, attrWorkspace, &WorkspaceUpdateOptions{
			HYOKConfigurationID: jsonapi.NewNullNullableAttr[string](),
		}, true, nil)
	})
	t.Run("workspace unset", func(t *testing.T) {
		check(t, attrWorkspace, &WorkspaceUpdateOptions{Name: String("ws")}, false, nil)
	})

	t.Run("create options omit unset ID", func(t *testing.T) {
		check(t, attrProject, &ProjectCreateOptions{Name: "p"}, false, nil)
		check(t, attrWorkspace, &WorkspaceCreateOptions{Name: String("ws")}, false, nil)
	})
	t.Run("create options send set ID", func(t *testing.T) {
		check(t, attrProject, &ProjectCreateOptions{Name: "p", DefaultHYOKConfigurationID: String("hyokc-1")}, true, "hyokc-1")
		check(t, attrWorkspace, &WorkspaceCreateOptions{Name: String("ws"), HYOKConfigurationID: String("hyokc-2")}, true, "hyokc-2")
	})
}
