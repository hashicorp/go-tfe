package tfe

import (
	"context"
	"fmt"
	"net/url"
)

type StackDeploymentGroupSummaries interface {
	// List lists all the stack deployment group summaries for a stack.
	List(ctx context.Context, configurationID string, options *StackDeploymentGroupSummaryListOptions) (*StackDeploymentGroupSummaryList, error)
}

type stackDeploymentGroupSummaries struct {
	client *Client
}

var _ StackDeploymentGroupSummaries = &stackDeploymentGroupSummaries{}

type StackDeploymentGroupSummaryList struct {
	*Pagination
	Items []*StackDeploymentGroupSummary
}

type StackDeploymentGroupSummaryListOptions struct {
	ListOptions
}

type StackDeploymentGroupStatusCounts struct {
	Pending                     int `jsonapi:"attr,pending"`
	PreDeploying                int `jsonapi:"attr,pre-deploying"`
	PreDeployingPendingOperator int `jsonapi:"attr,pending-operator"`
	AcquiringLock               int `jsonapi:"attr,acquiring-lock"`
	Deploying                   int `jsonapi:"attr,deploying"`
	Succeeded                   int `jsonapi:"attr,succeeded"`
	Failed                      int `jsonapi:"attr,failed"`
	Abandoned                   int `jsonapi:"attr,abandoned"`
}

type StackDeploymentGroupSummary struct {
	ID string `jsonapi:"primary,stack-deployment-group-summaries"`

	// Attributes
	Name         string                            `jsonapi:"attr,name"`
	Status       string                            `jsonapi:"attr,status"`
	StatusCounts *StackDeploymentGroupStatusCounts `jsonapi:"attr,status-counts"`

	// Relationships
	StackDeploymentGroup *StackDeploymentGroup `jsonapi:"relation,stack-deployment-group"`
}

func (s stackDeploymentGroupSummaries) List(ctx context.Context, stackID string, options *StackDeploymentGroupSummaryListOptions) (*StackDeploymentGroupSummaryList, error) {
	if !validStringID(&stackID) {
		return nil, fmt.Errorf("invalid stack ID: %s", stackID)
	}

	if options == nil {
		options = &StackDeploymentGroupSummaryListOptions{}
	}

	req, err := s.client.NewRequest("GET", fmt.Sprintf("stack-configurations/%s/stack-deployment-group-summaries", url.PathEscape(stackID)), options)
	if err != nil {
		return nil, err
	}

	scl := &StackDeploymentGroupSummaryList{}
	err = req.Do(ctx, scl)
	if err != nil {
		return nil, err
	}

	return scl, nil
}
