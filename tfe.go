// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	auth "github.com/microsoft/kiota-abstractions-go/authentication"

	"github.com/google/go-querystring/query"
	cleanhttp "github.com/hashicorp/go-cleanhttp"
	retryablehttp "github.com/hashicorp/go-retryablehttp"
	"github.com/hashicorp/go-tfe/api"
	"github.com/hashicorp/go-tfe/api/models"
	"github.com/hashicorp/jsonapi"
	"golang.org/x/time/rate"
)

const (
	_userAgent         = "go-tfe"
	_headerRateLimit   = "X-RateLimit-Limit"
	_headerRateReset   = "X-RateLimit-Reset"
	_headerAppName     = "TFP-AppName"
	_headerAPIVersion  = "TFP-API-Version"
	_headerTFEVersion  = "X-TFE-Version"
	_includeQueryParam = "include"

	DefaultAddress      = "https://app.terraform.io"
	DefaultBasePath     = "/api/v2"
	DefaultRegistryPath = "/api/registry/"
	// PingEndpoint is a no-op API endpoint used to configure the rate limiter
	PingEndpoint       = "ping"
	ContentTypeJSONAPI = "application/vnd.api+json"
	ContentTypeJSON    = "application/json"
)

// RetryLogHook allows a function to run before each retry.

type RetryLogHook func(attemptNum int, resp *http.Response)

// Config provides configuration details to the API client.

type Config struct {
	// The address of the Terraform Enterprise API.
	Address string

	// The base path on which the API is served.
	BasePath string

	// The base path for the Registry API
	RegistryBasePath string

	// API token used to access the Terraform Enterprise API.
	Token string

	// Headers that will be added to every request.
	Headers http.Header

	// A custom HTTP client to use.
	HTTPClient *http.Client

	// RetryLogHook is invoked each time a request is retried.
	RetryLogHook RetryLogHook

	// RetryServerErrors enables the retry logic in the client.
	RetryServerErrors bool
}

// DefaultConfig returns a default config structure.

func DefaultConfig() *Config {
	config := &Config{
		Address:           DefaultAddress,
		BasePath:          DefaultBasePath,
		RegistryBasePath:  DefaultRegistryPath,
		Token:             "",
		Headers:           make(http.Header),
		HTTPClient:        cleanhttp.DefaultPooledClient(),
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
	registryBaseURL   *url.URL
	token             string
	headers           http.Header
	http              *retryablehttp.Client
	limiter           *rate.Limiter
	retryLogHook      RetryLogHook
	retryServerErrors bool
	API               *api.ApiClient

	Meta Meta
}

// Meta contains any HCP Terraform APIs which provide data about the API itself.
type Meta struct {
	IPRanges IPRanges
}

// DoObjectPUTRequest performs a PUT request using the specific data body. The Content-Type
// header is set to application/octet-stream but no Authentication header is sent. No response
// body is decoded.
func (c *Client) DoObjectPUTRequest(ctx context.Context, foreignURL string, data io.Reader) error {
	u, err := url.Parse(foreignURL)
	if err != nil {
		return fmt.Errorf("specified URL was not valid: %w", err)
	}

	reqHeaders := make(http.Header)
	reqHeaders.Set("Accept", "application/json, */*")
	reqHeaders.Set("Content-Type", "application/octet-stream")

	req, err := retryablehttp.NewRequest("PUT", u.String(), data)
	if err != nil {
		return err
	}

	// Set the default headers.
	for k, v := range c.headers {
		req.Header[k] = v
	}

	// Set the request specific headers.
	for k, v := range reqHeaders {
		req.Header[k] = v
	}

	request := &ClientRequest{
		retryableRequest: req,
		http:             c.http,
		Header:           req.Header,
	}

	return request.DoJSON(ctx, nil)
}

func (c *Client) NewJSONAPIRequest(method, path string, reqBody, queryParams any) (*ClientRequest, error) {
	u, err := c.baseURL.Parse(path)
	if err != nil {
		return nil, err
	}

	// Create a request specific headers map.
	reqHeaders := make(http.Header)
	reqHeaders.Set("Authorization", "Bearer "+c.token)
	reqHeaders.Set("Accept", ContentTypeJSONAPI)

	var body any
	if reqBody != nil && (method == "DELETE" || method == "PATCH" || method == "POST" || method == "PUT") {
		reqHeaders.Set("Content-Type", ContentTypeJSONAPI)

		if body, err = serializeRequestBody(reqBody); err != nil {
			return nil, err
		}
	}

	qv, err := query.Values(queryParams)
	if err != nil {
		return nil, err
	}

	u.RawQuery = encodeQueryParams(qv)
	req, err := retryablehttp.NewRequest(method, u.String(), body)
	if err != nil {
		return nil, err
	}

	// Set the default headers.
	for k, v := range c.headers {
		req.Header[k] = v
	}

	// Set the request specific headers.
	for k, v := range reqHeaders {
		req.Header[k] = v
	}

	return &ClientRequest{
		retryableRequest: req,
		http:             c.http,
		Header:           req.Header,
	}, nil
}

// NewJSONRequest performs some basic API request preparation based on the method
func (c *Client) NewJSONRequest(method, path string, reqBody, queryParams any) (*ClientRequest, error) {
	u, err := c.baseURL.Parse(path)
	if err != nil {
		return nil, err
	}

	// Create a request specific headers map.
	reqHeaders := make(http.Header)
	reqHeaders.Set("Authorization", "Bearer "+c.token)
	reqHeaders.Set("Accept", ContentTypeJSON)

	var body any
	if reqBody != nil && (method == "DELETE" || method == "PATCH" || method == "POST" || method == "PUT") {
		reqHeaders.Set("Content-Type", ContentTypeJSONAPI)

		if body, err = serializeRequestBody(reqBody); err != nil {
			return nil, err
		}
	}

	qv, err := query.Values(queryParams)
	if err != nil {
		return nil, err
	}

	u.RawQuery = encodeQueryParams(qv)
	req, err := retryablehttp.NewRequest(method, u.String(), body)
	if err != nil {
		return nil, err
	}

	// Set the default headers.
	for k, v := range c.headers {
		req.Header[k] = v
	}

	// Set the request specific headers.
	for k, v := range reqHeaders {
		req.Header[k] = v
	}

	return &ClientRequest{
		retryableRequest: req,
		http:             c.http,
		Header:           req.Header,
	}, nil
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
		if cfg.RegistryBasePath != "" {
			config.RegistryBasePath = cfg.RegistryBasePath
		}
		if cfg.Token != "" {
			config.Token = cfg.Token
		}
		for k, v := range cfg.Headers {
			config.Headers[k] = v
		}
		if cfg.HTTPClient != nil {
			config.HTTPClient = cfg.HTTPClient
		}
		if cfg.RetryLogHook != nil {
			config.RetryLogHook = cfg.RetryLogHook
		}
		config.RetryServerErrors = cfg.RetryServerErrors
	}

	// Parse the address to make sure its a valid URL.
	baseURL, err := url.Parse(config.Address)
	if err != nil {
		return nil, fmt.Errorf("invalid address: %w", err)
	}

	baseURL.Path = config.BasePath

	registryURL, err := url.Parse(config.Address)
	if err != nil {
		return nil, fmt.Errorf("invalid address: %w", err)
	}

	registryURL.Path = config.RegistryBasePath

	// This value must be provided by the user.
	if config.Token == "" {
		return nil, fmt.Errorf("missing API token")
	}

	// Create the client.
	client := &Client{
		baseURL:           baseURL,
		registryBaseURL:   registryURL,
		token:             config.Token,
		headers:           config.Headers,
		retryLogHook:      config.RetryLogHook,
		retryServerErrors: config.RetryServerErrors,
	}

	client.http = &retryablehttp.Client{
		Backoff:      client.retryHTTPBackoff,
		CheckRetry:   client.retryHTTPCheck,
		ErrorHandler: retryablehttp.PassthroughErrorHandler,
		HTTPClient:   config.HTTPClient,
		RetryWaitMin: 100 * time.Millisecond,
		RetryWaitMax: 400 * time.Millisecond,
		RetryMax:     30,
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

	adapter, err := NewRequestAdapter(baseURL.String(), authProvider, client.http.HTTPClient)
	if err != nil {
		return nil, fmt.Errorf("error creating request adapter: %w", err)
	}
	client.API = api.NewApiClient(adapter)

	client.Meta = Meta{
		IPRanges: &ipRanges{
			client: client,
		},
	}

	client.limiter = rate.NewLimiter(rate.Inf, 0)

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

// retryHTTPCheck provides a callback for Client.CheckRetry which
// will retry both rate limit (429) and server (>= 500) errors.
func (c *Client) retryHTTPCheck(ctx context.Context, resp *http.Response, err error) (bool, error) {
	if ctx.Err() != nil {
		return false, ctx.Err()
	}
	if err != nil {
		return c.retryServerErrors, err
	}
	if resp.StatusCode == 429 || (c.retryServerErrors && resp.StatusCode >= 500) {
		return true, nil
	}
	return false, nil
}

// retryHTTPBackoff provides a generic callback for Client.Backoff which
// will pass through all calls based on the status code of the response.
func (c *Client) retryHTTPBackoff(minimum, maximum time.Duration, attemptNum int, resp *http.Response) time.Duration {
	if c.retryLogHook != nil {
		c.retryLogHook(attemptNum, resp)
	}

	// Use the rate limit backoff function when we are rate limited.
	if resp != nil && resp.StatusCode == 429 {
		return rateLimitBackoff(minimum, maximum, resp)
	}

	// Set custom duration's when we experience a service interruption.
	minimum = 700 * time.Millisecond
	maximum = 900 * time.Millisecond

	return retryablehttp.LinearJitterBackoff(minimum, maximum, attemptNum, resp)
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
		sb.WriteString(*e.GetTitle())
	}

	return sb.String()
}

// ConfigureLimiter configures the rate limiter.
func (c *Client) ConfigureLimiter(rawLimit string) {
	// Set default values for when rate limiting is disabled.
	limit := rate.Inf
	burst := 0

	if v := rawLimit; v != "" {
		if rateLimit, err := strconv.ParseFloat(v, 64); rateLimit > 0 {
			if err != nil {
				log.Fatal(err)
			}
			// Configure the limit and burst using a split of 2/3 for the limit and
			// 1/3 for the burst. This enables clients to burst 1/3 of the allowed
			// calls before the limiter kicks in. The remaining calls will then be
			// spread out evenly using intervals of time.Second / limit which should
			// prevent hitting the rate limit.
			limit = rate.Limit(rateLimit * 0.66)
			burst = int(rateLimit * 0.33)
		}
	}

	// Create a new limiter using the calculated values.
	c.limiter = rate.NewLimiter(limit, burst)
}

// encodeQueryParams encodes the values into "URL encoded" form
// ("bar=baz&foo=quux") sorted by key. This version behaves as url.Values
// Encode, except that it encodes certain keys as comma-separated values instead
// of using multiple keys.
func encodeQueryParams(v url.Values) string {
	if v == nil {
		return ""
	}
	var buf strings.Builder
	keys := make([]string, 0, len(v))
	for k := range v {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		vs := v[k]
		if len(vs) > 1 && validSliceKey(k) {
			val := strings.Join(vs, ",")
			vs = vs[:0]
			vs = append(vs, val)
		}
		keyEscaped := url.QueryEscape(k)

		for _, v := range vs {
			if buf.Len() > 0 {
				buf.WriteByte('&')
			}
			buf.WriteString(keyEscaped)
			buf.WriteByte('=')
			buf.WriteString(url.QueryEscape(v))
		}
	}
	return buf.String()
}

// serializeRequestBody serializes the given ptr or ptr slice into a JSON
// request. It automatically uses jsonapi or json serialization, depending
// on the body type's tags.
func serializeRequestBody(v interface{}) (interface{}, error) {
	// The body can be a slice of pointers or a pointer. In either
	// case we want to choose the serialization type based on the
	// individual record type. To determine that type, we need
	// to either follow the pointer or examine the slice element type.
	// There are other theoretical possibilities (e. g. maps,
	// non-pointers) but they wouldn't work anyway because the
	// json-api library doesn't support serializing other things.
	var modelType reflect.Type
	bodyType := reflect.TypeOf(v)
	switch bodyType.Kind() {
	case reflect.Slice:
		sliceElem := bodyType.Elem()
		if sliceElem.Kind() != reflect.Ptr {
			return nil, ErrInvalidRequestBody
		}
		modelType = sliceElem.Elem()
	case reflect.Ptr:
		modelType = reflect.ValueOf(v).Elem().Type()
	default:
		return nil, ErrInvalidRequestBody
	}

	// Infer whether the request uses jsonapi or regular json
	// serialization based on how the fields are tagged.
	jsonAPIFields := 0
	jsonFields := 0
	for i := 0; i < modelType.NumField(); i++ {
		structField := modelType.Field(i)
		if structField.Tag.Get("jsonapi") != "" {
			jsonAPIFields++
		}
		if structField.Tag.Get("json") != "" {
			jsonFields++
		}
	}
	if jsonAPIFields > 0 && jsonFields > 0 {
		// Defining a struct with both json and jsonapi tags doesn't
		// make sense, because a struct can only be serialized
		// as one or another. If this does happen, it's a bug
		// in the library that should be fixed at development time
		return nil, ErrInvalidStructFormat
	}

	if jsonFields > 0 {
		return json.Marshal(v)
	}
	buf := bytes.NewBuffer(nil)
	if err := jsonapi.MarshalPayloadWithoutIncluded(buf, v); err != nil {
		return nil, err
	}
	return buf, nil
}

func unmarshalResponse(responseBody io.Reader, model interface{}) error {
	// Get the value of model so we can test if it's a struct.
	dst := reflect.Indirect(reflect.ValueOf(model))

	// Return an error if model is not a struct or an io.Writer.
	if dst.Kind() != reflect.Struct {
		return fmt.Errorf("%v must be a struct or an io.Writer", dst)
	}

	// Try to get the Items and Pagination struct fields.
	items := dst.FieldByName("Items")

	// Unmarshal a single value if model does not contain the
	// Items and Pagination struct fields.
	if !items.IsValid() {
		return jsonapi.UnmarshalPayload(responseBody, model)
	}

	// Return an error if model.Items is not a slice.
	if items.Type().Kind() != reflect.Slice {
		return ErrItemsMustBeSlice
	}

	// Create a temporary buffer and copy all the read data into it.
	body := bytes.NewBuffer(nil)
	reader := io.TeeReader(responseBody, body)

	// Unmarshal as a list of values as model.Items is a slice.
	raw, err := jsonapi.UnmarshalManyPayload(reader, items.Type().Elem())
	if err != nil {
		return err
	}

	// Make a new slice to hold the results.
	sliceType := reflect.SliceOf(items.Type().Elem())
	result := reflect.MakeSlice(sliceType, 0, len(raw))

	// Add all of the results to the new slice.
	for _, v := range raw {
		result = reflect.Append(result, reflect.ValueOf(v))
	}

	// Pointer-swap the result.
	items.Set(result)

	pagination := dst.FieldByName("Pagination")
	paginationWithoutTotals := dst.FieldByName("PaginationNextPrev")

	// As we are getting a list of values, we need to decode
	// the pagination details out of the response body.
	// Pointer-swap the decoded pagination details.
	if paginationWithoutTotals.IsValid() {
		p, err := parsePaginationWithoutTotal(body)
		if err != nil {
			return err
		}
		paginationWithoutTotals.Set(reflect.ValueOf(p))
	} else if pagination.IsValid() {
		p, err := parsePagination(body)
		if err != nil {
			return err
		}
		pagination.Set(reflect.ValueOf(p))
	}

	return nil
}

// ListOptions is used to specify pagination options when making API requests.
// Pagination allows breaking up large result sets into chunks, or "pages".
type ListOptions struct {
	// The page number to request. The results vary based on the PageSize.
	PageNumber int `url:"page[number],omitempty"`

	// The number of elements returned in a single page.
	PageSize int `url:"page[size],omitempty"`
}

// PaginationNextPrev is used to return the pagination details of an API request.
type PaginationNextPrev struct {
	CurrentPage  int `json:"current-page"`
	PreviousPage int `json:"prev-page"`
	NextPage     int `json:"next-page"`
}

// Pagination is used to return the pagination details of an API request including TotalCount.
type Pagination struct {
	CurrentPage  int `json:"current-page"`
	PreviousPage int `json:"prev-page"`
	NextPage     int `json:"next-page"`
	TotalCount   int `json:"total-count"`
	TotalPages   int `json:"total-pages"`
}

func parsePaginationWithoutTotal(body io.Reader) (*PaginationNextPrev, error) {
	var raw struct {
		Meta struct {
			Pagination PaginationNextPrev `jsonapi:"pagination"`
		} `jsonapi:"meta"`
	}

	// JSON decode the raw response.
	if err := json.NewDecoder(body).Decode(&raw); err != nil {
		return &PaginationNextPrev{}, err
	}

	return &raw.Meta.Pagination, nil
}

func parsePagination(body io.Reader) (*Pagination, error) {
	var raw struct {
		Meta struct {
			Pagination Pagination `jsonapi:"pagination"`
		} `jsonapi:"meta"`
	}

	// JSON decode the raw response.
	if err := json.NewDecoder(body).Decode(&raw); err != nil {
		return &Pagination{}, err
	}

	return &raw.Meta.Pagination, nil
}

// checkResponseCode refines typical API errors into more specific errors
// if possible. It returns nil if the response code < 400
func checkResponseCode(r *http.Response) error {
	if r.StatusCode >= 200 && r.StatusCode < 400 {
		return nil
	}

	var errs []string
	var err error

	switch r.StatusCode {
	case 400:
		errs, err = decodeErrorPayload(r)
		if err != nil {
			return err
		}

		if errorPayloadContains(errs, "include parameter") {
			return ErrInvalidIncludeValue
		}
		return errors.New(strings.Join(errs, "\n"))
	case 401:
		return ErrUnauthorized
	case 404:
		return ErrResourceNotFound
	}

	errs, err = decodeErrorPayload(r)
	if err != nil {
		return err
	}

	return errors.New(strings.Join(errs, "\n"))
}

func decodeErrorPayload(r *http.Response) ([]string, error) {
	// Decode the error payload.
	var errs []string
	errPayload := &jsonapi.ErrorsPayload{}
	err := json.NewDecoder(r.Body).Decode(errPayload)
	if err != nil || len(errPayload.Errors) == 0 {
		return errs, errors.New(r.Status)
	}

	// Parse and format the errors.
	for _, e := range errPayload.Errors {
		if e.Detail == "" {
			errs = append(errs, e.Title)
		} else {
			errs = append(errs, fmt.Sprintf("%s\n\n%s", e.Title, e.Detail))
		}
	}

	return errs, nil
}

func errorPayloadContains(payloadErrors []string, match string) bool {
	for _, e := range payloadErrors {
		if strings.Contains(e, match) {
			return true
		}
	}
	return false
}

func validSliceKey(key string) bool {
	return key == _includeQueryParam || strings.Contains(key, "filter[")
}
