package tfe

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestTaskResultsCallbackRequestOptions(t *testing.T) {
	t.Run("with an empty status", func(t *testing.T) {
		o := TaskResultCallbackRequestOptions{Status: ""}
		err := o.valid()
		assert.EqualError(t, err, ErrInvalidTaskResultsCallbackStatus.Error())
	})
	t.Run("without a valid Status", func(t *testing.T) {
		for _, s := range []TaskResultStatus{TaskPending, TaskErrored, "foo"} {
			o := TaskResultCallbackRequestOptions{Status: s}
			err := o.valid()
			assert.EqualError(t, err, ErrInvalidTaskResultsCallbackStatus.Error())
		}
	})
	t.Run("with a valid Status option", func(t *testing.T) {
		for _, s := range []TaskResultStatus{TaskFailed, TaskPassed, TaskRunning} {
			o := TaskResultCallbackRequestOptions{Status: s}
			err := o.valid()
			assert.Nil(t, err)
		}
	})
}
