// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfe

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// NOTE WELL: This is a beta feature and is subject to change until noted otherwise in the
// release notes.
type StackPlanOperations interface {
	// Read returns a stack plan operation by its ID.
	Read(ctx context.Context, stackPlanOperationID string) (*StackPlanOperation, error)

	// Get Stack Plans from Configuration Version
	DownloadEventStream(ctx context.Context, stackPlanOperationID string) ([]byte, error)
}

type stackPlanOperations struct {
	client *Client
}

var _ StackPlanOperations = &stackPlanOperations{}

type StackPlanOperation struct {
	ID             string `jsonapi:"primary,stack-plan-operations"`
	Type           string `jsonapi:"attr,operation-type"`
	Status         string `jsonapi:"attr,status"`
	EventStreamURL string `jsonapi:"attr,event-stream-url"`
	Diagnostics    []*StackDiagnostic `jsonapi:"attr,diags"`

	// Relations
	StackPlan *StackPlan `jsonapi:"relation,stack-plan"`
}

// StackDiagnostic represents any sourcebundle.Diagnostic value. The simplest form has
// just a severity, single line summary, and optional detail. If there is more
// information about the source of the diagnostic, this is represented in the
// range field.
type StackDiagnostic struct {
	Severity string           `jsonapi:"attr,severity"`
	Summary  string           `jsonapi:"attr,summary"`
	Detail   string           `jsonapi:"attr,detail"`
	Range    *DiagnosticRange `jsonapi:"attr,range"`
}

// DiagnosticPos represents a position in the source code.
type DiagnosticPos struct {
	// Line is a one-based count for the line in the indicated file.
	Line int `jsonapi:"attr,line"`

	// Column is a one-based count of Unicode characters from the start of the line.
	Column int `jsonapi:"attr,column"`

	// Byte is a zero-based offset into the indicated file.
	Byte int `jsonapi:"attr,byte"`
}

// DiagnosticRange represents the filename and position of the diagnostic
// subject. This defines the range of the source to be highlighted in the
// output. Note that the snippet may include additional surrounding source code
// if the diagnostic has a context range.
//
// The stacks-specific source field represents the full source bundle address
// of the file, while the filename field is the sub path relative to its
// enclosing package. This represents an attempt to be somewhat backwards
// compatible with the existing Terraform JSON diagnostic format, where
// filename is root module relative.
//
// The Start position is inclusive, and the End position is exclusive. Exact
// positions are intended for highlighting for human interpretation only and
// are subject to change.
type DiagnosticRange struct {
	Filename string        `jsonapi:"attr,filename"`
	Source   string        `jsonapi:"attr,source"`
	Start    DiagnosticPos `jsonapi:"attr,start"`
	End      DiagnosticPos `jsonapi:"attr,end"`
}

func (s stackPlanOperations) Read(ctx context.Context, stackPlanOperationID string) (*StackPlanOperation, error) {
    req, err := s.client.NewRequest("GET", fmt.Sprintf("stack-plan-operations/%s", url.PathEscape(stackPlanOperationID)), nil)
	if err != nil {
		return nil, err
	}

	spo := &StackPlanOperation{}
	err = req.Do(ctx, spo)
	if err != nil {
		return nil, err
	}

	return spo, nil
}

func (s stackPlanOperations) DownloadEventStream(ctx context.Context, eventStreamURL string) ([]byte, error) {
	// Create a new request.
	req, err := http.NewRequest("GET", eventStreamURL, nil)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)

	// Attach the default headers.
	for k, v := range s.client.headers {
		req.Header[k] = v
	}

	// Retrieve the next chunk.
	resp, err := s.client.http.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Basic response checking.
	if err := checkResponseCode(resp); err != nil {
		return nil, err
	}

	// Read the retrieved chunk.
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return b, nil
}
