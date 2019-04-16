package tfe

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"time"
)

// Compile-time proof of interface implementation.
var _ CostEstimations = (*costEstimations)(nil)

// CostEstimations describes all the costEstimation related methods that
// the Terraform Enterprise API supports.
//
// TFE API docs: https://www.terraform.io/docs/enterprise/api/ (TBD)
type CostEstimations interface {
	// Read a costEstimation by its ID.
	Read(ctx context.Context, costEstimationID string) (*CostEstimation, error)

	// Logs retrieves the logs of a costEstimation.
	Logs(ctx context.Context, costEstimationID string) (io.Reader, error)
}

// costEstimations implements CostEstimations.
type costEstimations struct {
	client *Client
}

// CostEstimationStatus represents a costEstimation state.
type CostEstimationStatus string

//List all available costEstimation statuses.
const (
	CostEstimationCanceled CostEstimationStatus = "canceled"
	CostEstimationErrored  CostEstimationStatus = "errored"
	CostEstimationFinished CostEstimationStatus = "finished"
	CostEstimationQueued   CostEstimationStatus = "queued"
)

// CostEstimation represents a Terraform Enterprise costEstimation.
type CostEstimation struct {
	ID               string                          `jsonapi:"primary,cost-estimations"`
	LogReadURL       string                          `jsonapi:"attr,log-read-url"`
	Status           CostEstimationStatus            `jsonapi:"attr,status"`
	StatusTimestamps *CostEstimationStatusTimestamps `jsonapi:"attr,status-timestamps"`
	ErrorMessage     string                          `jsonapi:"attr,error-message"`
	// Costs            *CostEstimationCosts            `jsonapi:"attr,costs"`
}

// CostEstimationStatusTimestamps holds the timestamps for individual costEstimation statuses.
type CostEstimationStatusTimestamps struct {
	CanceledAt time.Time `json:"canceled-at"`
	ErroredAt  time.Time `json:"errored-at"`
	FinishedAt time.Time `json:"finished-at"`
	QueuedAt   time.Time `json:"queued-at"`
}

// Read a costEstimation by its ID.
func (s *costEstimations) Read(ctx context.Context, costEstimationID string) (*CostEstimation, error) {
	if !validStringID(&costEstimationID) {
		return nil, errors.New("invalid value for cost estimation ID")
	}

	u := fmt.Sprintf("cost-estimations/%s", url.QueryEscape(costEstimationID))
	req, err := s.client.newRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	p := &CostEstimation{}
	err = s.client.do(ctx, req, p)
	if err != nil {
		return nil, err
	}

	return p, nil
}

// Logs retrieves the logs of a costEstimation.
func (s *costEstimations) Logs(ctx context.Context, costEstimationID string) (io.Reader, error) {
	if !validStringID(&costEstimationID) {
		return nil, errors.New("invalid value for cost estimation ID")
	}

	// Get the costEstimation to make sure it exists.
	p, err := s.Read(ctx, costEstimationID)
	if err != nil {
		return nil, err
	}

	// Return an error if the log URL is empty.
	if p.LogReadURL == "" {
		return nil, fmt.Errorf("cost estimation %s does not have a log URL", costEstimationID)
	}

	u, err := url.Parse(p.LogReadURL)
	if err != nil {
		return nil, fmt.Errorf("invalid log URL: %v", err)
	}

	done := func() (bool, error) {
		p, err := s.Read(ctx, p.ID)
		if err != nil {
			return false, err
		}

		switch p.Status {
		case CostEstimationCanceled, CostEstimationErrored, CostEstimationFinished:
			return true, nil
		default:
			return false, nil
		}
	}

	return &LogReader{
		client: s.client,
		ctx:    ctx,
		done:   done,
		logURL: u,
	}, nil
}
