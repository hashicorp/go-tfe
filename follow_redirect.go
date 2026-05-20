// Copyright IBM Corp. 2026
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
)

const maxRedirectHops = 10

// ErrRedirectLoop is returned when a redirect cycle is detected or the
// maximum number of redirect hops is exceeded.
var ErrRedirectLoop = errors.New("redirect loop detected")

// FollowAPIRedirect follows the redirect chain from an API response until it
// reaches the final response body, returning it as an io.ReadCloser. The caller
// is responsible for closing the returned body and for limiting the amount of
// data read (the response may be arbitrarily large).
//
// This is intended for API endpoints that return a 302 redirect to an Archivist
// presigned URL, which may in turn issue a 307 redirect to a storage backend
// (e.g., S3). The method intentionally does NOT reuse the client's own HTTP
// transport or send the Authorization header on follow-up requests, because the
// redirect targets are presigned URLs that are self-authenticating and must not
// receive the bearer token.
//
// Returns ErrRedirectLoop if a cycle is detected or more than 10 hops occur.
// Returns an error if the final response status is not 200.
func (c *Client) FollowAPIRedirect(ctx context.Context, resp *http.Response) (io.ReadCloser, error) {
	if resp == nil {
		return nil, errors.New("response must not be nil")
	}

	httpClient := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	visited := make(map[string]struct{})
	currentResp := resp

	for i := 0; i < maxRedirectHops; i++ {
		if currentResp.StatusCode == http.StatusOK {
			return currentResp.Body, nil
		}

		if currentResp.StatusCode != http.StatusFound &&
			currentResp.StatusCode != http.StatusMovedPermanently &&
			currentResp.StatusCode != http.StatusTemporaryRedirect &&
			currentResp.StatusCode != http.StatusPermanentRedirect &&
			currentResp.StatusCode != http.StatusSeeOther {
			currentResp.Body.Close() //nolint:errcheck
			return nil, fmt.Errorf("unexpected response status: %d", currentResp.StatusCode)
		}

		location := currentResp.Header.Get("Location")
		if location == "" {
			currentResp.Body.Close() //nolint:errcheck
			return nil, errors.New("redirect response missing Location header")
		}

		currentResp.Body.Close() //nolint:errcheck

		if _, ok := visited[location]; ok {
			return nil, ErrRedirectLoop
		}
		visited[location] = struct{}{}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, location, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create redirect request: %w", err)
		}

		currentResp, err = httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to follow redirect: %w", err)
		}
	}

	if currentResp != nil && currentResp.Body != nil {
		currentResp.Body.Close() //nolint:errcheck
	}
	return nil, ErrRedirectLoop
}
