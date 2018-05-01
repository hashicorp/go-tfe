package tfe

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"

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
	address := os.Getenv("TFE_ADDRESS")
	if address == "" {
		address = DefaultAddress
	}

	token := os.Getenv("TFE_TOKEN")

	return &Config{
		Address: address,
		Token:   token,
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
	config := DefaultConfig()

	// Layer in the provided config for any non-blank values.
	if c != nil {
		if c.Address != "" {
			config.Address = c.Address
		}
		if c.Token != "" {
			config.Token = c.Token
		}
		if c.HTTPClient != nil {
			config.HTTPClient = c.HTTPClient
		}
	}

	// Basic config validation. These values must be provided by the user
	// and no safe default can be assumed.
	if config.Token == "" {
		return nil, fmt.Errorf("Missing client token")
	}

	// Create the client.
	client := &Client{
		config: config,
		http:   config.HTTPClient,
	}

	// Populate the default HTTP client if none given.
	if client.http == nil {
		client.http = cleanhttp.DefaultClient()
	}

	return client, nil
}

// ListOptions is used to specify pagination options when making API requests.
// Pagination allows breaking up large result sets into chunks, or "pages".
type ListOptions struct {
	// The page number to request. The results vary based on the PageSize.
	PageNumber int

	// The number of elements returned in a single page.
	PageSize int
}

// request is a convenient way of describing an HTTP request.
type request struct {
	method string
	path   string
	query  url.Values
	header http.Header
	body   io.Reader
	lopt   ListOptions

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
	// Initialize members.
	if r.query == nil {
		r.query = url.Values{}
	}

	// Add the pagination options, if given.
	if r.lopt.PageNumber != 0 {
		r.query.Set("page[number]", strconv.Itoa(r.lopt.PageNumber))
	}
	if r.lopt.PageSize != 0 {
		r.query.Set("page[size]", strconv.Itoa(r.lopt.PageSize))
	}

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

	// Basic response checking.
	if err := checkResponseCode(resp); err != nil {
		resp.Body.Close()
		return nil, err
	}

	// Decode the response, if an output was given.
	if r.output != nil {
		body, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
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

// upload is a generic uploader helper which can be used to upload artifacts
// typically destined for an Archivist URL.
func (c *Client) upload(url string, data io.Reader) error {
	req, err := http.NewRequest("PUT", url, data)
	if err != nil {
		return err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return checkResponseCode(resp)
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
