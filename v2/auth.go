package tfe

import (
	"context"
	u "net/url"
	"sync"

	auth "github.com/microsoft/kiota-abstractions-go/authentication"
)

type accessTokenProvider struct {
	allowedHosts *auth.AllowedHostsValidator
	host         string
	accessToken  string
	mu           sync.Mutex
}

var _ auth.AccessTokenProvider = &accessTokenProvider{}

func (c *accessTokenProvider) GetAllowedHostsValidator() *auth.AllowedHostsValidator {
	return c.allowedHosts
}

func (c *accessTokenProvider) GetAuthorizationToken(ctx context.Context, url *u.URL, additionalAuthenticationContext map[string]interface{}) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.accessToken, nil
}
