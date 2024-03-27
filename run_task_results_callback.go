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
	// Update sends updates to TFC/E Run Task Callback URL..
	Update(ctx context.Context, callbackURL string, accessToken string, options TaskResultsCallbackOptions) error
}

// taskResultsCallback implements RunTasksCallback.
type taskResultsCallback struct {
	client *Client
}

const (
	TaskResultsCallbackType = "task-results"
)

// Update sends updates to TFC/E Run Task Callback URL
func (s *taskResultsCallback) Update(ctx context.Context, callbackURL string, accessToken string, options TaskResultsCallbackOptions) error {
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

// TaskResultsCallbackOptions represents the options for a TFE Task result callback request
// https://developer.hashicorp.com/terraform/enterprise/api-docs/run-tasks/run-tasks-integration#request-body-1
type TaskResultsCallbackOptions struct {
	Data *TaskResultsCallbackData `json:"data"`
}

type TaskResultsCallbackData struct {
	// Required: Must be set to `task-results`
	Type *string `json:"type"`
	// Required: Attributes of the Task Results Callback Response
	Attributes    *TaskResultsCallbackDataAttributes `json:"attributes"`
	Relationships *TaskResultsCallbackRelationships  `json:"relationships,omitempty"`
}

type TaskResultsCallbackDataAttributes struct {
	// Status Must be one of TaskFailed, TaskPassed or TaskRunning
	Status TaskResultStatus `json:"status"`
	// Message A short message describing the status of the task.
	Message string `json:"message,omitempty"`
	// URL that the user can use to get more information from the external service
	URL string `json:"url,omitempty"`
}

type TaskResultsCallbackRelationships struct {
	// Outcomes A run task result may optionally contain one or more detailed outcomes, which improves result visibility and content in the Terraform Cloud user interface.
	// https://developer.hashicorp.com/terraform/enterprise/api-docs/run-tasks/run-tasks-integration#outcomes-payload-body
	Outcomes *TaskResultsCallbackRelationshipsOutcomes `json:"outcomes"`
}

type TaskResultsCallbackRelationshipsOutcomes struct {
	Data []*TaskResultsCallbackRelationshipsOutcomesData `json:"data"`
}

type TaskResultsCallbackRelationshipsOutcomesData struct {
	Type       string                                                  `json:"type"`
	Attributes *TaskResultsCallbackRelationshipsOutcomesDataAttributes `json:"attributes"`
}

type TaskResultsCallbackRelationshipsOutcomesDataAttributes struct {
	OutcomeID   string                                                                   `json:"outcome-id"`
	Description string                                                                   `json:"description"`
	Body        string                                                                   `json:"body,omitempty"`
	URL         string                                                                   `json:"url,omitempty"`
	Tags        map[string][]*TaskResultsCallbackRelationshipsOutcomesDataTagsAttributes `json:"tags,omitempty"`
}

// TaskResultsCallbackRelationshipsOutcomesDataTagsAttributes can be used to enrich outcomes display list in TFC/E.
// https://developer.hashicorp.com/terraform/enterprise/api-docs/run-tasks/run-tasks-integration#severity-and-status-tags
type TaskResultsCallbackRelationshipsOutcomesDataTagsAttributes struct {
	Label string `json:"label"`
	Level string `json:"level,omitempty"`
}

func (o *TaskResultsCallbackOptions) valid() error {
	if o.Data == nil {
		return ErrRequiredCallbackData
	}
	if validStringID(o.Data.Type) && o.Data.Type != String(TaskResultsCallbackType) {
		return ErrInvalidTaskResultsCallbackType
	}
	if o.Data.Attributes == nil {
		return ErrRequiredCallbackDataAttributes
	}
	if o.Data.Attributes.Status != TaskFailed || o.Data.Attributes.Status != TaskPassed || o.Data.Attributes.Status != TaskRunning {
		return ErrInvalidTaskResultsCallbackStatus
	}
	return nil
}
