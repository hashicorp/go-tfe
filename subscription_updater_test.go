// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"fmt"
	"net/url"
	"testing"
	"time"
)

type featureSet struct {
	ID string `jsonapi:"primary,feature-sets"`
}

type featureSetList struct {
	Items []*featureSet
	*Pagination
}

type featureSetListOptions struct {
	Q string `url:"q,omitempty"`
}

type retryableFn func() (interface{}, error)

type updateFeatureSetOptions struct {
	Type                          string     `jsonapi:"primary,subscription"`
	RunsCeiling                   *int       `jsonapi:"attr,runs-ceiling,omitempty"`
	AgentsCeiling                 *int       `jsonapi:"attr,agents-ceiling,omitempty"`
	ContractStartAt               *time.Time `jsonapi:"attr,contract-start-at,iso8601,omitempty"`
	ContractUserLimit             *int       `jsonapi:"attr,contract-user-limit,omitempty"`
	ContractApplyLimit            *int       `jsonapi:"attr,contract-apply-limit,omitempty"`
	ContractManagedResourcesLimit *int       `jsonapi:"attr,contract-managed-resources-limit,omitempty"`

	FeatureSet *featureSet `jsonapi:"relation,feature-set"`
}

type organizationSubscriptionUpdater struct {
	organization *Organization
	planName     string
	updateOpts   updateFeatureSetOptions
}

func newSubscriptionUpdater(organization *Organization) *organizationSubscriptionUpdater {
	return &organizationSubscriptionUpdater{
		organization: organization,
		updateOpts:   updateFeatureSetOptions{},
	}
}

func (b *organizationSubscriptionUpdater) WithBusinessPlan() *organizationSubscriptionUpdater {
	b.planName = "Business"

	ceiling := 10
	start := time.Now()
	userLimit := 1000
	applyLimit := 5000

	b.updateOpts.RunsCeiling = &ceiling
	b.updateOpts.ContractStartAt = &start
	b.updateOpts.ContractUserLimit = &userLimit
	b.updateOpts.ContractApplyLimit = &applyLimit
	return b
}

func (b *organizationSubscriptionUpdater) WithTrialPlan() *organizationSubscriptionUpdater {
	b.planName = "Trial"
	ceiling := 1
	b.updateOpts.RunsCeiling = &ceiling
	return b
}

func (b *organizationSubscriptionUpdater) WithStandardEntitlementPlan() *organizationSubscriptionUpdater {
	b.planName = "Standard (entitlement)"

	start := time.Now()
	ceiling := 1
	managedResourcesLimit := 1000

	b.updateOpts.ContractStartAt = &start
	b.updateOpts.RunsCeiling = &ceiling
	b.updateOpts.ContractManagedResourcesLimit = &managedResourcesLimit
	return b
}

func (b *organizationSubscriptionUpdater) WithPremiumPlan() *organizationSubscriptionUpdater {
	b.planName = "Premium (entitlement)"

	ceiling := 10
	start := time.Now()
	managedResourcesLimit := 1000

	b.updateOpts.RunsCeiling = &ceiling
	b.updateOpts.AgentsCeiling = &ceiling
	b.updateOpts.ContractStartAt = &start
	b.updateOpts.ContractManagedResourcesLimit = &managedResourcesLimit
	return b
}

// Attempts to change an organization's subscription to a different plan. Requires a user token with admin access.
func (b *organizationSubscriptionUpdater) Update(t *testing.T) {
	if enterpriseEnabled() {
		t.Skip("Cannot upgrade an organization's subscription when enterprise is enabled. Set ENABLE_TFE=0 to run.")
	}

	if b.planName == "" {
		t.Fatal("organizationSubscriptionUpdater requires a plan")
		return
	}

	adminClient := testAdminClient(t, provisionLicensesAdmin)
	req, err := adminClient.NewRequest("GET", "admin/feature-sets", featureSetListOptions{
		Q: b.planName,
	})
	if err != nil {
		t.Fatal(err)
		return
	}

	fsl := &featureSetList{}
	err = req.Do(context.Background(), fsl)
	if err != nil {
		t.Fatalf("failed to enumerate feature sets: %v", err)
		return
	} else if len(fsl.Items) == 0 {
		t.Fatalf("feature set response was empty")
		return
	}

	b.updateOpts.FeatureSet = fsl.Items[0]

	u := fmt.Sprintf("admin/organizations/%s/subscription", url.PathEscape(b.organization.Name))
	req, err = adminClient.NewRequest("POST", u, &b.updateOpts)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
		return
	}

	err = req.Do(context.Background(), nil)
	if err != nil {
		t.Fatalf("Failed to upgrade subscription: %v", err)
	}
}
