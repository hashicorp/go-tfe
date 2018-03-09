package tfe

import (
	"errors"
	"io"
	"net/http"
	"net/url"

	"github.com/hashicorp/go-cleanhttp"
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
		return nil, errors.New("Missing client config")
	}
	if c.Token == "" {
		return nil, errors.New("Missing client token")
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
}

// request is a helper to make HTTP requests to the Terraform Enterprise API.
// It is the reponsiblity of the caller to close any return response body.
func (c *Client) do(r *request) (*http.Response, error) {
	// Form the full URL.
	u, err := url.Parse(c.config.Address)
	if err != nil {
		return nil, err
	}
	u.RawQuery = r.query.Encode()
	u.Path = r.path
	fullURL := u.String()

	req, err := http.NewRequest(r.method, fullURL, r.body)
	if err != nil {
		return nil, err
	}

	// Add the headers.
	if r.header != nil {
		req.Header = r.header
	}
	req.Header.Set("Authorization", "Bearer "+c.config.Token)

	// Execute the query and return the result.
	return c.http.Do(req)
}
