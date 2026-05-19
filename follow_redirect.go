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

var ErrRedirectLoop = errors.New("redirect loop detected")

func (c *Client) FollowAPIRedirect(ctx context.Context, resp *http.Response) (io.ReadCloser, error) {
	if resp == nil {
		return nil, errors.New("response must not be nil")
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

		httpClient := &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
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
