package tfe

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"reflect"

	"github.com/google/go-querystring/query"
	"github.com/hashicorp/go-cleanhttp"
	"github.com/svanharmelen/jsonapi"
)

// DefaultAddress of Terraform Enterprise.
const DefaultAddress = "https://app.terraform.io"

const (
	apiVersionPath = "/api/v2/"
	userAgent      = "go-tfe"
)

// Config provides configuration details to the API client.
type Config struct {
	// The address of the Terraform Enterprise API.
	Address string

	// API token used to access the Terraform Enterprise API.
	Token string

	// A custom HTTP client to use.
	HTTPClient *http.Client
}

// DefaultConfig returns a default config structure.
func DefaultConfig() *Config {
	config := &Config{
		Address: os.Getenv("TFE_ADDRESS"),
		Token:   os.Getenv("TFE_TOKEN"),
	}

	// Set the default address if none is given.
	if config.Address == "" {
		config.Address = DefaultAddress
	}

	// Set a default HTTP client if none given.
	if config.HTTPClient == nil {
		config.HTTPClient = cleanhttp.DefaultClient()
	}

	return config
}

// Client is the Terraform Enterprise API client. It provides the basic
// connectivity and configuration for accessing the TFE API.
type Client struct {
	baseURL   *url.URL
	token     string
	http      *http.Client
	userAgent string

	ConfigurationVersions *ConfigurationVersions
	Organizations         *Organizations
	Runs                  *Runs
	Workspaces            *Workspaces
}

// NewClient creates a new Terraform Enterprise API client.
func NewClient(cfg *Config) (*Client, error) {
	config := DefaultConfig()

	// Layer in the provided config for any non-blank values.
	if cfg != nil {
		if cfg.Address != "" {
			config.Address = cfg.Address
		}
		if cfg.Token != "" {
			config.Token = cfg.Token
		}
		if cfg.HTTPClient != nil {
			config.HTTPClient = cfg.HTTPClient
		}
	}

	// Parse the address to make sure its a valid URL.
	baseURL, err := url.Parse(config.Address)
	if err != nil {
		return nil, fmt.Errorf("Invalid address: %v", err)
	}
	baseURL.Path = apiVersionPath

	// This value must be provided by the user.
	if config.Token == "" {
		return nil, fmt.Errorf("Missing API token")
	}

	// Create the client.
	client := &Client{
		baseURL:   baseURL,
		token:     config.Token,
		http:      config.HTTPClient,
		userAgent: userAgent,
	}

	// Create the services.
	client.ConfigurationVersions = &ConfigurationVersions{client: client}
	client.Organizations = &Organizations{client: client}
	client.Runs = &Runs{client: client}
	client.Workspaces = &Workspaces{client: client}

	return client, nil
}

// ListOptions is used to specify pagination options when making API requests.
// Pagination allows breaking up large result sets into chunks, or "pages".
type ListOptions struct {
	// The page number to request. The results vary based on the PageSize.
	PageNumber int `url:"page[number],omitempty"`

	// The number of elements returned in a single page.
	PageSize int `url:"page[size],omitempty"`
}

// newRequest creates an API request. A relative URL path can be provided in
// path, in which case it is resolved relative to the apiVersionPath of the
// Client. Relative URL paths should always be specified without a preceding
// slash.
// If v is supplied, the value will be JSONAPI encoded and included as the
// request body. If the method is GET, the value will be parsed and added as
// query parameters.
func (c *Client) newRequest(method, path string, v interface{}) (*http.Request, error) {
	u, err := c.baseURL.Parse(path)
	if err != nil {
		return nil, err
	}

	req := &http.Request{
		Method:     method,
		URL:        u,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     make(http.Header),
		Host:       u.Host,
	}

	if v != nil {
		switch method {
		case "GET":
			q, err := query.Values(v)
			if err != nil {
				return nil, err
			}
			u.RawQuery = q.Encode()
		case "PATCH", "POST", "PUT":
			var body bytes.Buffer
			if err := jsonapi.MarshalPayloadWithoutIncluded(&body, v); err != nil {
				return nil, err
			}
			req.Body = ioutil.NopCloser(&body)
			req.ContentLength = int64(body.Len())
			req.Header.Set("Content-Type", "application/vnd.api+json")
		}
	}

	// Set the auth token.
	req.Header.Set("Authorization", "Bearer "+c.token)

	// Set a custom User-Agent.
	req.Header.Set("User-Agent", c.userAgent)

	return req, nil
}

// do sends an API request and returns the API response. The API response is
// JSONAPI decoded and stored in the value pointed to by v, or returned as an
// error if an API error has occurred.
// If v implements the io.Writer interface, the raw response body will be
// written to v, without attempting to first decode it.
func (c *Client) do(req *http.Request, v interface{}) (interface{}, error) {
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

	// Decode the response, if v was given.
	if v != nil {
		if reflect.TypeOf(v).Kind() == reflect.Slice {
			return jsonapi.UnmarshalManyPayload(resp.Body, reflect.TypeOf(v).Elem())
		}

		if w, ok := v.(io.Writer); ok {
			_, err = io.Copy(w, resp.Body)
		} else {
			err = jsonapi.UnmarshalPayload(resp.Body, v)
		}
	}

	return v, err
}

// TODO SvH: This logic to do this should be added to the newRequest method.
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

// TODO SvH: I suggest to use jsonapi.MarshalErrors instead.
// checkResponseCode can be used to check the status code of an HTTP request.
func checkResponseCode(r *http.Response) error {
	if r.StatusCode == 404 {
		return fmt.Errorf("Resource not found")
	}
	if r.StatusCode < 200 || r.StatusCode > 299 {
		body, _ := ioutil.ReadAll(r.Body)
		return fmt.Errorf(
			"Unexpected status code: %d\n\nBody:\n%s",
			r.StatusCode,
			body,
		)
	}
	return nil
}
