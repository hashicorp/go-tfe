package tfe

import (
	"errors"
	nethttp "net/http"

	"github.com/hashicorp/go-tfe/middleware"
	abs "github.com/microsoft/kiota-abstractions-go"
	absauth "github.com/microsoft/kiota-abstractions-go/authentication"
	absser "github.com/microsoft/kiota-abstractions-go/serialization"
	khttp "github.com/microsoft/kiota-http-go"
	serjson "github.com/microsoft/kiota-serialization-json-go"
)

func init() {
	registerDefaults()
}

// DefaultRequestAdapter is the core service used by GraphServiceClient to make requests to Microsoft Graph.
type TFERequestAdapter struct {
	khttp.NetHttpRequestAdapter
	Client *nethttp.Client
}

func NewHTTPClient(options []middleware.MiddlewareOption) (*nethttp.Client, error) {
	middleware, err := middleware.GetForKiota(version, options...)
	if err != nil {
		return nil, err
	}
	httpClient := khttp.GetDefaultClient(middleware...)
	return httpClient, nil
}

// NewRequestAdapter creates a new TFERequestAdapter with the given parameters
func NewRequestAdapter(baseURL string, options []middleware.MiddlewareOption, authenticationProvider absauth.AuthenticationProvider) (*TFERequestAdapter, error) {
	if authenticationProvider == nil {
		return nil, errors.New("authenticationProvider cannot be nil")
	}

	httpClient, err := NewHTTPClient(options)
	if err != nil {
		return nil, err
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
	return result, nil
}

func registerDefaults() {
	abs.RegisterDefaultSerializer(func() absser.SerializationWriterFactory {
		return serjson.NewJsonSerializationWriterFactory()
	})
	abs.RegisterDefaultDeserializer(func() absser.ParseNodeFactory {
		return serjson.NewJsonParseNodeFactory()
	})
}
