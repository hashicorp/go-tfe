package tfe

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Compile-time proof of interface implementation.
var _ Applies = (*applies)(nil)

// Applies describes all the apply related methods that the Terraform
// Enterprise API supports.
//
// TFE API docs: https://www.terraform.io/docs/enterprise/api/apply.html
type Applies interface {
	// Read an apply by its ID.
	Read(ctx context.Context, applyID string) (*Apply, error)

	// Logs retrieves the logs of an apply.
	Logs(ctx context.Context, applyID string) (io.Reader, error)
}

// applies implements Applys.
type applies struct {
	client *Client
}

// ApplyStatus represents an apply state.
type ApplyStatus string

//List all available apply statuses.
const (
	ApplyCanceled   ApplyStatus = "canceled"
	ApplyCreated    ApplyStatus = "created"
	ApplyErrored    ApplyStatus = "errored"
	ApplyFinished   ApplyStatus = "finished"
	ApplyMFAWaiting ApplyStatus = "mfa_waiting"
	ApplyPending    ApplyStatus = "pending"
	ApplyQueued     ApplyStatus = "queued"
	ApplyRunning    ApplyStatus = "running"
)

// Apply represents a Terraform Enterprise apply.
type Apply struct {
	ID                   string                 `jsonapi:"primary,applies"`
	LogReadURL           string                 `jsonapi:"attr,log-read-url"`
	ResourceAdditions    int                    `jsonapi:"attr,resource-additions"`
	ResourceChanges      int                    `jsonapi:"attr,resource-changes"`
	ResourceDestructions int                    `jsonapi:"attr,resource-destructions"`
	Status               ApplyStatus            `jsonapi:"attr,status"`
	StatusTimestamps     *ApplyStatusTimestamps `jsonapi:"attr,status-timestamps"`
}

// ApplyStatusTimestamps holds the timestamps for individual apply statuses.
type ApplyStatusTimestamps struct {
	CanceledAt      time.Time `json:"canceled-at"`
	ErroredAt       time.Time `json:"errored-at"`
	FinishedAt      time.Time `json:"finished-at"`
	ForceCanceledAt time.Time `json:"force-canceled-at"`
	QueuedAt        time.Time `json:"queued-at"`
	StartedAt       time.Time `json:"started-at"`
}

// Read an apply by its ID.
func (s *applies) Read(ctx context.Context, applyID string) (*Apply, error) {
	if !validStringID(&applyID) {
		return nil, errors.New("Invalid value for apply ID")
	}

	u := fmt.Sprintf("applies/%s", url.QueryEscape(applyID))
	req, err := s.client.newRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	a := &Apply{}
	err = s.client.do(ctx, req, a)
	if err != nil {
		return nil, err
	}

	return a, nil
}

// Logs retrieves the logs of an apply.
func (s *applies) Logs(ctx context.Context, applyID string) (io.Reader, error) {
	if !validStringID(&applyID) {
		return nil, errors.New("Invalid value for apply ID")
	}

	// Get the apply to make sure it exists.
	a, err := s.Read(ctx, applyID)
	if err != nil {
		return nil, err
	}

	// Return an error if the log URL is empty.
	if a.LogReadURL == "" {
		return nil, fmt.Errorf("Apply %s does not have a log URL", applyID)
	}

	u, err := url.Parse(a.LogReadURL)
	if err != nil {
		return nil, fmt.Errorf("Invalid log URL: %v", err)
	}

	return &ApplyLogReader{
		client: s.client,
		ctx:    ctx,
		logURL: u,
		apply:  a,
	}, nil
}

// ApplyLogReader implements io.Reader for streaming apply logs.
type ApplyLogReader struct {
	client *Client
	ctx    context.Context
	logURL *url.URL
	offset int64
	apply  *Apply
	reads  uint64
}

func (r *ApplyLogReader) Read(l []byte) (int, error) {
	if written, err := r.read(l); err != io.ErrNoProgress {
		return written, err
	}

	// Loop until we can any data, the context is canceled or the apply
	// is finsished running. If we would return right away without any
	// data, we could and up causing a io.ErrNoProgress error.
	for {
		select {
		case <-r.ctx.Done():
			return 0, r.ctx.Err()
		case <-time.After(500 * time.Millisecond):
			if written, err := r.read(l); err != io.ErrNoProgress {
				return written, err
			}
		}
	}
}

func (r *ApplyLogReader) read(l []byte) (int, error) {
	// Update the query string.
	r.logURL.RawQuery = fmt.Sprintf("limit=%d&offset=%d", len(l), r.offset)

	// Create a new request.
	req, err := http.NewRequest("GET", r.logURL.String(), nil)
	if err != nil {
		return 0, err
	}
	req = req.WithContext(r.ctx)

	// Retrieve the next chunk.
	resp, err := r.client.http.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	// Basic response checking.
	if err := checkResponseCode(resp); err != nil {
		return 0, err
	}

	// Read the retrieved chunk.
	written, err := resp.Body.Read(l)
	if err != nil && err != io.EOF {
		// Ignore io.EOF errors returned when reading from the response
		// body as this indicates the end of the chunk and not the end
		// of the logfile.
		return written, err
	}

	// Check if we need to continue the loop and wait 500 miliseconds
	// before checking if there is a new chunk available or that the
	// apply is finished and we are done reading all chunks.
	if written == 0 {
		if r.reads%2 == 0 {
			r.apply, err = r.client.Applies.Read(r.ctx, r.apply.ID)
			if err != nil {
				return 0, err
			}
		}

		switch r.apply.Status {
		case ApplyCanceled, ApplyErrored, ApplyFinished:
			return 0, io.EOF
		default:
			r.reads++
			return 0, io.ErrNoProgress
		}
	}

	// Update the offset for the next read.
	r.offset += int64(written)

	return written, nil
}
