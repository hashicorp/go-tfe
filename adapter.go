package tfe

import (
	"errors"
	nethttp "net/http"
	"time"

	abs "github.com/microsoft/kiota-abstractions-go"
	absauth "github.com/microsoft/kiota-abstractions-go/authentication"
	absser "github.com/microsoft/kiota-abstractions-go/serialization"
	khttp "github.com/microsoft/kiota-http-go"
	serjson "github.com/microsoft/kiota-serialization-json-go"
)

// DefaultRequestAdapter is the core service used by GraphServiceClient to make requests to Microsoft Graph.
type TFERequestAdapter struct {
	khttp.NetHttpRequestAdapter
	Client *nethttp.Client
}

func getMiddlewares() ([]khttp.Middleware, error) {
	retryOptions := khttp.RetryHandlerOptions{
		ShouldRetry: func(delay time.Duration, executionCount int, request *nethttp.Request, response *nethttp.Response) bool {
			// Retry on 425, 429, 5XX
			return executionCount < 5 && (response.StatusCode == 429 || response.StatusCode == 425 || response.StatusCode >= 500)
		},
	}
	redirectHandlerOptions := khttp.RedirectHandlerOptions{
		MaxRedirects: 5,
		ShouldRedirect: func(req *nethttp.Request, res *nethttp.Response) bool {
			return true
		},
	}
	compressionOptions := khttp.NewCompressionOptionsReference(false)
	userAgentHandlerOptions := khttp.UserAgentHandlerOptions{
		Enabled:        true,
		ProductName:    "go-tfe",
		ProductVersion: version,
	}

	return khttp.GetDefaultMiddlewaresWithOptions(
		&retryOptions,
		&redirectHandlerOptions,
		compressionOptions,
		&userAgentHandlerOptions,
	)
}

// NewRequestAdapter creates a new TFERequestAdapter with the given parameters
func NewRequestAdapter(baseURL string, authenticationProvider absauth.AuthenticationProvider, httpClient *nethttp.Client) (*TFERequestAdapter, error) {
	if authenticationProvider == nil {
		return nil, errors.New("authenticationProvider cannot be nil")
	}

	if httpClient == nil {
		middleware, err := getMiddlewares()
		if err != nil {
			return nil, err
		}
		httpClient = khttp.GetDefaultClient(middleware...)
	}

	defaultAdapter, err := khttp.NewNetHttpRequestAdapterWithParseNodeFactoryAndSerializationWriterFactoryAndHttpClient(authenticationProvider, absser.DefaultParseNodeFactoryInstance, absser.DefaultSerializationWriterFactoryInstance, httpClient)
	if err != nil {
		return nil, err
	}
	result := &TFERequestAdapter{
		NetHttpRequestAdapter: *defaultAdapter,
		Client:                httpClient,
	}

	result.SetBaseUrl(baseURL)

	setupDefaults()

	return result, nil
}

func setupDefaults() {
	abs.RegisterDefaultSerializer(func() absser.SerializationWriterFactory {
		return serjson.NewJsonSerializationWriterFactory()
	})
	abs.RegisterDefaultDeserializer(func() absser.ParseNodeFactory {
		return serjson.NewJsonParseNodeFactory()
	})
}
