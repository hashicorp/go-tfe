package tfe

import (
	"bytes"
	"context"
	"encoding/json"
	"time"
)

// Compile-time proof of interface implementation.
var _ AuditTrails = (*auditTrails)(nil)

// AuditTrails describes all the audit trail related methods that the
// Terraform Cloud API supports.
//
// TFE API docs: https://www.terraform.io/docs/cloud/api/audit-trails.html
type AuditTrails interface {
	// List audit trails visible to the current user.
	List(ctx context.Context, options AuditTrailListOptions) (*AuditTrailList, error)
}

// auditTrails implements AuditTrails.
type auditTrails struct {
	client *Client
}

// AuditTrail represents a Terraform Cloud audit trail.
type AuditTrail struct {
	ID        string              `json:"id"`
	Version   string              `json:"version"`
	Type      string              `json:"type"`
	Timestamp time.Time           `json:"timestamp,iso8601"`
	Auth      *AuditTrailAuth     `json:"auth"`
	Request   *AuditTrailRequest  `json:"request"`
	Resource  *AuditTrailResource `json:"resource"`
}

// AuditTrailList represents a list of audit trails.
type AuditTrailList struct {
	*Pagination `json:"pagination"`
	Data        []*AuditTrail `json:"data"`
}

// AuditTrailAuthType represents tye types of audit trail auth.
type AuditTrailAuthType string

// List all available audit trail auth types.
const (
	AuditTrailAuthTypeClient       AuditTrailAuthType = "Client"
	AuditTrailAuthTypeImpersonated AuditTrailAuthType = "Impersonated"
	AuditTrailAuthTypeSystem       AuditTrailAuthType = "System"
)

// AuditTrailAuth holds the auth data.
type AuditTrailAuth struct {
	AccessorID     string             `json:"accessor_id"`
	Description    string             `json:"description"`
	Type           AuditTrailAuthType `json:"type"`
	ImpersonatorID string             `json:"impersonator_id"`
	OrganizationID string             `json:"organization_id"`
}

// AuditTrailRequest holds the request data.
type AuditTrailRequest struct {
	ID string `json:"id"`
}

// AuditTrailResource holds the resource data.
type AuditTrailResource struct {
	ID     string                 `json:"id"`
	Type   string                 `json:"type"`
	Action string                 `json:"action"`
	Meta   map[string]interface{} `json:"meta"`
}

// AuditTrailListOptions represents the options for listing audit trails.
type AuditTrailListOptions struct {
	ListOptions

	// Returns only audit trails created after this date.
	Since *time.Time `url:"since,omitempty,iso8601"`
}

// List all the audit trails visible to the current user.
func (s *auditTrails) List(ctx context.Context, options AuditTrailListOptions) (*AuditTrailList, error) {
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

	auditTrailList := &AuditTrailList{}
	if err := json.Unmarshal(buffer.Bytes(), auditTrailList); err != nil {
		return nil, err
	}

	return auditTrailList, nil
}
