package tfe

import (
	"bytes"
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

// TestTaskResultsCallbackRequestOptions_Validate runs a series of tests that test whether various TaskResultCallbackRequestOptions objects can be considered valid or not
func TestTaskResultsCallbackRequestOptions_Validate(t *testing.T) {
	t.Run("with an empty status", func(t *testing.T) {
		opts := TaskResultCallbackRequestOptions{Status: ""}
		err := opts.valid()
		assert.EqualError(t, err, ErrInvalidTaskResultsCallbackStatus.Error())
	})
	t.Run("without valid Status options", func(t *testing.T) {
		for _, s := range []TaskResultStatus{TaskPending, TaskErrored, "foo"} {
			opts := TaskResultCallbackRequestOptions{Status: s}
			err := opts.valid()
			assert.EqualError(t, err, ErrInvalidTaskResultsCallbackStatus.Error())
		}
	})
	t.Run("with valid Status options", func(t *testing.T) {
		for _, s := range []TaskResultStatus{TaskFailed, TaskPassed, TaskRunning} {
			opts := TaskResultCallbackRequestOptions{Status: s}
			err := opts.valid()
			require.NoError(t, err)
		}
	})
}

// TestTaskResultsCallbackRequestOptions_Marshal tests whether you can properly serialise a TaskResultCallbackRequestOptions object
// You may find the expected body here: https://developer.hashicorp.com/terraform/enterprise/api-docs/run-tasks/run-tasks-integration#request-body-1
func TestTaskResultsCallbackRequestOptions_Marshal(t *testing.T) {
	opts := TaskResultCallbackRequestOptions{
		Status:  TaskPassed,
		Message: "4 passed, 0 skipped, 0 failed",
		URL:     "https://external.service.dev/terraform-plan-checker/run-i3Df5to9ELvibKpQ",
		Outcomes: []*TaskResultOutcome{
			{
				OutcomeID:   "PRTNR-CC-TF-127",
				Description: "ST-2942:S3 Bucket will not enforce MFA login on delete requests",
				Body:        "# Resolution for issue ST-2942\n\n## Impact\n\nFollow instructions in the [AWS S3 docs](https://docs.aws.amazon.com/AmazonS3/latest/userguide/MultiFactorAuthenticationDelete.html) to manually configure the MFA setting.\n—-- Payload truncated —--",
				URL:         "https://external.service.dev/result/PRTNR-CC-TF-127",
				Tags: map[string][]*TaskResultTag{
					"Status": {&TaskResultTag{Label: "Denied", Level: "error"}},
					"Severity": {
						&TaskResultTag{Label: "High", Level: "error"},
						&TaskResultTag{Label: "Recoverable", Level: "info"},
					},
					"Cost Centre": {&TaskResultTag{Label: "IT-OPS"}},
				},
			},
		},
	}
	require.NoError(t, opts.valid())
	reqBody, err := serializeRequestBody(&opts)
	require.NoError(t, err)
	expectedBody := `{"data":{"type":"task-results","attributes":{"message":"4 passed, 0 skipped, 0 failed","status":"passed","url":"https://external.service.dev/terraform-plan-checker/run-i3Df5to9ELvibKpQ"},"relationships":{"outcomes":{"data":[{"type":"task-result-outcomes","attributes":{"body":"# Resolution for issue ST-2942\n\n## Impact\n\nFollow instructions in the [AWS S3 docs](https://docs.aws.amazon.com/AmazonS3/latest/userguide/MultiFactorAuthenticationDelete.html) to manually configure the MFA setting.\n—-- Payload truncated —--","description":"ST-2942:S3 Bucket will not enforce MFA login on delete requests","outcome-id":"PRTNR-CC-TF-127","tags":{"Cost Centre":[{"label":"IT-OPS"}],"Severity":[{"label":"High","level":"error"},{"label":"Recoverable","level":"info"}],"Status":[{"label":"Denied","level":"error"}]},"url":"https://external.service.dev/result/PRTNR-CC-TF-127"}}]}}}}
`
	assert.Equal(t, reqBody.(*bytes.Buffer).String(), expectedBody)
}

func TestTaskResultsCallbackUpdate_Validate(t *testing.T) {
	t.Run("with invalid callbackURL", func(t *testing.T) {
		trc := taskResultsCallback{client: nil}
		err := trc.Update(context.Background(), "", "", TaskResultCallbackRequestOptions{})
		assert.EqualError(t, err, ErrInvalidCallbackURL.Error())
	})
	t.Run("with invalid accessToken", func(t *testing.T) {
		trc := taskResultsCallback{client: nil}
		err := trc.Update(context.Background(), "https://app.terraform.io/foo", "", TaskResultCallbackRequestOptions{})
		assert.EqualError(t, err, ErrInvalidAccessToken.Error())
	})
}

func TestTaskResultsCallbackUpdate(t *testing.T) {
	ts := runTaskCallbackMockServer(t)
	defer ts.Close()

	client, err := NewClient(&Config{
		RetryServerErrors: true,
		Token:             testInitialClientToken,
		Address:           ts.URL,
	})
	require.NoError(t, err)
	trc := taskResultsCallback{
		client: client,
	}
	req := RunTaskRequest{
		AccessToken:           testTaskResultCallbackToken,
		TaskResultCallbackURL: ts.URL,
	}
	err = trc.Update(context.Background(), req.TaskResultCallbackURL, req.AccessToken, TaskResultCallbackRequestOptions{Status: TaskPassed})
	require.NoError(t, err)
}
