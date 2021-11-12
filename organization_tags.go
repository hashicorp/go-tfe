package tfe

import (
	"context"
	"errors"
	"fmt"
	"net/url"
)

var _ OrganizationTags = (*organizationTags)(nil)

type OrganizationTags interface {
	List(ctx context.Context, organization string, options OrganizationTagsListOptions) (*OrganizationTagsList, error)

	Delete(ctx context.Context, organization string, options OrganizationTagsDeleteOptions) error

	AddWorkspaces(ctx context.Context, tag string, options AddWorkspacesToTagOptions) error
}

type organizationTags struct {
	client *Client
}

type OrganizationTagsList struct {
	*Pagination
	Items []*OrganizationTag
}

type OrganizationTag struct {
	ID            string `jsonapi:"primary,tags"`
	Name          string `jsonapi:"attr,name,omitempty"`
	InstanceCount string `jsonapi:"attr,instance_count,omitempty"`

	// Relations
	Organization *Organization `jsonapi:"relation,organization"`
}

type OrganizationTagsListOptions struct {
	ListOptions

	FilterExclude  *string `url:"filter[exclude],omitempty"`
	FilterTaggable *string `url:"filter[taggable],omitempty"`
	FilterId       *string `url:"filter[id],omitempty"`
}

func (s *organizationTags) List(ctx context.Context, organization string, options OrganizationTagsListOptions) (*OrganizationTagsList, error) {
	if !validStringID(&organization) {
		return nil, ErrInvalidOrg
	}

	u := fmt.Sprintf("organizations/%s/tags", url.QueryEscape(organization))
	req, err := s.client.newRequest("GET", u, &options)
	if err != nil {
		return nil, err
	}

	tags := &OrganizationTagsList{}
	err = s.client.do(ctx, req, tags)
	if err != nil {
		return nil, err
	}

	fmt.Println(tags.Items[0])

	return tags, nil
}

type OrganizationTagsDeleteOptions struct {
	IDs []string
}

type tagID struct {
	ID string `jsonapi:primary,tag`
}

func (opts *OrganizationTagsDeleteOptions) valid() error {
	if opts.IDs == nil || len(opts.IDs) == 0 {
		return errors.New("you must specify at least one tag id to remove")
	}

	for _, id := range opts.IDs {
		if !validStringID(&id) {
			errorMsg := fmt.Sprintf("%s is not a valid id value", id)
			return errors.New(errorMsg)
		}
	}

	return nil
}

func (s *organizationTags) Delete(ctx context.Context, organization string, options OrganizationTagsDeleteOptions) error {
	if !validStringID(&organization) {
		return ErrInvalidOrg
	}

	if err := options.valid(); err != nil {
		return err
	}

	u := fmt.Sprintf("organizations/%s/tags", url.QueryEscape(organization))
	var tagsToRemove []*tagID
	for _, id := range options.IDs {
		tagsToRemove = append(tagsToRemove, &tagID{ID: id})
	}

	req, err := s.client.newRequest("DELETE", u, tagsToRemove)
	if err != nil {
		return err
	}

	return s.client.do(ctx, req, nil)
}

type AddWorkspacesToTagOptions struct {
	WorkspaceIDs []string
}

func (w *AddWorkspacesToTagOptions) valid() error {
	if w.WorkspaceIDs == nil || len(w.WorkspaceIDs) == 0 {
		return errors.New("you must specify at least one workspace to add tag to")
	}

	for _, id := range w.WorkspaceIDs {
		if !validStringID(&id) {
			errorMsg := fmt.Sprintf("%s is not a valid id value", id)
			return errors.New(errorMsg)
		}
	}

	return nil
}

type workspaceID struct {
	Id string `jsonapi:primary,workspaces`
}

func (s *organizationTags) AddWorkspaces(ctx context.Context, tag string, options AddWorkspacesToTagOptions) error {
	if !validStringID(&tag) {
		return errors.New("invalid tag id")
	}

	if err := options.valid(); err != nil {
		return err
	}

	u := fmt.Sprintf("tags/%s/relationships/workspaces", url.QueryEscape(tag))
	var workspaces []*workspaceID
	for _, id := range options.WorkspaceIDs {
		workspaces = append(workspaces, &workspaceID{Id: id})
	}

	req, err := s.client.newRequest("POST", u, workspaces)
	if err != nil {
		return err
	}

	return s.client.do(ctx, req, nil)
}
