package tfe

import (
	"context"
	"net/http"
)

// Compile-time proof of interface implementation.
var _ RunTasksCallback = (*taskResultsCallback)(nil)

// RunTasksCallback describes all the Run Tasks Integration Callback API methods.
//
// TFE API docs:
// https://developer.hashicorp.com/terraform/enterprise/api-docs/run-tasks/run-tasks-integration
type RunTasksCallback interface {
	// Update sends updates to TFC/E Run Task Callback URL
	Update(ctx context.Context, callbackURL string, accessToken string, options TaskResultCallbackRequestOptions) error
}

// taskResultsCallback implements RunTasksCallback.
type taskResultsCallback struct {
	client *Client
}

const (
	TaskResultsCallbackType = "task-results"
)

// Update sends updates to TFC/E Run Task Callback URL
func (s *taskResultsCallback) Update(ctx context.Context, callbackURL string, accessToken string, options TaskResultCallbackRequestOptions) error {
	if !validString(&callbackURL) {
		return ErrInvalidCallbackURL
	}
	if !validString(&accessToken) {
		return ErrInvalidAccessToken
	}
	req, err := s.client.NewRequest(http.MethodPatch, callbackURL, &options)
	if err != nil {
		return err
	}
	// The PATCH request must use the token supplied in the originating request (access_token) for authentication.
	// https://developer.hashicorp.com/terraform/enterprise/api-docs/run-tasks/run-tasks-integration#request-headers-1
	req.Header.Set("Authorization", "Bearer "+accessToken)
	return req.Do(ctx, nil)
}

// TaskResultCallbackRequestOptions represents the TFC/E Task result callback request
// https://developer.hashicorp.com/terraform/enterprise/api-docs/run-tasks/run-tasks-integration#request-body-1
type TaskResultCallbackRequestOptions struct {
	Type     string               `jsonapi:"primary,task-results"`
	Status   TaskResultStatus     `jsonapi:"attr,status"`
	Message  string               `jsonapi:"attr,message,omitempty"`
	URL      string               `jsonapi:"attr,url,omitempty"`
	Outcomes []*TaskResultOutcome `jsonapi:"relation,outcomes,omitempty"`
}

// TaskResultOutcome represents a detailed TFC/E run task outcome, which improves result visibility and content in the TFC/E UI.
// https://developer.hashicorp.com/terraform/enterprise/api-docs/run-tasks/run-tasks-integration#outcomes-payload-body
type TaskResultOutcome struct {
	Type        string                      `jsonapi:"primary,task-result-outcomes"`
	OutcomeID   string                      `jsonapi:"attr,outcome-id,omitempty"`
	Description string                      `jsonapi:"attr,description,omitempty"`
	Body        string                      `jsonapi:"attr,body,omitempty"`
	URL         string                      `jsonapi:"attr,url,omitempty"`
	Tags        map[string][]*TaskResultTag `jsonapi:"attr,tags,omitempty"`
}

// TaskResultTag can be used to enrich outcomes display list in TFC/E.
// https://developer.hashicorp.com/terraform/enterprise/api-docs/run-tasks/run-tasks-integration#severity-and-status-tags
type TaskResultTag struct {
	Label string  `json:"label"`
	Level *string `json:"level,omitempty"`
}

func (o *TaskResultCallbackRequestOptions) valid() error {
	if !validStringID(&o.Type) || o.Type != TaskResultsCallbackType {
		return ErrInvalidTaskResultsCallbackType
	}
	if !validStringID(String(string(o.Status))) || (o.Status != TaskFailed && o.Status != TaskPassed && o.Status != TaskRunning) {
		return ErrInvalidTaskResultsCallbackStatus
	}
	return nil
}
