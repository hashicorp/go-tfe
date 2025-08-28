package tfe

import (
	"context"
	"fmt"
	"net/url"
)

type StackConfigurationSummaries interface {
	//List lists all the stack configuration summaries for a stack.
	List(ctx context.Context, stackID string) (*StackConfigurationSummaryList, error)
}

type stackConfigurationSummaries struct {
	client *Client
}

var _ StackConfigurationSummaries = &stackConfigurationSummaries{}

type StackConfigurationSummaryList struct {
	*Pagination
	Items []*StackConfigurationSummary
}

type StackConfigurationSummary struct {
	ID             string
	Type           string
	Status         string
	SequenceNumber int
}

func (s stackConfigurationSummaries) List(ctx context.Context, stackID string) (*StackConfigurationSummaryList, error) {
	if !validStringID(&stackID) {
		return nil, fmt.Errorf("invalid stack ID: %s", stackID)
	}

	req, err := s.client.NewRequest("GET", fmt.Sprintf("stacks/%s/stack-configuration-summaries", url.PathEscape(stackID)), nil)
	if err != nil {
		return nil, err
	}

	scl := &StackConfigurationSummaryList{}
	err = req.Do(ctx, scl)
	if err != nil {
		return nil, err
	}

	return scl, nil
}
