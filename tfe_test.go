package tfe

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	//"github.com/svanharmelen/jsonapi"
	"github.com/google/jsonapi"
	"github.com/hashicorp/go-retryablehttp"
	"golang.org/x/time/rate"
)

func TestClient_newClient(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		w.Header().Set("X-RateLimit-Limit", "30")
		w.Header().Set("TFP-API-Version", "34.21.9")
		w.WriteHeader(204) // We query the configured ping URL which should return a 204.
	}))
	defer ts.Close()

	cfg := &Config{
		HTTPClient: ts.Client(),
	}

	t.Run("uses env vars if values are missing", func(t *testing.T) {
		defer setupEnvVars("abcd1234", ts.URL)()

		client, err := NewClient(cfg)
		if err != nil {
			t.Fatal(err)
		}
		if client.token != "abcd1234" {
			t.Fatalf("unexpected token: %q", client.token)
		}
		if client.baseURL.String() != ts.URL+DefaultBasePath {
			t.Fatalf("unexpected address: %q", client.baseURL.String())
		}
	})

	t.Run("fails if token is empty", func(t *testing.T) {
		defer setupEnvVars("", "")()

		_, err := NewClient(cfg)
		if err == nil || err.Error() != "missing API token" {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("makes a new client with good settings", func(t *testing.T) {
		config := &Config{
			Address:    ts.URL,
			Token:      "abcd1234",
			HTTPClient: ts.Client(),
		}

		client, err := NewClient(config)
		if err != nil {
			t.Fatal(err)
		}

		if config.Address+DefaultBasePath != client.baseURL.String() {
			t.Fatalf("unexpected client address %q", client.baseURL.String())
		}
		if config.Token != client.token {
			t.Fatalf("unexpected client token %q", client.token)
		}
		if ts.Client() != client.http.HTTPClient {
			t.Fatal("unexpected HTTP client value")
		}
		if want := "34.21.9"; client.RemoteAPIVersion() != want {
			t.Errorf("unexpected remote API version %q; want %q", client.RemoteAPIVersion(), want)
		}

		client.SetFakeRemoteAPIVersion("1.0")

		if want := "1.0"; client.RemoteAPIVersion() != want {
			t.Errorf("unexpected remote API version %q; want %q", client.RemoteAPIVersion(), want)
		}
	})
}

func TestClient_defaultConfig(t *testing.T) {
	t.Run("with no environment variables", func(t *testing.T) {
		defer setupEnvVars("", "")()

		config := DefaultConfig()

		if config.Address != DefaultAddress {
			t.Fatalf("expected %q, got %q", DefaultAddress, config.Address)
		}
		if config.Token != "" {
			t.Fatalf("expected empty token, got %q", config.Token)
		}
		if config.HTTPClient == nil {
			t.Fatalf("expected default http client, got %v", config.HTTPClient)
		}
	})

	t.Run("with environment variables", func(t *testing.T) {
		defer setupEnvVars("abcd1234", "https://mytfe.local")()
	})
}

func TestClient_headers(t *testing.T) {
	testedCalls := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testedCalls++

		if testedCalls == 1 {
			w.Header().Set("Content-Type", "application/vnd.api+json")
			w.Header().Set("X-RateLimit-Limit", "30")
			w.WriteHeader(204) // We query the configured ping URL which should return a 204.
			return
		}

		if r.Header.Get("Accept") != "application/vnd.api+json" {
			t.Fatalf("unexpected accept header: %q", r.Header.Get("Accept"))
		}
		if r.Header.Get("Authorization") != "Bearer dummy-token" {
			t.Fatalf("unexpected authorization header: %q", r.Header.Get("Authorization"))
		}
		if r.Header.Get("My-Custom-Header") != "foobar" {
			t.Fatalf("unexpected custom header: %q", r.Header.Get("My-Custom-Header"))
		}
		if r.Header.Get("Terraform-Version") != "0.11.9" {
			t.Fatalf("unexpected Terraform version header: %q", r.Header.Get("Terraform-Version"))
		}
		if r.Header.Get("User-Agent") != "go-tfe" {
			t.Fatalf("unexpected user agent header: %q", r.Header.Get("User-Agent"))
		}
	}))
	defer ts.Close()

	cfg := &Config{
		Address:    ts.URL,
		Token:      "dummy-token",
		Headers:    make(http.Header),
		HTTPClient: ts.Client(),
	}

	// Set some custom header.
	cfg.Headers.Set("My-Custom-Header", "foobar")
	cfg.Headers.Set("Terraform-Version", "0.11.9")

	// This one should be overridden!
	cfg.Headers.Set("Authorization", "bad-token")

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	// Make a few calls so we can check they all send the expected headers.
	_, _ = client.Organizations.List(ctx, OrganizationListOptions{})
	_, _ = client.Plans.Logs(ctx, "plan-123456789")
	_ = client.Runs.Apply(ctx, "run-123456789", RunApplyOptions{})
	_, _ = client.Workspaces.Lock(ctx, "ws-123456789", WorkspaceLockOptions{})
	_, _ = client.Workspaces.Read(ctx, "organization", "workspace")

	if testedCalls != 6 {
		t.Fatalf("expected 6 tested calls, got: %d", testedCalls)
	}
}

func TestClient_userAgent(t *testing.T) {
	testedCalls := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testedCalls++

		if testedCalls == 1 {
			w.Header().Set("Content-Type", "application/vnd.api+json")
			w.Header().Set("X-RateLimit-Limit", "30")
			w.WriteHeader(204) // We query the configured ping URL which should return a 204.
			return
		}

		if r.Header.Get("User-Agent") != "hashicorp" {
			t.Fatalf("unexpected user agent header: %q", r.Header.Get("User-Agent"))
		}
	}))
	defer ts.Close()

	cfg := &Config{
		Address:    ts.URL,
		Token:      "dummy-token",
		Headers:    make(http.Header),
		HTTPClient: ts.Client(),
	}

	// Set a custom user agent.
	cfg.Headers.Set("User-Agent", "hashicorp")

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	// Make a few calls so we can check they all send the expected headers.
	_, _ = client.Organizations.List(ctx, OrganizationListOptions{})
	_, _ = client.Plans.Logs(ctx, "plan-123456789")
	_ = client.Runs.Apply(ctx, "run-123456789", RunApplyOptions{})
	_, _ = client.Workspaces.Lock(ctx, "ws-123456789", WorkspaceLockOptions{})
	_, _ = client.Workspaces.Read(ctx, "organization", "workspace")

	if testedCalls != 6 {
		t.Fatalf("expected 6 tested calls, got: %d", testedCalls)
	}
}

type JSONAPIBody struct {
	StrAttr string `jsonapi:"attr,str_attr"`
}

type JSONPlainBody struct {
	StrAttr string `jsonapi:"attr,str_attr"`
}

type InvalidBody struct {
	Attr1 string `jsonapi:"attr,attr1"`
	Attr2 string `jsonapi:"attr,attr2"`
}

func TestClient_requestBodySerialization(t *testing.T) {
	t.Run("jsonapi request", func(t *testing.T) {
		body := JSONAPIBody{StrAttr: "foo"}
		_, requestBody, err := createRequest(&body)
		if err != nil {
			t.Fatal(err)
		}

		unmarshalledRequestBody := JSONAPIBody{}
		err = jsonapi.UnmarshalPayload(bytes.NewReader(requestBody), &unmarshalledRequestBody)
		if err != nil {
			t.Fatal(err)
		}
		if unmarshalledRequestBody.StrAttr != body.StrAttr {
			t.Fatal("Request serialized incorrectly")
		}
	})

	t.Run("jsonapi slice of pointers request", func(t *testing.T) {
		var body []*JSONAPIBody
		body = append(body, &JSONAPIBody{StrAttr: "foo"})
		_, requestBody, err := createRequest(body)
		if err != nil {
			t.Fatal(err)
		}

		// The jsonapi library doesn't support unmarshalling bulk objects,
		// so for this test we deserialize to the jsonapi intermediate
		// format and validate it manually
		parsedResponse := new(jsonapi.ManyPayload)
		err = json.Unmarshal(requestBody, &parsedResponse)
		if err != nil {
			t.Fatal(err)
		}

		if len(parsedResponse.Data) != 1 || parsedResponse.Data[0].Attributes["str_attr"] != "foo" {
			t.Fatal("Request serialized incorrectly")
		}
	})

	t.Run("plain json request", func(t *testing.T) {
		body := JSONPlainBody{StrAttr: "foo"}
		_, requestBody, err := createRequest(&body)
		if err != nil {
			t.Fatal(err)
		}

		unmarshalledRequestBody := JSONPlainBody{}
		err = json.Unmarshal(requestBody, &unmarshalledRequestBody)
		if err != nil {
			t.Fatal(err)
		}
		if unmarshalledRequestBody.StrAttr != body.StrAttr {
			t.Fatal("Request serialized incorrectly")
		}
	})

	t.Run("plain json slice of pointers request", func(t *testing.T) {
		var body []*JSONPlainBody
		body = append(body, &JSONPlainBody{StrAttr: "foo"})
		_, requestBody, err := createRequest(body)
		if err != nil {
			t.Fatal(err)
		}

		var unmarshalledRequestBody []*JSONPlainBody
		err = json.Unmarshal(requestBody, &unmarshalledRequestBody)
		if err != nil {
			t.Fatal(err)
		}
		if len(unmarshalledRequestBody) != 1 || unmarshalledRequestBody[0].StrAttr != body[0].StrAttr {
			t.Fatal("Request serialized incorrectly")
		}
	})

	t.Run("nil request", func(t *testing.T) {
		_, requestBody, err := createRequest(nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(requestBody) != 0 {
			t.Fatal("nil request serialized incorrectly")
		}
	})

	t.Run("invalid struct request", func(t *testing.T) {
		body := InvalidBody{}
		_, _, err := createRequest(&body)
		if err == nil || err.Error() != "go-tfe bug: struct can't use both json and jsonapi attributes" {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("non-pointer request", func(t *testing.T) {
		body := InvalidBody{}
		_, _, err := createRequest(body)
		if err == nil || err.Error() != "go-tfe bug: DELETE/PATCH/POST body must be nil, ptr, or ptr slice" {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("slice of non-pointer request", func(t *testing.T) {
		body := []InvalidBody{{}}
		_, _, err := createRequest(body)
		if err == nil || err.Error() != "go-tfe bug: DELETE/PATCH/POST body must be nil, ptr, or ptr slice" {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("map request", func(t *testing.T) {
		body := make(map[string]string)
		_, _, err := createRequest(body)
		if err == nil || err.Error() != "go-tfe bug: DELETE/PATCH/POST body must be nil, ptr, or ptr slice" {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("string request", func(t *testing.T) {
		body := "foo"
		_, _, err := createRequest(body)
		if err == nil || err.Error() != "go-tfe bug: DELETE/PATCH/POST body must be nil, ptr, or ptr slice" {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func createRequest(v interface{}) (*retryablehttp.Request, []byte, error) {
	config := DefaultConfig()
	config.Token = "dummy"
	client, err := NewClient(config)
	if err != nil {
		return nil, nil, err
	}

	request, err := client.newRequest("POST", "/bar", v)
	if err != nil {
		return nil, nil, err
	}
	body, err := request.BodyBytes()
	if err != nil {
		return request, nil, err
	}
	return request, body, nil
}

func TestClient_configureLimiter(t *testing.T) {
	rateLimit := ""
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		w.Header().Set("X-RateLimit-Limit", rateLimit)
		w.WriteHeader(204) // We query the configured ping URL which should return a 204.
	}))
	defer ts.Close()

	cfg := &Config{
		Address:    ts.URL,
		Token:      "dummy-token",
		HTTPClient: ts.Client(),
	}

	cases := map[string]struct {
		rate  string
		limit rate.Limit
		burst int
	}{
		"no-value": {
			rate:  "",
			limit: rate.Inf,
			burst: 0,
		},
		"limit-0": {
			rate:  "0",
			limit: rate.Inf,
			burst: 0,
		},
		"limit-30": {
			rate:  "30",
			limit: rate.Limit(19.8),
			burst: 9,
		},
		"limit-100": {
			rate:  "100",
			limit: rate.Limit(66),
			burst: 33,
		},
	}

	for name, tc := range cases {
		// First set the test rate limit.
		rateLimit = tc.rate

		client, err := NewClient(cfg)
		if err != nil {
			t.Fatal(err)
		}

		if client.limiter.Limit() != tc.limit {
			t.Fatalf("test %s expected limit %f, got: %f", name, tc.limit, client.limiter.Limit())
		}

		if client.limiter.Burst() != tc.burst {
			t.Fatalf("test %s expected burst %d, got: %d", name, tc.burst, client.limiter.Burst())
		}
	}
}

func TestClient_retryHTTPCheck(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		w.Header().Set("X-RateLimit-Limit", "30")
		w.WriteHeader(204) // We query the configured ping URL which should return a 204.
	}))
	defer ts.Close()

	cfg := &Config{
		Address:    ts.URL,
		Token:      "dummy-token",
		HTTPClient: ts.Client(),
	}

	connErr := errors.New("connection error")

	cases := map[string]struct {
		resp              *http.Response
		err               error
		retryServerErrors bool
		checkOK           bool
		checkErr          error
	}{
		"429-no-server-errors": {
			resp:     &http.Response{StatusCode: 429},
			err:      nil,
			checkOK:  true,
			checkErr: nil,
		},
		"429-with-server-errors": {
			resp:              &http.Response{StatusCode: 429},
			err:               nil,
			retryServerErrors: true,
			checkOK:           true,
			checkErr:          nil,
		},
		"500-no-server-errors": {
			resp:     &http.Response{StatusCode: 500},
			err:      nil,
			checkOK:  false,
			checkErr: nil,
		},
		"500-with-server-errors": {
			resp:              &http.Response{StatusCode: 500},
			err:               nil,
			retryServerErrors: true,
			checkOK:           true,
			checkErr:          nil,
		},
		"err-no-server-errors": {
			err:      connErr,
			checkOK:  false,
			checkErr: connErr,
		},
		"err-with-server-errors": {
			err:               connErr,
			retryServerErrors: true,
			checkOK:           true,
			checkErr:          connErr,
		},
	}

	ctx := context.Background()

	for name, tc := range cases {
		client, err := NewClient(cfg)
		if err != nil {
			t.Fatal(err)
		}

		client.RetryServerErrors(tc.retryServerErrors)

		checkOK, checkErr := client.retryHTTPCheck(ctx, tc.resp, tc.err)
		if checkOK != tc.checkOK {
			t.Fatalf("test %s expected checkOK %t, got: %t", name, tc.checkOK, checkOK)
		}
		if checkErr != tc.checkErr {
			t.Fatalf("test %s expected checkErr %v, got: %v", name, tc.checkErr, checkErr)
		}
	}
}

func TestClient_retryHTTPBackoff(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		w.Header().Set("X-RateLimit-Limit", "30")
		w.WriteHeader(204) // We query the configured ping URL which should return a 204.
	}))
	defer ts.Close()

	var attempts int
	retryLogHook := func(attemptNum int, resp *http.Response) {
		attempts++
	}

	cfg := &Config{
		Address:      ts.URL,
		Token:        "dummy-token",
		HTTPClient:   ts.Client(),
		RetryLogHook: retryLogHook,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatal(err)
	}

	retries := 3
	resp := &http.Response{StatusCode: 500}

	for i := 0; i < retries; i++ {
		client.retryHTTPBackoff(time.Second, time.Second, i, resp)
	}

	if attempts != retries {
		t.Fatalf("expected %d log hook callbacks, got: %d callbacks", retries, attempts)
	}
}

func setupEnvVars(token, address string) func() {
	origToken := os.Getenv("TFE_TOKEN")
	origAddress := os.Getenv("TFE_ADDRESS")

	os.Setenv("TFE_TOKEN", token)
	os.Setenv("TFE_ADDRESS", address)

	return func() {
		os.Setenv("TFE_TOKEN", origToken)
		os.Setenv("TFE_ADDRESS", origAddress)
	}
}
