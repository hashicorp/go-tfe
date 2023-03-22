// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type wellKnownJwks struct {
	Keys []struct {
		Kid string `json:"kid"`
	} `json:"keys"`
}

func TestAdminSettings_Oidc_RotateKey(t *testing.T) {
	skipUnlessEnterprise(t)

	client := testClient(t)
	jwksClient := http.Client{
		Timeout: time.Second * 2,
	}
	baseURL := client.baseURL
	token := client.token

	ctx := context.Background()

	jwks, err := getJwks(jwksClient, baseURL, token)
	require.NoError(t, err)

	// Don't assume there is only 1 key to start
	originalNumKeys := len(jwks.Keys)

	err = client.Admin.Settings.OIDC.RotateKey(ctx)
	require.NoError(t, err)

	jwks, err = getJwks(jwksClient, baseURL, token)
	require.NoError(t, err)

	newNumKeys := len(jwks.Keys)

	// Rotate should add 1 additional key
	assert.Equal(t, originalNumKeys+1, newNumKeys)
}

func TestAdminSettings_Oidc_TrimKey(t *testing.T) {
	skipUnlessEnterprise(t)

	client := testClient(t)
	jwksClient := http.Client{
		Timeout: time.Second * 2,
	}
	baseURL := client.baseURL
	token := client.token

	ctx := context.Background()

	jwks, err := getJwks(jwksClient, baseURL, token)
	require.NoError(t, err)

	// Don't assume there is only 1 key to start
	originalNumKeys := len(jwks.Keys)

	originalKids := make([]string, originalNumKeys)

	for i := 0; i < originalNumKeys; i++ {
		originalKids[i] = jwks.Keys[i].Kid
	}

	err = client.Admin.Settings.OIDC.RotateKey(ctx)
	require.NoError(t, err)

	jwks, err = getJwks(jwksClient, baseURL, token)
	require.NoError(t, err)

	beforeTrimNumKeys := len(jwks.Keys)

	assert.Equal(t, originalNumKeys+1, beforeTrimNumKeys)

	err = client.Admin.Settings.OIDC.TrimKey(ctx)
	require.NoError(t, err)

	jwks, err = getJwks(jwksClient, baseURL, token)
	require.NoError(t, err)

	afterTrimNumKeys := len(jwks.Keys)

	assert.Equal(t, 1, afterTrimNumKeys)

	// Make sure we actually trimmed the keys
	assert.NotContains(t, originalKids, jwks.Keys[0].Kid)
}

func getJwks(client http.Client, baseURL *url.URL, token string) (*wellKnownJwks, error) {
	jwksEndpoint, err := baseURL.Parse("/.well-known/jwks")
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodGet, jwksEndpoint.String(), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	res, getErr := client.Do(req)
	if getErr != nil {
		return nil, getErr
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status code: %d. Expected a 200 response", res.StatusCode)
	}

	var result wellKnownJwks
	jsonErr := json.NewDecoder(res.Body).Decode(&result)
	if jsonErr != nil {
		return nil, jsonErr
	}

	return &result, nil
}
