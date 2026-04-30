// Copyright IBM Corp. 2018, 2026
// SPDX-License-Identifier: MPL-2.0

package tfe

// SCIMResource groups the SCIM related resources together.
// This struct should be constructed with keyed fields only or obtained via the client
// to prevent breakages when new fields are added.
type SCIMResource struct {
	SCIMSettings
	Tokens            AdminSCIMTokens
	Groups            AdminSCIMGroups
	SCIMGroupMappings AdminSCIMGroupMappings
}

// AdminSettings describes all the admin settings related methods that the Terraform Enterprise API supports.
// Note that admin settings are only available in Terraform Enterprise.
//
// TFE API docs: https://developer.hashicorp.com/terraform/enterprise/api-docs/admin/settings
type AdminSettings struct {
	General        GeneralSettings
	SAML           SAMLSettings
	CostEstimation CostEstimationSettings
	SMTP           SMTPSettings
	Twilio         TwilioSettings
	Customization  CustomizationSettings
	OIDC           OIDCSettings
	SCIM           *SCIMResource
}

func newAdminSettings(client *Client) *AdminSettings {
	return &AdminSettings{
		General:        &adminGeneralSettings{client: client},
		SAML:           &adminSAMLSettings{client: client},
		CostEstimation: &adminCostEstimationSettings{client: client},
		SMTP:           &adminSMTPSettings{client: client},
		Twilio:         &adminTwilioSettings{client: client},
		Customization:  &adminCustomizationSettings{client: client},
		OIDC:           &adminOIDCSettings{client: client},
		SCIM: &SCIMResource{
			SCIMSettings:      &adminSCIMSettings{client: client},
			Tokens:            &adminSCIMTokens{client: client},
			Groups:            &adminSCIMGroups{client: client},
			SCIMGroupMappings: &adminSCIMGroupMappings{client: client},
		},
	}
}
