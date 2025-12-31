// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	auth "github.com/microsoft/kiota-abstractions-go/authentication"

	"github.com/hashicorp/go-tfe/api"
	"github.com/hashicorp/go-tfe/api/models"
	"github.com/hashicorp/go-tfe/middleware"
)

const (
	_userAgent         = "go-tfe"
	_headerRateLimit   = "X-RateLimit-Limit"
	_headerRateReset   = "X-RateLimit-Reset"
	_headerAppName     = "TFP-AppName"
	_headerAPIVersion  = "TFP-API-Version"
	_headerTFEVersion  = "X-TFE-Version"
	_includeQueryParam = "include"

	DefaultAddress  = "https://app.terraform.io"
	DefaultBasePath = "/api/v2"
	// PingEndpoint is a no-op API endpoint used to configure the rate limiter
	PingEndpoint       = "ping"
	ContentTypeJSONAPI = "application/vnd.api+json"
	ContentTypeJSON    = "application/json"
)

// RetryHook allows a function to run before each retry.

type RetryHook func(attemptNum int, resp *http.Response)

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

	// RetryServerErrors enables the retry logic in the client.
	RetryServerErrors bool
}

// DefaultConfig returns a default config structure.

func DefaultConfig() *Config {
	config := &Config{
		Address:           DefaultAddress,
		BasePath:          DefaultBasePath,
		Token:             "",
		Headers:           make(http.Header),
		RetryServerErrors: false,
	}

	// Set the default user agent.
	config.Headers.Set("User-Agent", _userAgent)

	return config
}

// Client is the Terraform Enterprise API client. It provides the basic
// connectivity and configuration for accessing the TFE API
type Client struct {
	baseURL           *url.URL
	token             string
	headers           http.Header
	retryHook         RetryHook
	retryServerErrors bool
	adapter           *TFERequestAdapter
	API               *api.ApiClient

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
		accessToken:  cfg.Token,
		host:         baseURL.Host,
	}

	authProvider := auth.NewBaseBearerTokenAuthenticationProvider(tokenProvider)

	adapter, err := NewRequestAdapter(baseURL.String(), []middleware.MiddlewareOption{
		middleware.WithRetryServerErrorsOption(config.RetryServerErrors),
	}, authProvider)
	if err != nil {
		return nil, fmt.Errorf("error creating request adapter: %w", err)
	}

	// Create the client.
	client := &Client{
		baseURL:           baseURL,
		token:             config.Token,
		headers:           config.Headers,
		retryHook:         config.RetryHook,
		retryServerErrors: config.RetryServerErrors,
		adapter:           adapter,
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

// RetryServerErrors configures the retry HTTP check to also retry
// unexpected errors or requests that failed with a server error.
func (c *Client) RetryServerErrors(retry bool) {
	c.retryServerErrors = retry
}

// rateLimitBackoff provides a callback for Client.Backoff which will use the
// X-RateLimit_Reset header to determine the time to wait. We add some jitter
// to prevent a thundering herd.
//
// minimum and maximum are mainly used for bounding the jitter that will be added to
// the reset time retrieved from the headers. But if the final wait time is
// less than minimum, minimum will be used instead.
func rateLimitBackoff(minimum, maximum time.Duration, resp *http.Response) time.Duration {
	// rnd is used to generate pseudo-random numbers.
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))

	// First create some jitter bounded by the min and max durations.
	jitter := time.Duration(rnd.Float64() * float64(maximum-minimum))

	if resp != nil && resp.Header.Get(_headerRateReset) != "" {
		v := resp.Header.Get(_headerRateReset)
		reset, err := strconv.ParseFloat(v, 64)
		if err != nil {
			log.Fatal(err)
		}
		// Only update min if the given time to wait is longer
		if reset > 0 && time.Duration(reset*1e9) > minimum {
			minimum = time.Duration(reset * 1e9)
		}
	}

	return minimum + jitter
}

func SummarizeAPIErrors(err error) string {
	merr, ok := err.(*models.Errors)
	if !ok {
		return err.Error()
	}

	var sb strings.Builder
	for _, e := range merr.GetErrors() {
		if sb.Len() > 0 {
			sb.WriteString(", ")
		}
		detail := e.GetDetail()
		if detail != nil {
			sb.WriteString(fmt.Sprintf("%s: %s", *e.GetTitle(), *detail))
		} else {
			sb.WriteString(fmt.Sprintf("%s: %s", *e.GetStatus(), *e.GetTitle()))
		}
	}

	return sb.String()
}
