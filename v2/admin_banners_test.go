// Copyright IBM Corp. 2018, 2026
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mustParseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t
}

func TestAdminBannersList(t *testing.T) {
	t.Run("returns banners", func(t *testing.T) {
		server, client := testServerWithClient(t, "/api/v2", map[string]http.HandlerFunc{
			"/api/v2/admin/banners": func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, http.MethodGet, r.Method)
				setDefaultServerHeaders(w)
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{
					"data": [
						{
							"id": "ab-123",
							"type": "admin-banners",
							"attributes": {
								"title": "Maintenance",
								"body": "<p>Scheduled maintenance tonight.</p>",
								"style": "warning",
								"audience": "all_users",
								"timezone": "UTC",
								"published-at": "2026-09-24T10:00:00Z",
								"scheduled-publish-at": null,
								"scheduled-expire-at": null
							}
						}
					]
				}`))
			},
		})
		defer server.Close()

		banners, err := client.Admin.Banners.List(context.Background())
		require.NoError(t, err)
		require.Len(t, banners, 1)
		assert.Equal(t, "ab-123", banners[0].ID)
		assert.Equal(t, "Maintenance", banners[0].Title)
		assert.Equal(t, "<p>Scheduled maintenance tonight.</p>", banners[0].Body)
		assert.Equal(t, "warning", banners[0].Style)
		assert.Equal(t, "all_users", banners[0].Audience)
		assert.Equal(t, "UTC", banners[0].Timezone)
		assert.NotNil(t, banners[0].PublishedAt)
		assert.Nil(t, banners[0].ScheduledPublishAt)
		assert.Nil(t, banners[0].ScheduledExpireAt)
	})

	t.Run("returns empty slice when no banners exist", func(t *testing.T) {
		server, client := testServerWithClient(t, "/api/v2", map[string]http.HandlerFunc{
			"/api/v2/admin/banners": func(w http.ResponseWriter, r *http.Request) {
				setDefaultServerHeaders(w)
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"data": []}`))
			},
		})
		defer server.Close()

		banners, err := client.Admin.Banners.List(context.Background())
		require.NoError(t, err)
		assert.Empty(t, banners)
	})
}

func TestAdminBannersCreate(t *testing.T) {
	t.Run("immediate banner", func(t *testing.T) {
		var captured map[string]any
		server, client := testServerWithClient(t, "/api/v2", map[string]http.HandlerFunc{
			"/api/v2/admin/banners": func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, http.MethodPost, r.Method)
				captured = attributesFromBody(t, captureBody(t, r))
				setDefaultServerHeaders(w)
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte(`{
					"data": {
						"id": "ab-123",
						"type": "admin-banners",
						"attributes": {
							"title": "Outage",
							"body": "<p>Service degraded.</p>",
							"style": "critical",
							"audience": "all_users",
							"timezone": "",
							"published-at": "2026-09-24T10:00:00Z",
							"scheduled-publish-at": null,
							"scheduled-expire-at": null
						}
					}
				}`))
			},
		})
		defer server.Close()

		banner, err := client.Admin.Banners.Create(context.Background(), AdminBannerCreateOptions{
			Title:    "Outage",
			Body:     "<p>Service degraded.</p>",
			Style:    "critical",
			Audience: "all_users",
		})
		require.NoError(t, err)
		require.NotNil(t, banner)

		assert.Equal(t, "Outage", captured["title"])
		assert.Equal(t, "<p>Service degraded.</p>", captured["body"])
		assert.Equal(t, "critical", captured["style"])
		assert.Equal(t, "all_users", captured["audience"])

		assert.Equal(t, "ab-123", banner.ID)
		assert.Equal(t, "Outage", banner.Title)
		assert.Equal(t, "critical", banner.Style)
		assert.NotNil(t, banner.PublishedAt)
		assert.Nil(t, banner.ScheduledPublishAt)
	})

	t.Run("scheduled banner", func(t *testing.T) {
		var captured map[string]any
		server, client := testServerWithClient(t, "/api/v2", map[string]http.HandlerFunc{
			"/api/v2/admin/banners": func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, http.MethodPost, r.Method)
				captured = attributesFromBody(t, captureBody(t, r))
				setDefaultServerHeaders(w)
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte(`{
					"data": {
						"id": "ab-456",
						"type": "admin-banners",
						"attributes": {
							"title": "Maintenance",
							"body": "<p>Scheduled downtime.</p>",
							"style": "info",
							"audience": "authenticated_only",
							"timezone": "America/New_York",
							"published-at": null,
							"scheduled-publish-at": "2026-10-01T02:00:00Z",
							"scheduled-expire-at": "2026-10-01T06:00:00Z"
						}
					}
				}`))
			},
		})
		defer server.Close()

		publishAt := mustParseTime("2026-10-01T02:00:00Z")
		expireAt := mustParseTime("2026-10-01T06:00:00Z")

		banner, err := client.Admin.Banners.Create(context.Background(), AdminBannerCreateOptions{
			Title:              "Maintenance",
			Body:               "<p>Scheduled downtime.</p>",
			Style:              "info",
			Audience:           "authenticated_only",
			ScheduledPublishAt: &publishAt,
			ScheduledExpireAt:  &expireAt,
			Timezone:           new("America/New_York"),
		})
		require.NoError(t, err)
		require.NotNil(t, banner)

		assert.NotEmpty(t, captured["scheduled-publish-at"])
		assert.NotEmpty(t, captured["scheduled-expire-at"])
		assert.Equal(t, "America/New_York", captured["timezone"])

		assert.Equal(t, "ab-456", banner.ID)
		assert.Nil(t, banner.PublishedAt)
		assert.NotNil(t, banner.ScheduledPublishAt)
		assert.NotNil(t, banner.ScheduledExpireAt)
	})

	t.Run("rejects invalid style", func(t *testing.T) {
		called := false
		server, client := testServerWithClient(t, "/api/v2", map[string]http.HandlerFunc{
			"/api/v2/admin/banners": func(w http.ResponseWriter, r *http.Request) {
				called = true
				setDefaultServerHeaders(w)
				w.WriteHeader(http.StatusCreated)
			},
		})
		defer server.Close()

		_, err := client.Admin.Banners.Create(context.Background(), AdminBannerCreateOptions{
			Title:    "Test",
			Body:     "Test body",
			Style:    "invalid",
			Audience: "all_users",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid")
		assert.False(t, called, "no request must be issued when validation fails")
	})

	t.Run("rejects invalid audience", func(t *testing.T) {
		called := false
		server, client := testServerWithClient(t, "/api/v2", map[string]http.HandlerFunc{
			"/api/v2/admin/banners": func(w http.ResponseWriter, r *http.Request) {
				called = true
				setDefaultServerHeaders(w)
				w.WriteHeader(http.StatusCreated)
			},
		})
		defer server.Close()

		_, err := client.Admin.Banners.Create(context.Background(), AdminBannerCreateOptions{
			Title:    "Test",
			Body:     "Test body",
			Style:    "info",
			Audience: "invalid",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid")
		assert.False(t, called, "no request must be issued when validation fails")
	})

	t.Run("propagates error when body exceeds character limit", func(t *testing.T) {
		server, client := testServerWithClient(t, "/api/v2", map[string]http.HandlerFunc{
			"/api/v2/admin/banners": func(w http.ResponseWriter, r *http.Request) {
				setDefaultServerHeaders(w)
				w.WriteHeader(http.StatusUnprocessableEntity)
				_, _ = w.Write([]byte(`{
					"errors": [
						{
							"status": "422",
							"title": "Invalid Attribute",
							"detail": "Body is too long (maximum is 500 characters)"
						}
					]
				}`))
			},
		})
		defer server.Close()

		_, err := client.Admin.Banners.Create(context.Background(), AdminBannerCreateOptions{
			Title:    "Test",
			Body:     string(make([]byte, 501)),
			Style:    "info",
			Audience: "all_users",
		})
		require.Error(t, err)
	})
}

func TestAdminBannersUpdate(t *testing.T) {
	t.Run("updates provided fields only", func(t *testing.T) {
		var captured map[string]any
		server, client := testServerWithClient(t, "/api/v2", map[string]http.HandlerFunc{
			"/api/v2/admin/banners/ab-123": func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, http.MethodPatch, r.Method)
				captured = attributesFromBody(t, captureBody(t, r))
				setDefaultServerHeaders(w)
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{
					"data": {
						"id": "ab-123",
						"type": "admin-banners",
						"attributes": {
							"title": "Updated title",
							"body": "<p>Service degraded.</p>",
							"style": "critical",
							"audience": "all_users",
							"timezone": "",
							"published-at": "2026-09-24T10:00:00Z",
							"scheduled-publish-at": null,
							"scheduled-expire-at": null
						}
					}
				}`))
			},
		})
		defer server.Close()

		banner, err := client.Admin.Banners.Update(context.Background(), "ab-123", AdminBannerUpdateOptions{
			Title: new("Updated title"),
		})
		require.NoError(t, err)
		require.NotNil(t, banner)

		assert.Equal(t, "Updated title", captured["title"])
		_, present := captured["body"]
		assert.False(t, present, "unprovided fields must not be sent on the wire")

		assert.Equal(t, "ab-123", banner.ID)
		assert.Equal(t, "Updated title", banner.Title)
	})

	t.Run("updates schedule fields", func(t *testing.T) {
		var captured map[string]any
		server, client := testServerWithClient(t, "/api/v2", map[string]http.HandlerFunc{
			"/api/v2/admin/banners/ab-123": func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, http.MethodPatch, r.Method)
				captured = attributesFromBody(t, captureBody(t, r))
				setDefaultServerHeaders(w)
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{
					"data": {
						"id": "ab-123",
						"type": "admin-banners",
						"attributes": {
							"title": "Maintenance",
							"body": "<p>Scheduled downtime.</p>",
							"style": "info",
							"audience": "authenticated_only",
							"timezone": "America/New_York",
							"published-at": null,
							"scheduled-publish-at": "2026-10-01T02:00:00Z",
							"scheduled-expire-at": "2026-10-01T06:00:00Z"
						}
					}
				}`))
			},
		})
		defer server.Close()

		publishAt := mustParseTime("2026-10-01T02:00:00Z")
		expireAt := mustParseTime("2026-10-01T06:00:00Z")

		banner, err := client.Admin.Banners.Update(context.Background(), "ab-123", AdminBannerUpdateOptions{
			ScheduledPublishAt: &publishAt,
			ScheduledExpireAt:  &expireAt,
			Timezone:           new("America/New_York"),
		})
		require.NoError(t, err)
		require.NotNil(t, banner)

		assert.NotEmpty(t, captured["scheduled-publish-at"])
		assert.NotEmpty(t, captured["scheduled-expire-at"])
		assert.Equal(t, "America/New_York", captured["timezone"])

		assert.Nil(t, banner.PublishedAt)
		assert.NotNil(t, banner.ScheduledPublishAt)
		assert.NotNil(t, banner.ScheduledExpireAt)
	})

	t.Run("rejects invalid style", func(t *testing.T) {
		called := false
		server, client := testServerWithClient(t, "/api/v2", map[string]http.HandlerFunc{
			"/api/v2/admin/banners/ab-123": func(w http.ResponseWriter, r *http.Request) {
				called = true
				setDefaultServerHeaders(w)
				w.WriteHeader(http.StatusOK)
			},
		})
		defer server.Close()

		style := "invalid"
		_, err := client.Admin.Banners.Update(context.Background(), "ab-123", AdminBannerUpdateOptions{
			Style: &style,
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid")
		assert.False(t, called, "no request must be issued when validation fails")
	})

	t.Run("rejects invalid audience", func(t *testing.T) {
		called := false
		server, client := testServerWithClient(t, "/api/v2", map[string]http.HandlerFunc{
			"/api/v2/admin/banners/ab-123": func(w http.ResponseWriter, r *http.Request) {
				called = true
				setDefaultServerHeaders(w)
				w.WriteHeader(http.StatusOK)
			},
		})
		defer server.Close()

		audience := "invalid"
		_, err := client.Admin.Banners.Update(context.Background(), "ab-123", AdminBannerUpdateOptions{
			Audience: &audience,
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid")
		assert.False(t, called, "no request must be issued when validation fails")
	})

	t.Run("propagates 404 when banner not found", func(t *testing.T) {
		server, client := testServerWithClient(t, "/api/v2", map[string]http.HandlerFunc{
			"/api/v2/admin/banners/ab-doesnotexist": func(w http.ResponseWriter, r *http.Request) {
				setDefaultServerHeaders(w)
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{
					"errors": [
						{
							"status": "404",
							"title": "Not Found"
						}
					]
				}`))
			},
		})
		defer server.Close()

		_, err := client.Admin.Banners.Update(context.Background(), "ab-doesnotexist", AdminBannerUpdateOptions{
			Title: new("Updated title"),
		})
		require.Error(t, err)
	})

	t.Run("clears schedule", func(t *testing.T) {
		var captured map[string]any
		server, client := testServerWithClient(t, "/api/v2", map[string]http.HandlerFunc{
			"/api/v2/admin/banners/ab-123": func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, http.MethodPatch, r.Method)
				captured = attributesFromBody(t, captureBody(t, r))
				setDefaultServerHeaders(w)
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{
					"data": {
						"id": "ab-123",
						"type": "admin-banners",
						"attributes": {
							"title": "Maintenance",
							"body": "<p>Scheduled downtime.</p>",
							"style": "info",
							"audience": "all_users",
							"timezone": null,
							"published-at": "2026-10-01T02:00:00Z",
							"scheduled-publish-at": null,
							"scheduled-expire-at": null
						}
					}
				}`))
			},
		})
		defer server.Close()

		banner, err := client.Admin.Banners.Update(context.Background(), "ab-123", AdminBannerUpdateOptions{
			ClearSchedule: true,
		})
		require.NoError(t, err)
		require.NotNil(t, banner)

		assert.Nil(t, captured["scheduled-publish-at"], "must send explicit null for scheduled-publish-at")
		assert.Nil(t, captured["scheduled-expire-at"], "must send explicit null for scheduled-expire-at")
		assert.Nil(t, captured["timezone"], "must send explicit null for timezone")
		assert.NotNil(t, banner.PublishedAt)
		assert.Nil(t, banner.ScheduledPublishAt)
		assert.Nil(t, banner.ScheduledExpireAt)
	})

	t.Run("rejects ClearSchedule with schedule fields", func(t *testing.T) {
		called := false
		server, client := testServerWithClient(t, "/api/v2", map[string]http.HandlerFunc{
			"/api/v2/admin/banners/ab-123": func(w http.ResponseWriter, r *http.Request) {
				called = true
				setDefaultServerHeaders(w)
				w.WriteHeader(http.StatusOK)
			},
		})
		defer server.Close()

		publishAt := mustParseTime("2026-10-01T02:00:00Z")
		_, err := client.Admin.Banners.Update(context.Background(), "ab-123", AdminBannerUpdateOptions{
			ClearSchedule:      true,
			ScheduledPublishAt: &publishAt,
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ClearSchedule")
		assert.False(t, called, "no request must be issued when validation fails")
	})
}

func TestAdminBannersDelete(t *testing.T) {
	t.Run("deletes banner", func(t *testing.T) {
		server, client := testServerWithClient(t, "/api/v2", map[string]http.HandlerFunc{
			"/api/v2/admin/banners/ab-123": func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, http.MethodDelete, r.Method)
				setDefaultServerHeaders(w)
				w.WriteHeader(http.StatusNoContent)
			},
		})
		defer server.Close()

		err := client.Admin.Banners.Delete(context.Background(), "ab-123")
		require.NoError(t, err)
	})

	t.Run("propagates 404 when banner not found", func(t *testing.T) {
		server, client := testServerWithClient(t, "/api/v2", map[string]http.HandlerFunc{
			"/api/v2/admin/banners/ab-doesnotexist": func(w http.ResponseWriter, r *http.Request) {
				setDefaultServerHeaders(w)
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{
					"errors": [
						{
							"status": "404",
							"title": "Not Found"
						}
					]
				}`))
			},
		})
		defer server.Close()

		err := client.Admin.Banners.Delete(context.Background(), "ab-doesnotexist")
		require.Error(t, err)
	})
}
