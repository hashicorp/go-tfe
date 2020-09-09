package tfe

import (
	"context"
	"errors"
	"fmt"
	"net/url"
)

var _ IngressAttributes = (*ingressAttributes)(nil)

type IngressAttributes interface {
	Read(ctx context.Context, ingressAttributeID string) (*IngressAttribute, error)
}

type ingressAttributes struct {
	client *Client
}

type IngressAttribute struct {
	ID                string `jsonapi:"primary,ingress-attributes"`
	Branch            string `jsonapi:"attr,branch"`
	CloneURL          string `jsonapi:"attr,clone-url"`
	CommitMessage     string `jsonapi:"attr,commit-message"`
	CommitSHA         string `jsonapi:"attr,commit-sha"`
	CommitURL         string `jsonapi:"attr,commit-url"`
	CompareURL        string `jsonapi:"attr,compare-url"`
	Identifier        string `jsonapi:"attr,identifier"`
	IsPullRequest     bool   `jsonapi:"attr,is-pull-request"`
	OnDefaultBranch   bool   `jsonapi:"attr,on-default-branch"`
	PullRequestBody   string `jsonapi:"attr,pull-request-body"`
	PullRequestNumber string `jsonapi:"attr,pull-request-number"`
	PullRequestTitle  string `jsonapi:"attr,pull-request-title"`
	PullRequestURL    string `jsonapi:"attr,pull-request-url"`
	SenderAvatarURL   string `jsonapi:"attr,sender-avatar-url"`
	SenderHTMLURL     string `jsonapi:"attr,sender-html-url"`
	SenderUsername    string `jsonapi:"attr,sender-username"`
	Tag               string `jsonapi:"attr,tag"`
}

func (s *ingressAttributes) Read(ctx context.Context, ingressAttributeID string) (*IngressAttribute, error) {
	if !validStringID(&ingressAttributeID) {
		return nil, errors.New("invalid value for ingress attributes ID")
	}

	u := fmt.Sprintf("ingress-attributes/%s", url.QueryEscape(ingressAttributeID))
	req, err := s.client.newRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	i := &IngressAttribute{}
	err = s.client.do(ctx, req, i)
	if err != nil {
		return nil, err
	}

	return i, nil
}
