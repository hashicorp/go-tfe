package tfe

import (
	"context"
)

// Compile-time proof of interface implementation.
var _ GeneralSettings = (*adminGeneralSettings)(nil)

// GeneralSettings describes the general admin settings.
type GeneralSettings interface {
	// Read returns the general settings
	Read(ctx context.Context) (*AdminGeneralSetting, error)

	// Update updates general settings.
	Update(ctx context.Context, options AdminGeneralSettingsUpdateOptions) (*AdminGeneralSetting, error)
}

type adminGeneralSettings struct {
	client *Client
}

// AdminGeneralSetting represents a the general settings in Terraform Enterprise.
type AdminGeneralSetting struct {
	ID                            string `jsonapi:"primary,general-settings"`
	LimitUserOrganizationCreation bool   `jsonapi:"attr,limit-user-organization-creation"`
	APIRateLimitingEnabled        bool   `jsonapi:"attr,api-rate-limiting-enabled"`
	APIRateLimit                  int    `jsonapi:"attr,api-rate-limit"`
	SendPassingStatusesEnabled    bool   `jsonapi:"attr,send-passing-statuses-for-untriggered-speculative-plans"`
	AllowSpeculativePlansOnPR     bool   `jsonapi:"attr,allow-speculative-plans-on-pull-requests-from-forks"`
}

// Read returns the general settings.
func (s *adminGeneralSettings) Read(ctx context.Context) (*AdminGeneralSetting, error) {
	req, err := s.client.newRequest("GET", "admin/general-settings", nil)
	if err != nil {
		return nil, err
	}

	ags := &AdminGeneralSetting{}
	err = s.client.do(ctx, req, ags)
	if err != nil {
		return nil, err
	}

	return ags, nil
}

// AdminGeneralSettingsUpdateOptions represents the admin options for updating
// general settings.
// https://www.terraform.io/docs/cloud/api/admin/settings.html#request-body
type AdminGeneralSettingsUpdateOptions struct {
	LimitUserOrgCreation              *bool `jsonapi:"attr,limit-user-organization-creation,omitempty"`
	APIRateLimitingEnabled            *bool `jsonapi:"attr,api-rate-limiting-enabled,omitempty"`
	APIRateLimit                      *int  `jsonapi:"attr,api-rate-limit,omitempty"`
	SendPassingStatusUntriggeredPlans *bool `jsonapi:"attr,send-passing-statuses-for-untriggered-speculative-plans,omitempty"`
	AllowSpeculativePlansOnPR         *bool `jsonapi:"attr,allow-speculative-plans-on-pull-requests-from-forks,omitempty"`
}

// Update updates the general settings.
func (s *adminGeneralSettings) Update(ctx context.Context, options AdminGeneralSettingsUpdateOptions) (*AdminGeneralSetting, error) {
	req, err := s.client.newRequest("PATCH", "admin/general-settings", &options)
	if err != nil {
		return nil, err
	}

	ags := &AdminGeneralSetting{}
	err = s.client.do(ctx, req, ags)
	if err != nil {
		return nil, err
	}

	return ags, nil
}
