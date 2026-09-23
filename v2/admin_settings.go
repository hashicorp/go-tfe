// Copyright IBM Corp. 2018, 2026
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-tfe/v2/api/models"
)

// Compile-time proof of interface implementation.
var (
	_ AdminCustomizationSettings = (*adminCustomizationSettings)(nil)
	_ AdminGeneralSettings       = (*adminGeneralSettings)(nil)
)

// AdminSettings groups the site administration settings resources. It is
// nested under Client.Admin (see the Admin type) so that other admin resources
// can be grouped alongside it.
//
// TFE API docs: https://developer.hashicorp.com/terraform/enterprise/api-docs/admin/settings
type AdminSettings struct {
	Customization AdminCustomizationSettings
	General       AdminGeneralSettings
}

// AdminAvatarSource configures a single avatar source. Currently "gravatar" is
// the only valid Source; "initials" is the implicit fallback and is never
// returned in the catalog.
type AdminAvatarSource struct {
	Source  string
	Enabled bool
}

// AdminCustomizationSetting represents the admin customization settings.
//
// The Internal* URL and AvatarSources fields are only meaningful on Terraform
// Enterprise; on HCP Terraform they are zero-valued.
type AdminCustomizationSetting struct {
	ID           string
	SupportEmail string
	SupportURL   string
	LoginHelp    string
	Footer       string
	Error        string
	NewUser      string

	InternalSupportURL       string
	InternalDocumentationURL string
	InternalTutorialsURL     string
	AvatarSources            []AdminAvatarSource
}

// AdminCustomizationSettingsUpdateOptions represents the options for updating
// the admin customization settings. A nil pointer leaves the attribute
// unchanged; setting a URL to an empty string clears the override so the
// default link is restored.
type AdminCustomizationSettingsUpdateOptions struct {
	SupportEmail *string
	SupportURL   *string
	LoginHelp    *string
	Footer       *string
	Error        *string
	NewUser      *string

	InternalSupportURL       *string
	InternalDocumentationURL *string
	InternalTutorialsURL     *string
	// AvatarSources, when non-nil, fully replaces the catalog; a non-nil empty
	// slice clears all configured sources.
	AvatarSources []AdminAvatarSource
}

// AdminCustomizationSettings describes the admin customization settings.
type AdminCustomizationSettings interface {
	// Read returns the customization settings.
	Read(ctx context.Context) (*AdminCustomizationSetting, error)

	// Update updates the customization settings.
	Update(ctx context.Context, options AdminCustomizationSettingsUpdateOptions) (*AdminCustomizationSetting, error)
}

type adminCustomizationSettings struct {
	client *Client
}

// AdminGeneralSetting represents the admin general settings.
//
// The general settings singleton in the API exposes many attributes, but this
// client intentionally surfaces only EnablePublicRegistration, the field this
// resource was added to support. Additional fields can be added here as needed.
type AdminGeneralSetting struct {
	ID                       string
	EnablePublicRegistration bool
}

// AdminGeneralSettingsUpdateOptions represents the options for updating the
// admin general settings. A nil pointer leaves the attribute unchanged. Only
// EnablePublicRegistration is currently supported; see AdminGeneralSetting.
type AdminGeneralSettingsUpdateOptions struct {
	EnablePublicRegistration *bool
}

// AdminGeneralSettings describes the admin general settings.
type AdminGeneralSettings interface {
	// Read returns the general settings.
	Read(ctx context.Context) (*AdminGeneralSetting, error)

	// Update updates the general settings.
	Update(ctx context.Context, options AdminGeneralSettingsUpdateOptions) (*AdminGeneralSetting, error)
}

type adminGeneralSettings struct {
	client *Client
}

// Read returns the customization settings.
func (a *adminCustomizationSettings) Read(ctx context.Context) (*AdminCustomizationSetting, error) {
	envelope, err := a.client.API.Admin().CustomizationSettings().Get(ctx, nil)
	if err != nil {
		return nil, err
	}

	return customizationSettingFromEnvelope(envelope), nil
}

// Update updates the customization settings.
func (a *adminCustomizationSettings) Update(ctx context.Context, options AdminCustomizationSettingsUpdateOptions) (*AdminCustomizationSetting, error) {
	attrs := models.NewAdminCustomizationSettings_attributes()
	if options.SupportEmail != nil {
		attrs.SetSupportEmailAddress(options.SupportEmail)
	}
	if options.SupportURL != nil {
		attrs.SetSupportUrlAddress(options.SupportURL)
	}
	if options.LoginHelp != nil {
		attrs.SetLoginHelp(options.LoginHelp)
	}
	if options.Footer != nil {
		attrs.SetFooter(options.Footer)
	}
	if options.Error != nil {
		attrs.SetError(options.Error)
	}
	if options.NewUser != nil {
		attrs.SetNewUser(options.NewUser)
	}
	if options.InternalSupportURL != nil {
		attrs.SetInternalSupportUrl(options.InternalSupportURL)
	}
	if options.InternalDocumentationURL != nil {
		attrs.SetInternalDocumentationUrl(options.InternalDocumentationURL)
	}
	if options.InternalTutorialsURL != nil {
		attrs.SetInternalTutorialsUrl(options.InternalTutorialsURL)
	}
	if options.AvatarSources != nil {
		sources, err := avatarSourcesToModel(options.AvatarSources)
		if err != nil {
			return nil, err
		}
		attrs.SetAvatarSources(sources)
	}

	data := models.NewAdminCustomizationSettings()
	settingsType := models.CUSTOMIZATIONSETTINGS_ADMINCUSTOMIZATIONSETTINGS_TYPE
	data.SetTypeEscaped(&settingsType)
	data.SetAttributes(attrs)

	envelope := models.NewAdminCustomizationSettingsEnvelope()
	envelope.SetData(data)

	result, err := a.client.API.Admin().CustomizationSettings().Patch(ctx, envelope, nil)
	if err != nil {
		return nil, err
	}

	return customizationSettingFromEnvelope(result), nil
}

// Read returns the general settings.
func (a *adminGeneralSettings) Read(ctx context.Context) (*AdminGeneralSetting, error) {
	envelope, err := a.client.API.Admin().GeneralSettings().Get(ctx, nil)
	if err != nil {
		return nil, err
	}

	return generalSettingFromEnvelope(envelope), nil
}

// Update updates the general settings.
func (a *adminGeneralSettings) Update(ctx context.Context, options AdminGeneralSettingsUpdateOptions) (*AdminGeneralSetting, error) {
	attrs := models.NewAdminGeneralSettings_attributes()
	if options.EnablePublicRegistration != nil {
		attrs.SetEnablePublicRegistration(options.EnablePublicRegistration)
	}

	data := models.NewAdminGeneralSettings()
	settingsType := models.GENERALSETTINGS_ADMINGENERALSETTINGS_TYPE
	data.SetTypeEscaped(&settingsType)
	data.SetAttributes(attrs)

	envelope := models.NewAdminGeneralSettingsEnvelope()
	envelope.SetData(data)

	result, err := a.client.API.Admin().GeneralSettings().Patch(ctx, envelope, nil)
	if err != nil {
		return nil, err
	}

	return generalSettingFromEnvelope(result), nil
}

// customizationSettingFromEnvelope maps the wire envelope to the public struct.
func customizationSettingFromEnvelope(envelope models.AdminCustomizationSettingsEnvelopeable) *AdminCustomizationSetting {
	setting := &AdminCustomizationSetting{}
	if envelope == nil {
		return setting
	}

	data := envelope.GetData()
	if data == nil {
		return setting
	}

	if id := data.GetId(); id != nil {
		setting.ID = *id
	}

	attrs := data.GetAttributes()
	if attrs == nil {
		return setting
	}

	setting.SupportEmail = derefString(attrs.GetSupportEmailAddress())
	setting.SupportURL = derefString(attrs.GetSupportUrlAddress())
	setting.LoginHelp = derefString(attrs.GetLoginHelp())
	setting.Footer = derefString(attrs.GetFooter())
	setting.Error = derefString(attrs.GetError())
	setting.NewUser = derefString(attrs.GetNewUser())
	setting.InternalSupportURL = derefString(attrs.GetInternalSupportUrl())
	setting.InternalDocumentationURL = derefString(attrs.GetInternalDocumentationUrl())
	setting.InternalTutorialsURL = derefString(attrs.GetInternalTutorialsUrl())
	setting.AvatarSources = avatarSourcesFromModel(attrs.GetAvatarSources())

	return setting
}

// generalSettingFromEnvelope maps the wire envelope to the public struct.
func generalSettingFromEnvelope(envelope models.AdminGeneralSettingsEnvelopeable) *AdminGeneralSetting {
	setting := &AdminGeneralSetting{}
	if envelope == nil {
		return setting
	}

	data := envelope.GetData()
	if data == nil {
		return setting
	}

	if id := data.GetId(); id != nil {
		setting.ID = *id
	}

	attrs := data.GetAttributes()
	if attrs == nil {
		return setting
	}

	if v := attrs.GetEnablePublicRegistration(); v != nil {
		setting.EnablePublicRegistration = *v
	}

	return setting
}

// avatarSourcesFromModel maps the wire avatar catalog to the public slice.
func avatarSourcesFromModel(sources []models.AdminCustomizationSettings_attributes_avatarSourcesable) []AdminAvatarSource {
	if sources == nil {
		return nil
	}

	result := make([]AdminAvatarSource, 0, len(sources))
	for _, source := range sources {
		if source == nil {
			continue
		}

		entry := AdminAvatarSource{}
		if s := source.GetSource(); s != nil {
			entry.Source = s.String()
		}
		if e := source.GetEnabled(); e != nil {
			entry.Enabled = *e
		}
		result = append(result, entry)
	}

	return result
}

// avatarSourcesToModel maps the public slice to the wire avatar catalog.
//
// Atlas treats the avatar catalog as a full replacement and validates the whole
// set, rejecting any unknown source. To surface that error before issuing the
// PATCH (and to avoid silently turning an invalid-only input into an empty
// catalog, which Atlas would accept as "disable every source"), an unrecognized
// Source is reported as an error rather than skipped.
func avatarSourcesToModel(sources []AdminAvatarSource) ([]models.AdminCustomizationSettings_attributes_avatarSourcesable, error) {
	result := make([]models.AdminCustomizationSettings_attributes_avatarSourcesable, 0, len(sources))
	for _, source := range sources {
		parsed, err := models.ParseAdminCustomizationSettings_attributes_avatarSources_source(source.Source)
		if err != nil {
			return nil, fmt.Errorf("invalid avatar source %q: %w", source.Source, err)
		}
		v, ok := parsed.(*models.AdminCustomizationSettings_attributes_avatarSources_source)
		if !ok || v == nil {
			return nil, fmt.Errorf("invalid avatar source %q", source.Source)
		}

		entry := models.NewAdminCustomizationSettings_attributes_avatarSources()
		entry.SetSource(v)
		enabled := source.Enabled
		entry.SetEnabled(&enabled)
		result = append(result, entry)
	}

	return result, nil
}

// derefString returns the value of s, or "" when s is nil.
func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
