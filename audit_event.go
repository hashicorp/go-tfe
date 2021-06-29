package tfe

import (
	"bytes"
	"context"
	"encoding/json"
	"time"
)

// Compile-time proof of interface implementation.
var _ AuditEvents = (*auditEvents)(nil)

// AuditEvents describes all the audit event related methods that the
// Terraform Cloud API supports.
//
// TFE API docs: https://www.terraform.io/docs/cloud/api/audit-trails.html
type AuditEvents interface {
	// List audit events visible to the current user.
	List(ctx context.Context, options AuditEventListOptions) (*AuditEventList, error)
}

// auditEvents implements AuditEvents.
type auditEvents struct {
	client *Client
}

// AuditEvent represents a Terraform Cloud audit event.
type AuditEvent struct {
	ID        string              `json:"id"`
	Version   string              `json:"version"`
	Type      string              `json:"type"`
	Timestamp time.Time           `json:"timestamp,iso8601"`
	Auth      *AuditEventAuth     `json:"auth"`
	Request   *AuditEventRequest  `json:"request"`
	Resource  *AuditEventResource `json:"resource"`
}

// AuditEventList represents a list of audit events.
type AuditEventList struct {
	Pagination struct {
		CurrentPage  int `json:"current_page"`
		PreviousPage int `json:"prev_page"`
		NextPage     int `json:"next_page"`
		TotalPages   int `json:"total_pages"`
		TotalCount   int `json:"total_count"`
	} `json:"pagination"`
	Data []*AuditEvent `json:"data"`
}

// AuditEventAuthType represents tye types of audit event auth.
type AuditEventAuthType string

// List all available audit event auth types.
const (
	AuditEventAuthTypeClient       AuditEventAuthType = "Client"
	AuditEventAuthTypeImpersonated AuditEventAuthType = "Impersonated"
	AuditEventAuthTypeSystem       AuditEventAuthType = "System"
)

// AuditEventAuth holds the auth data.
type AuditEventAuth struct {
	AccessorID     string             `json:"accessor_id"`
	Description    string             `json:"description"`
	Type           AuditEventAuthType `json:"type"`
	ImpersonatorID string             `json:"impersonator_id"`
	OrganizationID string             `json:"organization_id"`
}

// AuditEventRequest holds the request data.
type AuditEventRequest struct {
	ID string `json:"id"`
}

// AuditEventResource holds the resource data.
type AuditEventResource struct {
	ID     string                 `json:"id"`
	Type   string                 `json:"type"`
	Action string                 `json:"action"`
	Meta   map[string]interface{} `json:"meta"`
}

// AuditEventListOptions represents the options for listing audit events.
type AuditEventListOptions struct {
	ListOptions

	// Returns only audit events created after this date.
	Since *time.Time `url:"since,omitempty,iso8601"`
}

// List all the audit events visible to the current user.
func (s *auditEvents) List(ctx context.Context, options AuditEventListOptions) (*AuditEventList, error) {
	req, err := s.client.newRequest("GET", "organization/audit-trail", &options)
	if err != nil {
		return nil, err
	}

	// Use an io.Writer to receive the raw response:
	buffer := bytes.NewBufferString("")
	err = s.client.do(ctx, req, buffer)
	if err != nil {
		return nil, err
	}

	auditEventList := &AuditEventList{}
	if err := json.Unmarshal(buffer.Bytes(), auditEventList); err != nil {
		return nil, err
	}

	return auditEventList, nil
}
