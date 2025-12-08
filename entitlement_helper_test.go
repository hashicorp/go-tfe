// Copyright IBM Corp. 2018, 2025
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"errors"
)

func getOrgEntitlements(client *Client, organizationName string) (*Entitlements, error) {
	ctx := context.Background()
	orgEntitlements, err := client.Organizations.ReadEntitlements(ctx, organizationName)
	if err != nil {
		return nil, err
	}
	if orgEntitlements == nil {
		return nil, errors.New("The organization entitlements are empty.")
	}
	return orgEntitlements, nil
}

func hasGlobalRunTasks(client *Client, organizationName string) (bool, error) {
	oe, err := getOrgEntitlements(client, organizationName)
	if err != nil {
		return false, err
	}
	return oe.GlobalRunTasks, nil
}

func hasPrivateRunTasks(client *Client, organizationName string) (bool, error) {
	oe, err := getOrgEntitlements(client, organizationName)
	if err != nil {
		return false, err
	}
	return oe.PrivateRunTasks, nil
}

func hasAuditLogging(client *Client, organizationName string) (bool, error) {
	oe, err := getOrgEntitlements(client, organizationName)
	if err != nil {
		return false, err
	}
	return oe.AuditLogging, nil
}
