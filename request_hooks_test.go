package tfe

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestContextWithResponseHeaderHook(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("x-thingy", "boop")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	cfg := &Config{
		Address:  server.URL,
		BasePath: "/anything",
		Token:    "placeholder",
	}
	client, err := NewClient(cfg)
	if err != nil {
		t.Fatal(err)
	}

	called := false
	var gotStatus int
	var gotHeader http.Header
	ctx := ContextWithResponseHeaderHook(context.Background(), func(status int, header http.Header) {
		called = true
		gotStatus = status
		gotHeader = header
	})

	req, err := client.NewRequest("GET", "boop", nil)
	if err != nil {
		t.Fatal(err)
	}

	err = req.Do(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}

	if !called {
		t.Fatal("hook was not called")
	}
	if got, want := gotStatus, http.StatusNoContent; got != want {
		t.Fatalf("wrong response status: got %d, want %d", got, want)
	}
	if got, want := gotHeader.Get("x-thingy"), "boop"; got != want {
		t.Fatalf("wrong value for x-thingy field: got %q, want %q", got, want)
	}
}
