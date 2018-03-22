package tfe

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/hashicorp/go-cleanhttp"
	"github.com/manyminds/api2go/jsonapi"
)

const (
	// The default address of Terraform Enterprise. This value points to
	// the public SaaS service.
	DefaultAddress = "https://app.terraform.io"
)

// Config provides configuration details to the API client.
type Config struct {
	// The address of the Terraform Enterprise API. Defaults the the public
	// SaaS service address.
	Address string

	// API token used to access the Terraform Enterprise API.
	Token string

	// A custom HTTP client to use.
	HTTPClient *http.Client
}

// DefaultConfig returns a default config structure.
func DefaultConfig() *Config {
	return &Config{
		Address: DefaultAddress,
	}
}

// Client is the Terraform Enterprise API client. It provides the basic
// connectivity and configuration for accessing the TFE API.
type Client struct {
	config *Config
	http   *http.Client
}

// NewClient creates a new Terraform Enterprise API client.
func NewClient(c *Config) (*Client, error) {
	// Basic config validation. These values must be provided by the user
	// and no safe default can be assumed.
	if c == nil {
		return nil, fmt.Errorf("Missing client config")
	}
	if c.Token == "" {
		return nil, fmt.Errorf("Missing client token")
	}

	// Create the config - lay the provied options over the defaults.
	config := DefaultConfig()
	config.Token = c.Token
	if c.Address != "" {
		config.Address = c.Address
	}

	// Create the client.
	client := &Client{config: config}

	// Allow a custom HTTP client, or create a default one if it is empty.
	if c.HTTPClient != nil {
		client.http = c.HTTPClient
	} else {
		client.http = cleanhttp.DefaultClient()
	}

	return client, nil
}

// request is a convenient way of describing an HTTP request.
type request struct {
	method string
	path   string
	query  url.Values
	header http.Header
	body   io.Reader

	// Pointer to an input struct to serialize as JSONAPI. When provided, the
	// body parameter is ignored, and this is used instead.
	input interface{}

	// Pointer to an output structure to deserialize JSONAPI responses to. If
	// this is provided, on successful requests, the response body is
	// automatically decoded onto this field, the body is automatically closed,
	// and no HTTP response object is returned.
	output interface{}
}

// request is a helper to make HTTP requests to the Terraform Enterprise API.
// It is the reponsiblity of the caller to close any return response body, if
// any is present.
func (c *Client) do(r *request) (*http.Response, error) {
	// Form the full URL.
	u, err := url.Parse(c.config.Address)
	if err != nil {
		return nil, err
	}
	u.RawQuery = r.query.Encode()
	u.Path = r.path
	fullURL := u.String()

	// Get the request body to send, preferring an input struct over a raw body.
	body := r.body
	if r.input != nil {
		payloadBytes, err := jsonapi.Marshal(r.input)
		if err != nil {
			return nil, err
		}
		body = bytes.NewBuffer(payloadBytes)
	}

	req, err := http.NewRequest(r.method, fullURL, body)
	if err != nil {
		return nil, err
	}

	// Add the supplied headers.
	if r.header != nil {
		req.Header = r.header
	}

	// Add the auth token.
	req.Header.Set("Authorization", "Bearer "+c.config.Token)

	// Use JSONAPI content-type by default.
	if req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/vnd.api+json")
	}

	// Execute the request and check the response.
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Basic response checking.
	if err := checkResponseCode(resp); err != nil {
		return nil, err
	}

	// Decode the response, if an output was given.
	if r.output != nil {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		if err := jsonapi.Unmarshal(body, r.output); err != nil {
			return nil, err
		}

		return nil, nil
	}

	return resp, nil
}

// checkResponseCode can be used to check the status code of an HTTP request.
func checkResponseCode(r *http.Response) error {
	if r.StatusCode == 404 {
		return fmt.Errorf("Resource not found")
	}
	if r.StatusCode < 200 || r.StatusCode > 299 {
		body, _ := ioutil.ReadAll(r.Body)
		r.Body.Close()
		return fmt.Errorf(
			"Unexpected status code: %d\n\nBody:\n%s",
			r.StatusCode,
			body,
		)
	}
	return nil
}
