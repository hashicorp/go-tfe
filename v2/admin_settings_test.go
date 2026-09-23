// Copyright IBM Corp. 2018, 2026
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// captureBody reads and decodes a JSON:API request body into a generic map.
func captureBody(t *testing.T, r *http.Request) map[string]any {
	t.Helper()
	raw, err := io.ReadAll(r.Body)
	require.NoError(t, err)
	if len(raw) == 0 {
		return nil
	}
	var body map[string]any
	require.NoError(t, json.Unmarshal(raw, &body))
	return body
}

// attributesFromBody returns the attributes object from a captured body.
func attributesFromBody(t *testing.T, body map[string]any) map[string]any {
	t.Helper()
	require.NotNil(t, body)
	data, ok := body["data"].(map[string]any)
	require.True(t, ok, "expected data object in request body")
	attrs, ok := data["attributes"].(map[string]any)
	require.True(t, ok, "expected attributes object in request body")
	return attrs
}

func TestAdminCustomizationSettingsRead(t *testing.T) {
	server, client := testServerWithClient(t, "/api/v2", map[string]http.HandlerFunc{
		"/api/v2/admin/customization-settings": func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, http.MethodGet, r.Method)
			setDefaultServerHeaders(w)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"data": {
					"id": "customization",
					"type": "customization-settings",
					"attributes": {
						"support-email-address": "support@example.com",
						"support-url-address": "support.example.com",
						"login-help": "<p>login</p>",
						"footer": "<p>footer</p>",
						"error": "<p>error</p>",
						"new-user": "<p>welcome</p>",
						"internal-support-url": "https://internal.example.com/support",
						"internal-documentation-url": "https://internal.example.com/docs",
						"internal-tutorials-url": "https://internal.example.com/tutorials",
						"avatar-sources": [
							{ "source": "gravatar", "enabled": true }
						]
					}
				}
			}`))
		},
	})
	defer server.Close()

	setting, err := client.Admin.Settings.Customization.Read(context.Background())
	require.NoError(t, err)
	require.NotNil(t, setting)

	assert.Equal(t, "customization", setting.ID)
	assert.Equal(t, "support@example.com", setting.SupportEmail)
	assert.Equal(t, "support.example.com", setting.SupportURL)
	assert.Equal(t, "<p>login</p>", setting.LoginHelp)
	assert.Equal(t, "<p>footer</p>", setting.Footer)
	assert.Equal(t, "<p>error</p>", setting.Error)
	assert.Equal(t, "<p>welcome</p>", setting.NewUser)

	// Internal override fields.
	assert.Equal(t, "https://internal.example.com/support", setting.InternalSupportURL)
	assert.Equal(t, "https://internal.example.com/docs", setting.InternalDocumentationURL)
	assert.Equal(t, "https://internal.example.com/tutorials", setting.InternalTutorialsURL)
	require.Len(t, setting.AvatarSources, 1)
	assert.Equal(t, "gravatar", setting.AvatarSources[0].Source)
	assert.True(t, setting.AvatarSources[0].Enabled)
}

func TestAdminCustomizationSettingsUpdate(t *testing.T) {
	t.Run("sets all new fields", func(t *testing.T) {
		var captured map[string]any
		server, client := testServerWithClient(t, "/api/v2", map[string]http.HandlerFunc{
			"/api/v2/admin/customization-settings": func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, http.MethodPatch, r.Method)
				captured = attributesFromBody(t, captureBody(t, r))
				setDefaultServerHeaders(w)
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{
					"data": {
						"id": "customization",
						"type": "customization-settings",
						"attributes": {
							"support-url-address": "support.example.com",
							"internal-support-url": "https://internal.example.com/support",
							"internal-documentation-url": "https://internal.example.com/docs",
							"internal-tutorials-url": "https://internal.example.com/tutorials",
							"avatar-sources": [
								{ "source": "gravatar", "enabled": true }
							]
						}
					}
				}`))
			},
		})
		defer server.Close()

		setting, err := client.Admin.Settings.Customization.Update(context.Background(), AdminCustomizationSettingsUpdateOptions{
			SupportURL:               new("support.example.com"),
			InternalSupportURL:       new("https://internal.example.com/support"),
			InternalDocumentationURL: new("https://internal.example.com/docs"),
			InternalTutorialsURL:     new("https://internal.example.com/tutorials"),
			AvatarSources: []AdminAvatarSource{
				{Source: "gravatar", Enabled: true},
			},
		})
		require.NoError(t, err)
		require.NotNil(t, setting)

		// Verify the wire payload carried the new fields.
		assert.Equal(t, "support.example.com", captured["support-url-address"])
		assert.Equal(t, "https://internal.example.com/support", captured["internal-support-url"])
		assert.Equal(t, "https://internal.example.com/docs", captured["internal-documentation-url"])
		assert.Equal(t, "https://internal.example.com/tutorials", captured["internal-tutorials-url"])
		sources, ok := captured["avatar-sources"].([]any)
		require.True(t, ok)
		require.Len(t, sources, 1)
		entry := sources[0].(map[string]any)
		assert.Equal(t, "gravatar", entry["source"])
		assert.Equal(t, true, entry["enabled"])

		// Verify the response round-tripped back into the struct.
		assert.Equal(t, "https://internal.example.com/support", setting.InternalSupportURL)
		require.Len(t, setting.AvatarSources, 1)
		assert.Equal(t, "gravatar", setting.AvatarSources[0].Source)
	})

	t.Run("clears an optional URL field to restore the default", func(t *testing.T) {
		var captured map[string]any
		server, client := testServerWithClient(t, "/api/v2", map[string]http.HandlerFunc{
			"/api/v2/admin/customization-settings": func(w http.ResponseWriter, r *http.Request) {
				captured = attributesFromBody(t, captureBody(t, r))
				setDefaultServerHeaders(w)
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{
					"data": {
						"id": "customization",
						"type": "customization-settings",
						"attributes": {
							"internal-support-url": ""
						}
					}
				}`))
			},
		})
		defer server.Close()

		setting, err := client.Admin.Settings.Customization.Update(context.Background(), AdminCustomizationSettingsUpdateOptions{
			InternalSupportURL: new(""),
		})
		require.NoError(t, err)
		require.NotNil(t, setting)

		// The cleared field must be present in the payload as an empty string
		// so the server restores the default link.
		val, present := captured["internal-support-url"]
		require.True(t, present, "cleared field must be sent on the wire")
		assert.Equal(t, "", val)
		assert.Equal(t, "", setting.InternalSupportURL)
	})

	t.Run("disables all avatar sources", func(t *testing.T) {
		var captured map[string]any
		server, client := testServerWithClient(t, "/api/v2", map[string]http.HandlerFunc{
			"/api/v2/admin/customization-settings": func(w http.ResponseWriter, r *http.Request) {
				captured = attributesFromBody(t, captureBody(t, r))
				setDefaultServerHeaders(w)
				w.WriteHeader(http.StatusOK)
				// Atlas always returns the full catalog in catalog order, with
				// each source flagged; disabling all yields enabled:false rather
				// than an empty array.
				_, _ = w.Write([]byte(`{
					"data": {
						"id": "customization",
						"type": "customization-settings",
						"attributes": {
							"avatar-sources": [
								{ "source": "gravatar", "enabled": false }
							]
						}
					}
				}`))
			},
		})
		defer server.Close()

		setting, err := client.Admin.Settings.Customization.Update(context.Background(), AdminCustomizationSettingsUpdateOptions{
			AvatarSources: []AdminAvatarSource{},
		})
		require.NoError(t, err)
		require.NotNil(t, setting)

		// A non-nil empty slice is sent as an empty array, which Atlas treats
		// as disabling every configurable source.
		sources, ok := captured["avatar-sources"].([]any)
		require.True(t, ok, "empty catalog must be sent as an empty array")
		assert.Empty(t, sources)

		// The response echoes the full catalog with every source disabled.
		require.Len(t, setting.AvatarSources, 1)
		assert.Equal(t, "gravatar", setting.AvatarSources[0].Source)
		assert.False(t, setting.AvatarSources[0].Enabled)
	})

	t.Run("omits fields that are not set", func(t *testing.T) {
		var captured map[string]any
		server, client := testServerWithClient(t, "/api/v2", map[string]http.HandlerFunc{
			"/api/v2/admin/customization-settings": func(w http.ResponseWriter, r *http.Request) {
				captured = attributesFromBody(t, captureBody(t, r))
				setDefaultServerHeaders(w)
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{
					"data": {
						"id": "customization",
						"type": "customization-settings",
						"attributes": { "new-user": "<p>welcome</p>" }
					}
				}`))
			},
		})
		defer server.Close()

		_, err := client.Admin.Settings.Customization.Update(context.Background(), AdminCustomizationSettingsUpdateOptions{
			NewUser: new("<p>welcome</p>"),
		})
		require.NoError(t, err)

		assert.Equal(t, "<p>welcome</p>", captured["new-user"])
		// Untouched fields must not appear in the payload.
		_, present := captured["internal-support-url"]
		assert.False(t, present)
		_, present = captured["avatar-sources"]
		assert.False(t, present)
	})

	t.Run("rejects avatar sources with an unknown source", func(t *testing.T) {
		called := false
		server, client := testServerWithClient(t, "/api/v2", map[string]http.HandlerFunc{
			"/api/v2/admin/customization-settings": func(w http.ResponseWriter, r *http.Request) {
				called = true
				setDefaultServerHeaders(w)
				w.WriteHeader(http.StatusOK)
			},
		})
		defer server.Close()

		_, err := client.Admin.Settings.Customization.Update(context.Background(), AdminCustomizationSettingsUpdateOptions{
			AvatarSources: []AdminAvatarSource{
				{Source: "gravatar", Enabled: true},
				// A mixed valid/invalid catalog must be rejected before the
				// PATCH so Atlas never receives a silently-truncated catalog.
				{Source: "saml", Enabled: true},
			},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "saml")
		assert.False(t, called, "no request must be issued when validation fails")
	})
}

func TestAdminGeneralSettingsRead(t *testing.T) {
	server, client := testServerWithClient(t, "/api/v2", map[string]http.HandlerFunc{
		"/api/v2/admin/general-settings": func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, http.MethodGet, r.Method)
			setDefaultServerHeaders(w)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"data": {
					"id": "general",
					"type": "general-settings",
					"attributes": { "enable-public-registration": true }
				}
			}`))
		},
	})
	defer server.Close()

	setting, err := client.Admin.Settings.General.Read(context.Background())
	require.NoError(t, err)
	require.NotNil(t, setting)
	assert.Equal(t, "general", setting.ID)
	assert.True(t, setting.EnablePublicRegistration)
}

func TestAdminGeneralSettingsUpdate(t *testing.T) {
	tests := []struct {
		name  string
		value bool
	}{
		{name: "enable public registration", value: true},
		{name: "disable public registration", value: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var captured map[string]any
			server, client := testServerWithClient(t, "/api/v2", map[string]http.HandlerFunc{
				"/api/v2/admin/general-settings": func(w http.ResponseWriter, r *http.Request) {
					require.Equal(t, http.MethodPatch, r.Method)
					captured = attributesFromBody(t, captureBody(t, r))
					setDefaultServerHeaders(w)
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`{
						"data": {
							"id": "general",
							"type": "general-settings",
							"attributes": { "enable-public-registration": ` + strconv.FormatBool(tc.value) + ` }
						}
					}`))
				},
			})
			defer server.Close()

			setting, err := client.Admin.Settings.General.Update(context.Background(), AdminGeneralSettingsUpdateOptions{
				EnablePublicRegistration: new(tc.value),
			})
			require.NoError(t, err)
			require.NotNil(t, setting)

			assert.Equal(t, tc.value, captured["enable-public-registration"])
			assert.Equal(t, tc.value, setting.EnablePublicRegistration)
		})
	}
}
