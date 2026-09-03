package tfe

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNotificationConfigurationsListForStack(t *testing.T) {
	client, server := newNotificationConfigurationTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/stacks/stack-123/notification-configurations" {
			t.Fatalf("expected stack notification configuration path, got %q", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET request, got %s", r.Method)
		}

		writeJSONAPIResponse(t, w, `{"data":[{"type":"notification-configurations","id":"nc-123","attributes":{"destination-type":"email","enabled":true,"name":"stack notifications","triggers":["run:created"]},"relationships":{"subscribable":{"data":{"type":"stacks","id":"stack-123"}}}}]}`)
	})
	defer server.Close()

	list, err := client.NotificationConfigurations.List(context.Background(), "stack-123", &NotificationConfigurationListOptions{
		SubscribableChoice: &NotificationConfigurationSubscribableChoice{Stack: &Stack{}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list.Items) != 1 || list.Items[0].SubscribableChoice == nil || list.Items[0].SubscribableChoice.Stack == nil {
		t.Fatalf("expected decoded stack subscribable choice, got %#v", list.Items)
	}
	if list.Items[0].SubscribableChoice.Stack.ID != "stack-123" {
		t.Fatalf("expected decoded stack ID %q, got %q", "stack-123", list.Items[0].SubscribableChoice.Stack.ID)
	}
}

func TestNotificationConfigurationsCreateForStack(t *testing.T) {
	client, server := newNotificationConfigurationTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/stacks/stack-123/notification-configurations" {
			t.Fatalf("expected stack notification configuration path, got %q", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST request, got %s", r.Method)
		}
		assertStackSubscribableRelationship(t, r.Body)

		writeJSONAPIResponse(t, w, `{"data":{"type":"notification-configurations","id":"nc-123","attributes":{"destination-type":"email","enabled":true,"name":"stack notifications","triggers":["run:created"]},"relationships":{"subscribable":{"data":{"type":"stacks","id":"stack-123"}}}}}`)
	})
	defer server.Close()

	enabled := true
	notification, err := client.NotificationConfigurations.Create(context.Background(), "stack-123", NotificationConfigurationCreateOptions{
		DestinationType:    NotificationDestination(NotificationDestinationTypeEmail),
		Enabled:            &enabled,
		Name:               String("stack notifications"),
		SubscribableChoice: &NotificationConfigurationSubscribableChoice{Stack: &Stack{}},
		Triggers:           []NotificationTriggerType{NotificationTriggerCreated},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if notification.SubscribableChoice == nil || notification.SubscribableChoice.Stack == nil || notification.SubscribableChoice.Stack.ID != "stack-123" {
		t.Fatalf("expected decoded stack subscribable choice, got %#v", notification.SubscribableChoice)
	}
}

func TestNotificationConfigurationsStackValidation(t *testing.T) {
	choice := &NotificationConfigurationSubscribableChoice{Stack: &Stack{}}

	if _, err := notificationSubscribableURL("", choice); err != ErrInvalidStackID {
		t.Fatalf("expected ErrInvalidStackID from URL construction, got %v", err)
	}
	if err := validateSubscribableChoice(choice); err != ErrInvalidStackID {
		t.Fatalf("expected ErrInvalidStackID from relationship validation, got %v", err)
	}
}

func newNotificationConfigurationTestClient(t *testing.T, handler http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/ping" {
			w.WriteHeader(http.StatusOK)
			return
		}
		handler(w, r)
	}))
	client, err := NewClient(&Config{Address: server.URL, Token: "test-token"})
	if err != nil {
		server.Close()
		t.Fatalf("create client: %v", err)
	}
	return client, server
}

func assertStackSubscribableRelationship(t *testing.T, body io.Reader) {
	t.Helper()

	var payload struct {
		Data struct {
			Relationships struct {
				Subscribable struct {
					Data struct {
						ID   string `json:"id"`
						Type string `json:"type"`
					} `json:"data"`
				} `json:"subscribable"`
			} `json:"relationships"`
		} `json:"data"`
	}
	if err := json.NewDecoder(body).Decode(&payload); err != nil {
		t.Fatalf("decode JSON:API request: %v", err)
	}
	if payload.Data.Relationships.Subscribable.Data.Type != "stacks" || payload.Data.Relationships.Subscribable.Data.ID != "stack-123" {
		t.Fatalf("expected stack subscribable relationship, got %#v", payload.Data.Relationships.Subscribable.Data)
	}
}

func writeJSONAPIResponse(t *testing.T, w http.ResponseWriter, payload string) {
	t.Helper()
	w.Header().Set("Content-Type", ContentTypeJSONAPI)
	if _, err := w.Write([]byte(payload)); err != nil {
		t.Fatalf("write JSON:API response: %v", err)
	}
}
