package tfe

import (
	"context"
	"fmt"
	"net/url"
)

// Compile-time proof of interface implementation.
var _ Comments = (*comments)(nil)

// Comments describes all the comment related methods that the
// Terraform Enterprise API supports.
//
// TFE API docs:
// https://www.terraform.io/docs/cloud/api/comments.html
type Comments interface {
	// List all comments of the given run.
	List(ctx context.Context, runID string, options *CommentListOptions) (*CommentList, error)

	// Read a comment by its ID.
	Read(ctx context.Context, CommentID string) (*Comment, error)

	// Create a new comment with the given options.
	Create(ctx context.Context, runID string, options CommentCreateOptions) (*Comment, error)
}

// Comments implements Comments.
type comments struct {
	client *Client
}

// CommentList represents a list of comments.
type CommentList struct {
	*Pagination
	Items []*Comment
}

// Comment represents a Terraform Enterprise comment..
type Comment struct {
	ID   string `jsonapi:"primary,comments"`
	Body string `jsonapi:"attr,body"`
}

// CommentListOptions represents the options for listing comments.
type CommentListOptions struct {
	ListOptions
}

type CommentCreateOptions struct {
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	// It is not a user-defined value and does not need to be set.
	// https://jsonapi.org/format/#crud-creating
	Type string `jsonapi:"primary,comments"`

	// Required: Body of the comment.
	Body *string `jsonapi:"attr,body"`

	// Optional: Run where the comment is attached
	Run *Run `jsonapi:"relation,run"`
}

// List all comments of the given run.
func (s *comments) List(ctx context.Context, runID string, options *CommentListOptions) (*CommentList, error) {
	if !validStringID(&runID) {
		return nil, ErrInvalidRunID
	}
	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("runs/%s/comments", url.QueryEscape(runID))
	req, err := s.client.newRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	cl := &CommentList{}
	err = s.client.do(ctx, req, cl)
	if err != nil {
		return nil, err
	}

	return cl, nil
}

// Create a new comment with the given options.
func (s *comments) Create(ctx context.Context, runID string, options CommentCreateOptions) (*Comment, error) {
	if !validStringID(&runID) {
		return nil, ErrInvalidRunID
	}

	if !validString(options.Body) {
		return nil, ErrCommentBody
	}

	u := fmt.Sprintf("runs/%s/comments", url.QueryEscape(runID))
	req, err := s.client.newRequest("POST", u, &options)
	if err != nil {
		return nil, err
	}

	comm := &Comment{}
	err = s.client.do(ctx, req, comm)
	if err != nil {
		return nil, err
	}

	return comm, err
}

// Read a comment by its ID.
func (s *comments) Read(ctx context.Context, CommentID string) (*Comment, error) {
	if !validStringID(&CommentID) {
		return nil, ErrInvalidCommentID
	}

	u := fmt.Sprintf("comments/%s", url.QueryEscape(CommentID))
	req, err := s.client.newRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	comm := &Comment{}
	err = s.client.do(ctx, req, comm)
	if err != nil {
		return nil, err
	}

	return comm, nil
}

func (o *CommentListOptions) valid() error {
	if o == nil {
		return nil // nothing to validate
	}

	return nil
}
