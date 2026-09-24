// Copyright IBM Corp. 2018, 2026
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/go-tfe/v2/api/models"
)

// Compile-time proof of interface implementation.
var _ AdminBanners = (*adminBanners)(nil)

// AdminBanners describes the admin banners API.
//
// TFE API docs: https://developer.hashicorp.com/terraform/enterprise/api-docs/admin/banners
type AdminBanners interface {
	// List returns all non-deactivated banners. The admin endpoint does not
	// filter by publish window, so scheduled and expired-but-not-deactivated
	// banners are included. Returns an empty slice (not an error) when no
	// non-deactivated banner exists.
	List(ctx context.Context) ([]*AdminBanner, error)

	// Create creates a new site-wide announcement banner. Title, Body, Style,
	// and Audience are required. To publish immediately omit the schedule
	// fields; to schedule provide ScheduledPublishAt, ScheduledExpireAt, and
	// Timezone together (all-or-nothing is enforced by the API).
	Create(ctx context.Context, options AdminBannerCreateOptions) (*AdminBanner, error)

	// Update partially updates an existing banner. Only non-nil fields are
	// sent; nil fields are left unchanged on the server.
	Update(ctx context.Context, bannerID string, options AdminBannerUpdateOptions) (*AdminBanner, error)

	// Delete deactivates the banner. The row is retained server-side and will
	// no longer appear in subsequent List calls.
	Delete(ctx context.Context, bannerID string) error
}

// AdminBanner represents a site-wide announcement banner.
type AdminBanner struct {
	// ID is the external identifier of the banner (e.g. ab-xxxxxxxxxxxxxxxx).
	// Read-only: set by the server on creation.
	ID string

	Title    string
	Body     string
	Style    string
	Audience string
	Timezone string

	// ScheduledPublishAt is the UTC time at which the banner becomes active.
	// Nil for immediately-published banners.
	ScheduledPublishAt *time.Time

	// ScheduledExpireAt is the UTC time at which the banner is automatically
	// deactivated. Nil for immediately-published banners.
	ScheduledExpireAt *time.Time

	// PublishedAt is set by the server the first time the banner becomes
	// immediately active. Read-only.
	PublishedAt *time.Time
}

// AdminBannerCreateOptions represents the options for creating a banner.
//
// Title, Body, Style, and Audience are required.
// ScheduledPublishAt, ScheduledExpireAt, and Timezone must all be provided
// together or all omitted; providing a subset is a validation error.
type AdminBannerCreateOptions struct {
	Title    string
	Body     string
	Style    string
	Audience string

	ScheduledPublishAt *time.Time
	ScheduledExpireAt  *time.Time
	Timezone           *string
}

// AdminBannerUpdateOptions represents the options for updating a banner.
// A nil pointer leaves the field unchanged on the server.
// ScheduledPublishAt, ScheduledExpireAt, and Timezone must all be provided
// together or all omitted; providing a subset is a validation error.
type AdminBannerUpdateOptions struct {
	Title    *string
	Body     *string
	Style    *string
	Audience *string
	Timezone *string

	ScheduledPublishAt *time.Time
	ScheduledExpireAt  *time.Time
}

type adminBanners struct {
	client *Client
}

func (a *adminBanners) List(ctx context.Context) ([]*AdminBanner, error) {
	envelope, err := a.client.API.Admin().Banners().Get(ctx, nil)
	if err != nil {
		return nil, err
	}

	if envelope == nil || envelope.GetData() == nil {
		return []*AdminBanner{}, nil
	}

	banners := make([]*AdminBanner, 0, len(envelope.GetData()))
	for _, item := range envelope.GetData() {
		if item == nil {
			continue
		}
		banners = append(banners, bannerFromData(item))
	}

	return banners, nil
}

// bannerFromData maps an AdminBannersable to an AdminBanner.
func bannerFromData(data models.AdminBannersable) *AdminBanner {
	banner := &AdminBanner{}
	if data == nil {
		return banner
	}

	if id := data.GetId(); id != nil {
		banner.ID = *id
	}

	attrs := data.GetAttributes()
	if attrs == nil {
		return banner
	}

	banner.Title = derefString(attrs.GetTitle())
	banner.Body = derefString(attrs.GetBody())
	banner.Timezone = derefString(attrs.GetTimezone())

	if s := attrs.GetStyle(); s != nil {
		banner.Style = s.String()
	}
	if a := attrs.GetAudience(); a != nil {
		banner.Audience = a.String()
	}

	banner.ScheduledPublishAt = attrs.GetScheduledPublishAt()
	banner.ScheduledExpireAt = attrs.GetScheduledExpireAt()
	banner.PublishedAt = attrs.GetPublishedAt()

	return banner
}

func (a *adminBanners) Create(ctx context.Context, options AdminBannerCreateOptions) (*AdminBanner, error) {
	attrs := models.NewAdminBanners_attributes()

	attrs.SetTitle(&options.Title)
	attrs.SetBody(&options.Body)

	style, err := parseBannerStyle(options.Style)
	if err != nil {
		return nil, err
	}
	attrs.SetStyle(style)

	audience, err := parseBannerAudience(options.Audience)
	if err != nil {
		return nil, err
	}
	attrs.SetAudience(audience)

	if options.ScheduledPublishAt != nil {
		attrs.SetScheduledPublishAt(options.ScheduledPublishAt)
	}
	if options.ScheduledExpireAt != nil {
		attrs.SetScheduledExpireAt(options.ScheduledExpireAt)
	}
	if options.Timezone != nil {
		attrs.SetTimezone(options.Timezone)
	}

	bannerType := models.ADMINBANNERS_ADMINBANNERS_TYPE
	data := models.NewAdminBanners()
	data.SetTypeEscaped(&bannerType)
	data.SetAttributes(attrs)

	envelope := models.NewAdminBannersEnvelope()
	envelope.SetData(data)

	result, err := a.client.API.Admin().Banners().Post(ctx, envelope, nil)
	if err != nil {
		return nil, err
	}

	return bannerFromEnvelope(result), nil
}

// bannerFromEnvelope maps a single-item envelope to an AdminBanner.
func bannerFromEnvelope(envelope models.AdminBannersEnvelopeable) *AdminBanner {
	if envelope == nil {
		return &AdminBanner{}
	}
	return bannerFromData(envelope.GetData())
}

// parseBannerStyle converts a style string to the generated enum type.
// Returns an error for unrecognised values to prevent them being silently
// dropped during Kiota serialisation.
func parseBannerStyle(s string) (*models.AdminBanners_attributes_style, error) {
	parsed, _ := models.ParseAdminBanners_attributes_style(s)
	v, ok := parsed.(*models.AdminBanners_attributes_style)
	if !ok || v == nil {
		return nil, fmt.Errorf("invalid banner style %q: valid values are info, warning, critical", s)
	}
	return v, nil
}

// parseBannerAudience converts an audience string to the generated enum type.
// Returns an error for unrecognised values to prevent them being silently
// dropped during Kiota serialisation.
func parseBannerAudience(s string) (*models.AdminBanners_attributes_audience, error) {
	parsed, _ := models.ParseAdminBanners_attributes_audience(s)
	v, ok := parsed.(*models.AdminBanners_attributes_audience)
	if !ok || v == nil {
		return nil, fmt.Errorf("invalid banner audience %q: valid values are all_users, authenticated_only", s)
	}
	return v, nil
}

func (a *adminBanners) Update(ctx context.Context, bannerID string, options AdminBannerUpdateOptions) (*AdminBanner, error) {
	attrs := models.NewAdminBanners_attributes()

	if options.Title != nil {
		attrs.SetTitle(options.Title)
	}
	if options.Body != nil {
		attrs.SetBody(options.Body)
	}
	if options.Style != nil {
		style, err := parseBannerStyle(*options.Style)
		if err != nil {
			return nil, err
		}
		attrs.SetStyle(style)
	}
	if options.Audience != nil {
		audience, err := parseBannerAudience(*options.Audience)
		if err != nil {
			return nil, err
		}
		attrs.SetAudience(audience)
	}
	if options.Timezone != nil {
		attrs.SetTimezone(options.Timezone)
	}
	if options.ScheduledPublishAt != nil {
		attrs.SetScheduledPublishAt(options.ScheduledPublishAt)
	}
	if options.ScheduledExpireAt != nil {
		attrs.SetScheduledExpireAt(options.ScheduledExpireAt)
	}

	bannerType := models.ADMINBANNERS_ADMINBANNERS_TYPE
	data := models.NewAdminBanners()
	data.SetTypeEscaped(&bannerType)
	data.SetAttributes(attrs)

	envelope := models.NewAdminBannersEnvelope()
	envelope.SetData(data)

	result, err := a.client.API.Admin().Banners().ById(bannerID).Patch(ctx, envelope, nil)
	if err != nil {
		return nil, err
	}

	return bannerFromEnvelope(result), nil
}

func (a *adminBanners) Delete(ctx context.Context, bannerID string) error {
	return a.client.API.Admin().Banners().ById(bannerID).Delete(ctx, nil)
}
