// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"fmt"
	"net/http"
	"net/url"

	auth "github.com/microsoft/kiota-abstractions-go/authentication"

	"github.com/hashicorp/go-tfe/api"
	"github.com/hashicorp/go-tfe/middleware"
)

const (
	DefaultUserAgent = "go-tfe"

	DefaultAddress  = "https://app.terraform.io"
	DefaultBasePath = "/api/v2"
	// PingEndpoint is a no-op API endpoint used to configure the rate limiter
	PingEndpoint       = "ping"
	ContentTypeJSONAPI = "application/vnd.api+json"
	ContentTypeJSON    = "application/json"
)

// RetryHook allows a function to run before each retry.
type RetryHook = middleware.RetryHookCallback

// Config provides configuration details to the API client.
type Config struct {
	// The address of the Terraform Enterprise API.
	Address string

	// The base path on which the API is served.
	BasePath string

	// API token used to access the Terraform Enterprise API.
	Token string

	// Headers that will be added to every request.
	Headers http.Header

	// RetryHook is invoked each time a request is retried.
	RetryHook RetryHook

	// Retry enables automatic retries for rate limited requests.
	RetryRateLimited bool

	// RetryServerErrors enables the retry logic in the client.
	RetryServerErrors bool

	// RetryMaxRetries sets the maximum number of retries for a request before giving up.
	RetryMaxRetries int

	// UserAgent is the User-Agent header value sent with each request.
	// If not set, a default value ("go-tfe") will be used.
	UserAgent string
}

// DefaultConfig returns a default config structure.
func DefaultConfig() *Config {
	config := &Config{
		Address:           DefaultAddress,
		BasePath:          DefaultBasePath,
		Token:             "",
		Headers:           make(http.Header),
		RetryRateLimited:  false,
		RetryServerErrors: false,
		RetryMaxRetries:   5,
		RetryHook:         func(retryCount int, response *http.Response) {},
		UserAgent:         DefaultUserAgent,
	}

	// Set the default user agent.
	config.Headers.Set("User-Agent", config.UserAgent)

	return config
}

// Client is the Terraform Enterprise API client. It provides the basic
// connectivity and configuration for accessing the TFE API
type Client struct {
	baseURL *url.URL
	token   string
	headers http.Header
	adapter *TFERequestAdapter
	API     *api.ApiClient

	Meta Meta
}

// Meta contains any HCP Terraform APIs which provide data about the API itself.
type Meta struct {
	IPRanges IPRanges
}

// NewClient creates a new Terraform Enterprise API client.
func NewClient(cfg *Config) (*Client, error) {
	config := DefaultConfig()

	// Layer in the provided config for any non-blank values.
	if cfg != nil { // nolint
		if cfg.Address != "" {
			config.Address = cfg.Address
		}
		if cfg.BasePath != "" {
			config.BasePath = cfg.BasePath
		}
		if cfg.Token != "" {
			config.Token = cfg.Token
		}
		for k, v := range cfg.Headers {
			config.Headers[k] = v
		}
		if cfg.RetryHook != nil {
			config.RetryHook = cfg.RetryHook
		}
		config.RetryServerErrors = cfg.RetryServerErrors
		config.RetryRateLimited = cfg.RetryRateLimited
		config.RetryMaxRetries = cfg.RetryMaxRetries

		if cfg.UserAgent != "" {
			config.UserAgent = cfg.UserAgent
			config.Headers.Set("User-Agent", config.UserAgent)
		}
	}

	// Parse the address to make sure its a valid URL.
	baseURL, err := url.Parse(config.Address)
	if err != nil {
		return nil, fmt.Errorf("invalid address: %w", err)
	}

	baseURL.Path = config.BasePath

	// This value must be provided by the user.
	if config.Token == "" {
		return nil, fmt.Errorf("missing API token")
	}

	validator, err := auth.NewAllowedHostsValidatorErrorCheck([]string{
		baseURL.Host,
	})
	if err != nil {
		return nil, fmt.Errorf("invalid host configuration: %w", err)
	}

	tokenProvider := &accessTokenProvider{
		allowedHosts: validator,
		accessToken:  config.Token,
		host:         baseURL.Host,
	}

	authProvider := auth.NewBaseBearerTokenAuthenticationProvider(tokenProvider)

	adapter, err := NewRequestAdapter(baseURL.String(), []middleware.MiddlewareOption{
		middleware.WithErrorInterceptorOption(APIErrorFactory),
		middleware.WithRetryOptions(config.RetryRateLimited, config.RetryServerErrors, config.RetryMaxRetries, config.RetryHook),
	}, authProvider)
	if err != nil {
		return nil, fmt.Errorf("error creating request adapter: %w", err)
	}

	// Create the client.
	client := &Client{
		baseURL: baseURL,
		token:   config.Token,
		headers: config.Headers,
		adapter: adapter,
	}

	client.API = api.NewApiClient(adapter)
	client.Meta = Meta{
		IPRanges: &ipRanges{
			client: client,
		},
	}

	return client, nil
}

// BaseURL returns the base URL as configured in the client
func (c Client) BaseURL() url.URL {
	return *c.baseURL
}
