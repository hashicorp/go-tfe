// Copyright IBM Corp. 2018, 2026
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"net/url"
	"testing"

	auth "github.com/microsoft/kiota-abstractions-go/authentication"
	"github.com/stretchr/testify/assert"
)

func TestAccessTokenProvider_GetAuthorizationToken_OnlyAllowedHosts(t *testing.T) {
	validator, err := auth.NewAllowedHostsValidatorErrorCheck([]string{"app.terraform.io"})
	if err != nil {
		t.Fatalf("unexpected validator error: %v", err)
	}

	provider := &accessTokenProvider{
		allowedHosts: validator,
		accessToken:  "test-token",
	}

	tests := []struct {
		name     string
		rawURL   string
		expected string
	}{
		{
			name:     "allowed host returns token",
			rawURL:   "https://app.terraform.io/api/v2/account/details",
			expected: "test-token",
		},
		{
			name:     "disallowed host returns empty token",
			rawURL:   "https://example.com/api/v2/account/details",
			expected: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			requestURL, err := url.Parse(tc.rawURL)
			if err != nil {
				t.Fatalf("unexpected parse error: %v", err)
			}

			token, err := provider.GetAuthorizationToken(context.Background(), requestURL, nil)
			if err != nil {
				t.Fatalf("unexpected token error: %v", err)
			}

			assert.Equal(t, tc.expected, token)
		})
	}
}
