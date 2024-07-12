package tfe

import (
	"context"
	"fmt"
	"net/url"
)

type StackConfigurations interface {
	// ReadConfiguration returns a stack configuration by its ID.
	Read(ctx context.Context, ID string) (*StackConfiguration, error)
}

type stackConfigurations struct {
	client *Client
}

var _ StackConfigurations = &stackConfigurations{}

func (s stackConfigurations) Read(ctx context.Context, ID string) (*StackConfiguration, error) {
	req, err := s.client.NewRequest("GET", fmt.Sprintf("stack-configurations/%s", url.PathEscape(ID)), nil)
	if err != nil {
		return nil, err
	}

	stackConfiguration := &StackConfiguration{}
	err = req.Do(ctx, stackConfiguration)
	if err != nil {
		return nil, err
	}

	return stackConfiguration, nil
}
