// Copyright IBM Corp. 2018, 2026
// SPDX-License-Identifier: MPL-2.0

package testserver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	tfe "github.com/hashicorp/go-tfe"
	"github.com/hashicorp/jsonapi"
)

const (
	defaultToken      = "mock-token"
	defaultAPIVersion = "2.0"
	defaultAppName    = "HCP Terraform"
	defaultRateLimit  = "30"
)

// Option configures a test API server.
type Option func(*Server)

// WithToken sets the bearer token that the server expects.
// If token is empty, authentication checks are disabled.
func WithToken(token string) Option {
	return func(s *Server) {
		s.token = token
	}
}

// WithAPIVersion sets the version returned in TFP-API-Version.
func WithAPIVersion(version string) Option {
	return func(s *Server) {
		s.apiVersion = version
	}
}

// WithAppName sets the value returned in TFP-AppName.
func WithAppName(name string) Option {
	return func(s *Server) {
		s.appName = name
	}
}

// Server is an in-memory HTTP implementation of a small subset of the
// Terraform API surface for local, deployment-free testing.
type Server struct {
	mu sync.RWMutex

	token                    string
	apiVersion               string
	appName                  string
	rateLimit                string
	tfeVersion               string
	tfeCurrentNumericVersion string

	idCounter uint64

	organizations           map[string]*organizationRecord
	workspacesByID          map[string]*workspaceRecord
	workspaceIDByOrgAndName map[string]map[string]string

	httpServer *httptest.Server
}

type organizationRecord struct {
	Name                  string
	Email                 string
	CreatedAt             time.Time
	CostEstimationEnabled bool
	DefaultExecutionMode  string
}

type workspaceRecord struct {
	ID               string
	OrganizationName string
	Name             string
	Description      string
	CreatedAt        time.Time
	UpdatedAt        time.Time
	AutoApply        bool
	ExecutionMode    string
	TerraformVersion string
}

// New creates and starts a steel-thread local API server.
func New(opts ...Option) *Server {
	s := &Server{
		token:                   defaultToken,
		apiVersion:              defaultAPIVersion,
		appName:                 defaultAppName,
		rateLimit:               defaultRateLimit,
		organizations:           make(map[string]*organizationRecord),
		workspacesByID:          make(map[string]*workspaceRecord),
		workspaceIDByOrgAndName: make(map[string]map[string]string),
	}

	for _, opt := range opts {
		opt(s)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handle)
	s.httpServer = httptest.NewServer(mux)

	return s
}

// Close shuts down the underlying HTTP server.
func (s *Server) Close() {
	if s == nil || s.httpServer == nil {
		return
	}
	s.httpServer.Close()
}

// URL returns the base URL for the test server.
func (s *Server) URL() string {
	if s == nil || s.httpServer == nil {
		return ""
	}
	return s.httpServer.URL
}

// Client returns an HTTP client preconfigured for this server.
func (s *Server) Client() *http.Client {
	if s == nil || s.httpServer == nil {
		return http.DefaultClient
	}
	return s.httpServer.Client()
}

// Token returns the configured bearer token.
func (s *Server) Token() string {
	return s.token
}

// ClientConfig returns a go-tfe client config wired to this server.
func (s *Server) ClientConfig() *tfe.Config {
	return &tfe.Config{
		Address:    s.URL(),
		Token:      s.token,
		HTTPClient: s.Client(),
	}
}

// SeedOrganization inserts an organization directly into in-memory state.
func (s *Server) SeedOrganization(name, email string) *tfe.Organization {
	now := time.Now().UTC()

	s.mu.Lock()
	defer s.mu.Unlock()

	s.organizations[name] = &organizationRecord{
		Name:                 name,
		Email:                email,
		CreatedAt:            now,
		DefaultExecutionMode: "remote",
	}

	return s.organizationToModel(s.organizations[name])
}

// SeedWorkspace inserts a workspace directly into in-memory state.
func (s *Server) SeedWorkspace(organization, name string) (*tfe.Workspace, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.organizations[organization]; !ok {
		return nil, fmt.Errorf("organization %q not found", organization)
	}

	if s.workspaceIDByOrgAndName[organization] == nil {
		s.workspaceIDByOrgAndName[organization] = make(map[string]string)
	}
	if _, exists := s.workspaceIDByOrgAndName[organization][name]; exists {
		return nil, fmt.Errorf("workspace %q already exists", name)
	}

	now := time.Now().UTC()
	id := s.nextID("ws")
	record := &workspaceRecord{
		ID:               id,
		OrganizationName: organization,
		Name:             name,
		CreatedAt:        now,
		UpdatedAt:        now,
		ExecutionMode:    "remote",
	}

	s.workspacesByID[id] = record
	s.workspaceIDByOrgAndName[organization][name] = id

	return s.workspaceToModel(record), nil
}

func (s *Server) handle(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		s.handleRoot(w, r)
		return
	}

	route, ok := trimAPIPrefix(r.URL.Path)
	if !ok {
		s.writeError(w, http.StatusNotFound, "Not Found", "endpoint not implemented")
		return
	}

	if !s.isAuthorized(r) {
		s.writeError(w, http.StatusUnauthorized, "Unauthorized", "invalid or missing bearer token")
		return
	}

	parts, err := decodePathParts(route)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "Bad Request", err.Error())
		return
	}

	if len(parts) == 1 && parts[0] == "ping" {
		s.handlePing(w, r)
		return
	}

	if len(parts) == 1 && parts[0] == "organizations" {
		s.handleOrganizations(w, r)
		return
	}

	if len(parts) == 2 && parts[0] == "organizations" {
		s.handleOrganization(w, r, parts[1])
		return
	}

	if len(parts) == 3 && parts[0] == "organizations" && parts[2] == "workspaces" {
		s.handleOrganizationWorkspaces(w, r, parts[1])
		return
	}

	if len(parts) == 4 && parts[0] == "organizations" && parts[2] == "workspaces" {
		s.handleOrganizationWorkspace(w, r, parts[1], parts[3])
		return
	}

	if len(parts) == 2 && parts[0] == "workspaces" {
		s.handleWorkspaceByID(w, r, parts[1])
		return
	}

	s.writeError(w, http.StatusNotFound, "Not Found", "endpoint not implemented")
}

func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeMethodNotAllowed(w, http.MethodGet)
		return
	}

	s.setMetadataHeaders(w)
	w.Header().Set("Content-Type", "application/json")

	_ = json.NewEncoder(w).Encode(map[string]any{
		"name":     "go-tfe steel-thread test server",
		"api_base": "/api/v2",
		"token":    s.Token(),
		"supported_endpoints": []string{
			"GET /api/v2/ping",
			"GET|POST /api/v2/organizations",
			"GET|PATCH|DELETE /api/v2/organizations/{organization}",
			"GET|POST /api/v2/organizations/{organization}/workspaces",
			"GET|PATCH|DELETE /api/v2/organizations/{organization}/workspaces/{workspace}",
			"GET|PATCH|DELETE /api/v2/workspaces/{workspace_id}",
		},
	})
}

func (s *Server) handlePing(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeMethodNotAllowed(w, http.MethodGet)
		return
	}

	s.setMetadataHeaders(w)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleOrganizations(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.mu.RLock()
		records := make([]*organizationRecord, 0, len(s.organizations))
		for _, org := range s.organizations {
			records = append(records, org)
		}
		s.mu.RUnlock()

		sort.Slice(records, func(i, j int) bool { return records[i].Name < records[j].Name })

		items := make([]*tfe.Organization, 0, len(records))
		for _, record := range records {
			items = append(items, s.organizationToModel(record))
		}

		s.writeJSONAPIList(w, http.StatusOK, items)
	case http.MethodPost:
		options, err := decodeJSONAPIRequest[tfe.OrganizationCreateOptions](r)
		if err != nil {
			s.writeError(w, http.StatusBadRequest, "Bad Request", err.Error())
			return
		}
		if options.Name == nil || strings.TrimSpace(*options.Name) == "" {
			s.writeError(w, http.StatusBadRequest, "Bad Request", "name is required")
			return
		}
		if options.Email == nil || strings.TrimSpace(*options.Email) == "" {
			s.writeError(w, http.StatusBadRequest, "Bad Request", "email is required")
			return
		}

		now := time.Now().UTC()
		name := *options.Name

		s.mu.Lock()
		defer s.mu.Unlock()

		if _, exists := s.organizations[name]; exists {
			s.writeError(w, http.StatusConflict, "Conflict", "organization already exists")
			return
		}

		record := &organizationRecord{
			Name:                  name,
			Email:                 *options.Email,
			CreatedAt:             now,
			CostEstimationEnabled: derefBool(options.CostEstimationEnabled),
			DefaultExecutionMode:  "remote",
		}
		s.organizations[name] = record

		s.writeJSONAPISingle(w, http.StatusCreated, s.organizationToModel(record))
	default:
		s.writeMethodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func (s *Server) handleOrganization(w http.ResponseWriter, r *http.Request, organization string) {
	switch r.Method {
	case http.MethodGet:
		s.mu.RLock()
		record, ok := s.organizations[organization]
		s.mu.RUnlock()
		if !ok {
			s.writeError(w, http.StatusNotFound, "Not Found", "organization not found")
			return
		}

		s.writeJSONAPISingle(w, http.StatusOK, s.organizationToModel(record))
	case http.MethodPatch:
		options, err := decodeJSONAPIRequest[tfe.OrganizationUpdateOptions](r)
		if err != nil {
			s.writeError(w, http.StatusBadRequest, "Bad Request", err.Error())
			return
		}

		s.mu.Lock()
		defer s.mu.Unlock()

		record, ok := s.organizations[organization]
		if !ok {
			s.writeError(w, http.StatusNotFound, "Not Found", "organization not found")
			return
		}

		newName := organization
		if options.Name != nil && strings.TrimSpace(*options.Name) != "" {
			newName = *options.Name
			if newName != organization {
				if _, exists := s.organizations[newName]; exists {
					s.writeError(w, http.StatusConflict, "Conflict", "organization already exists")
					return
				}
			}
		}

		if options.Email != nil {
			record.Email = *options.Email
		}
		if options.CostEstimationEnabled != nil {
			record.CostEstimationEnabled = *options.CostEstimationEnabled
		}
		if options.DefaultExecutionMode != nil {
			record.DefaultExecutionMode = *options.DefaultExecutionMode
		}

		if newName != organization {
			delete(s.organizations, organization)
			record.Name = newName
			s.organizations[newName] = record

			wsByName := s.workspaceIDByOrgAndName[organization]
			if wsByName != nil {
				delete(s.workspaceIDByOrgAndName, organization)
				s.workspaceIDByOrgAndName[newName] = wsByName
				for _, workspaceID := range wsByName {
					if ws, exists := s.workspacesByID[workspaceID]; exists {
						ws.OrganizationName = newName
					}
				}
			}
		}

		s.writeJSONAPISingle(w, http.StatusOK, s.organizationToModel(record))
	case http.MethodDelete:
		s.mu.Lock()
		defer s.mu.Unlock()

		if _, ok := s.organizations[organization]; !ok {
			s.writeError(w, http.StatusNotFound, "Not Found", "organization not found")
			return
		}

		if wsByName, exists := s.workspaceIDByOrgAndName[organization]; exists {
			for _, workspaceID := range wsByName {
				delete(s.workspacesByID, workspaceID)
			}
			delete(s.workspaceIDByOrgAndName, organization)
		}

		delete(s.organizations, organization)

		s.setMetadataHeaders(w)
		w.WriteHeader(http.StatusNoContent)
	default:
		s.writeMethodNotAllowed(w, http.MethodGet, http.MethodPatch, http.MethodDelete)
	}
}

func (s *Server) handleOrganizationWorkspaces(w http.ResponseWriter, r *http.Request, organization string) {
	s.mu.RLock()
	_, orgExists := s.organizations[organization]
	s.mu.RUnlock()
	if !orgExists {
		s.writeError(w, http.StatusNotFound, "Not Found", "organization not found")
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.mu.RLock()
		wsByName := s.workspaceIDByOrgAndName[organization]
		records := make([]*workspaceRecord, 0, len(wsByName))
		for _, workspaceID := range wsByName {
			records = append(records, s.workspacesByID[workspaceID])
		}
		s.mu.RUnlock()

		sort.Slice(records, func(i, j int) bool { return records[i].Name < records[j].Name })

		searchName := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("search[name]")))
		items := make([]*tfe.Workspace, 0, len(records))
		for _, record := range records {
			if searchName != "" && !strings.Contains(strings.ToLower(record.Name), searchName) {
				continue
			}
			items = append(items, s.workspaceToModel(record))
		}

		s.writeJSONAPIList(w, http.StatusOK, items)
	case http.MethodPost:
		options, err := decodeJSONAPIRequest[tfe.WorkspaceCreateOptions](r)
		if err != nil {
			s.writeError(w, http.StatusBadRequest, "Bad Request", err.Error())
			return
		}

		if options.Name == nil || strings.TrimSpace(*options.Name) == "" {
			s.writeError(w, http.StatusBadRequest, "Bad Request", "name is required")
			return
		}

		name := *options.Name
		now := time.Now().UTC()

		s.mu.Lock()
		defer s.mu.Unlock()

		if s.workspaceIDByOrgAndName[organization] == nil {
			s.workspaceIDByOrgAndName[organization] = make(map[string]string)
		}
		if _, exists := s.workspaceIDByOrgAndName[organization][name]; exists {
			s.writeError(w, http.StatusConflict, "Conflict", "workspace already exists")
			return
		}

		record := &workspaceRecord{
			ID:               s.nextID("ws"),
			OrganizationName: organization,
			Name:             name,
			Description:      derefString(options.Description),
			CreatedAt:        now,
			UpdatedAt:        now,
			AutoApply:        derefBool(options.AutoApply),
			ExecutionMode:    defaultString(options.ExecutionMode, "remote"),
			TerraformVersion: derefString(options.TerraformVersion),
		}

		s.workspacesByID[record.ID] = record
		s.workspaceIDByOrgAndName[organization][record.Name] = record.ID

		s.writeJSONAPISingle(w, http.StatusCreated, s.workspaceToModel(record))
	default:
		s.writeMethodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func (s *Server) handleOrganizationWorkspace(w http.ResponseWriter, r *http.Request, organization, workspace string) {
	s.mu.RLock()
	workspaceID, ok := s.workspaceIDByOrgAndName[organization][workspace]
	if !ok {
		s.mu.RUnlock()
		s.writeError(w, http.StatusNotFound, "Not Found", "workspace not found")
		return
	}
	record := s.workspacesByID[workspaceID]
	s.mu.RUnlock()

	s.handleWorkspace(w, r, record)
}

func (s *Server) handleWorkspaceByID(w http.ResponseWriter, r *http.Request, workspaceID string) {
	s.mu.RLock()
	record, ok := s.workspacesByID[workspaceID]
	s.mu.RUnlock()
	if !ok {
		s.writeError(w, http.StatusNotFound, "Not Found", "workspace not found")
		return
	}

	s.handleWorkspace(w, r, record)
}

func (s *Server) handleWorkspace(w http.ResponseWriter, r *http.Request, workspace *workspaceRecord) {
	switch r.Method {
	case http.MethodGet:
		s.writeJSONAPISingle(w, http.StatusOK, s.workspaceToModel(workspace))
	case http.MethodPatch:
		options, err := decodeJSONAPIRequest[tfe.WorkspaceUpdateOptions](r)
		if err != nil {
			s.writeError(w, http.StatusBadRequest, "Bad Request", err.Error())
			return
		}

		s.mu.Lock()
		defer s.mu.Unlock()

		fresh, ok := s.workspacesByID[workspace.ID]
		if !ok {
			s.writeError(w, http.StatusNotFound, "Not Found", "workspace not found")
			return
		}

		if options.Name != nil && strings.TrimSpace(*options.Name) != "" && *options.Name != fresh.Name {
			if s.workspaceIDByOrgAndName[fresh.OrganizationName] == nil {
				s.workspaceIDByOrgAndName[fresh.OrganizationName] = make(map[string]string)
			}
			if _, exists := s.workspaceIDByOrgAndName[fresh.OrganizationName][*options.Name]; exists {
				s.writeError(w, http.StatusConflict, "Conflict", "workspace already exists")
				return
			}

			delete(s.workspaceIDByOrgAndName[fresh.OrganizationName], fresh.Name)
			fresh.Name = *options.Name
			s.workspaceIDByOrgAndName[fresh.OrganizationName][fresh.Name] = fresh.ID
		}

		if options.Description != nil {
			fresh.Description = *options.Description
		}
		if options.AutoApply != nil {
			fresh.AutoApply = *options.AutoApply
		}
		if options.ExecutionMode != nil {
			fresh.ExecutionMode = *options.ExecutionMode
		}
		if options.TerraformVersion != nil {
			fresh.TerraformVersion = *options.TerraformVersion
		}
		fresh.UpdatedAt = time.Now().UTC()

		s.writeJSONAPISingle(w, http.StatusOK, s.workspaceToModel(fresh))
	case http.MethodDelete:
		s.mu.Lock()
		defer s.mu.Unlock()

		fresh, ok := s.workspacesByID[workspace.ID]
		if !ok {
			s.writeError(w, http.StatusNotFound, "Not Found", "workspace not found")
			return
		}

		delete(s.workspacesByID, fresh.ID)
		if wsByName := s.workspaceIDByOrgAndName[fresh.OrganizationName]; wsByName != nil {
			delete(wsByName, fresh.Name)
		}

		s.setMetadataHeaders(w)
		w.WriteHeader(http.StatusNoContent)
	default:
		s.writeMethodNotAllowed(w, http.MethodGet, http.MethodPatch, http.MethodDelete)
	}
}

func (s *Server) organizationToModel(record *organizationRecord) *tfe.Organization {
	if record == nil {
		return nil
	}

	return &tfe.Organization{
		Name:                  record.Name,
		Email:                 record.Email,
		CreatedAt:             record.CreatedAt,
		CostEstimationEnabled: record.CostEstimationEnabled,
		DefaultExecutionMode:  record.DefaultExecutionMode,
	}
}

func (s *Server) workspaceToModel(record *workspaceRecord) *tfe.Workspace {
	if record == nil {
		return nil
	}

	return &tfe.Workspace{
		ID:               record.ID,
		Name:             record.Name,
		Description:      record.Description,
		CreatedAt:        record.CreatedAt,
		UpdatedAt:        record.UpdatedAt,
		AutoApply:        record.AutoApply,
		ExecutionMode:    record.ExecutionMode,
		TerraformVersion: record.TerraformVersion,
		Organization:     &tfe.Organization{Name: record.OrganizationName},
		Source:           tfe.WorkspaceSourceAPI,
	}
}

func (s *Server) isAuthorized(r *http.Request) bool {
	s.mu.RLock()
	token := s.token
	s.mu.RUnlock()

	if token == "" {
		return true
	}

	return r.Header.Get("Authorization") == "Bearer "+token
}

func (s *Server) setMetadataHeaders(w http.ResponseWriter) {
	w.Header().Set("TFP-API-Version", s.apiVersion)
	w.Header().Set("TFP-AppName", s.appName)
	w.Header().Set("X-RateLimit-Limit", s.rateLimit)

	if s.tfeVersion != "" {
		w.Header().Set("X-TFE-Version", s.tfeVersion)
	}
	if s.tfeCurrentNumericVersion != "" {
		w.Header().Set("X-TFE-Current-Version", s.tfeCurrentNumericVersion)
	}
}

func (s *Server) writeJSONAPISingle(w http.ResponseWriter, status int, model any) {
	buf := bytes.NewBuffer(nil)
	if err := jsonapi.MarshalPayloadWithoutIncluded(buf, model); err != nil {
		s.writeError(w, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}

	s.setMetadataHeaders(w)
	w.Header().Set("Content-Type", tfe.ContentTypeJSONAPI)
	w.WriteHeader(status)
	_, _ = w.Write(buf.Bytes())
}

func (s *Server) writeJSONAPIList(w http.ResponseWriter, status int, models any) {
	buf := bytes.NewBuffer(nil)
	if err := jsonapi.MarshalPayload(buf, models); err != nil {
		s.writeError(w, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}

	s.setMetadataHeaders(w)
	w.Header().Set("Content-Type", tfe.ContentTypeJSONAPI)
	w.WriteHeader(status)
	_, _ = w.Write(buf.Bytes())
}

func (s *Server) writeError(w http.ResponseWriter, status int, title, detail string) {
	s.setMetadataHeaders(w)
	w.Header().Set("Content-Type", tfe.ContentTypeJSONAPI)
	w.WriteHeader(status)

	errorEntry := map[string]any{
		"status": strconv.Itoa(status),
		"title":  title,
	}
	if detail != "" {
		errorEntry["detail"] = detail
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"errors": []map[string]any{errorEntry},
	})
}

func (s *Server) writeMethodNotAllowed(w http.ResponseWriter, allowed ...string) {
	if len(allowed) > 0 {
		w.Header().Set("Allow", strings.Join(allowed, ", "))
	}
	s.writeError(w, http.StatusMethodNotAllowed, "Method Not Allowed", "unsupported method for endpoint")
}

func (s *Server) nextID(prefix string) string {
	s.idCounter++
	return fmt.Sprintf("%s-%08d", prefix, s.idCounter)
}

func trimAPIPrefix(path string) (string, bool) {
	const prefix = "/api/v2/"

	if !strings.HasPrefix(path, prefix) {
		return "", false
	}

	route := strings.TrimPrefix(path, prefix)
	route = strings.Trim(route, "/")
	if route == "" {
		return "", false
	}

	return route, true
}

func decodePathParts(route string) ([]string, error) {
	parts := strings.Split(route, "/")
	decoded := make([]string, 0, len(parts))
	for _, part := range parts {
		v, err := url.PathUnescape(part)
		if err != nil {
			return nil, fmt.Errorf("invalid path segment %q: %w", part, err)
		}
		decoded = append(decoded, v)
	}
	return decoded, nil
}

func decodeJSONAPIRequest[T any](r *http.Request) (T, error) {
	var out T
	defer r.Body.Close() //nolint:errcheck

	if err := jsonapi.UnmarshalPayload(r.Body, &out); err != nil {
		return out, err
	}

	return out, nil
}

func derefBool(v *bool) bool {
	if v == nil {
		return false
	}
	return *v
}

func derefString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func defaultString(v *string, fallback string) string {
	if v == nil || *v == "" {
		return fallback
	}
	return *v
}
