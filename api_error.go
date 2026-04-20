package tfe

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type APIError struct {
	StatusCode int
	Message    string
	Details    []string
}

type JSONAPIError struct {
	Errors []struct {
		Status string `json:"status"`
		Title  string `json:"title"`
		Detail string `json:"detail"`
	} `json:"errors"`
}

var (
	// ErrUnauthorized, ErrNotFound, ErrInternalServer, ErrForbidden, ErrBadRequest, and
	// ErrUnprocessableEntity are common API errors that can be used with errors.Is for
	// easy comparison.
	ErrUnauthorized        = &APIError{StatusCode: http.StatusUnauthorized, Message: http.StatusText(http.StatusUnauthorized)}
	ErrNotFound            = &APIError{StatusCode: http.StatusNotFound, Message: http.StatusText(http.StatusNotFound)}
	ErrInternalServer      = &APIError{StatusCode: http.StatusInternalServerError, Message: http.StatusText(http.StatusInternalServerError)}
	ErrForbidden           = &APIError{StatusCode: http.StatusForbidden, Message: http.StatusText(http.StatusForbidden)}
	ErrBadRequest          = &APIError{StatusCode: http.StatusBadRequest, Message: http.StatusText(http.StatusBadRequest)}
	ErrUnprocessableEntity = &APIError{StatusCode: http.StatusUnprocessableEntity, Message: http.StatusText(http.StatusUnprocessableEntity)}
	ErrTooManyRequests     = &APIError{StatusCode: http.StatusTooManyRequests, Message: http.StatusText(http.StatusTooManyRequests)}
)

// Error implements the error interface for APIError.
func (e *APIError) Error() string {
	return fmt.Sprintf("%d %s", e.StatusCode, e.Message)
}

// Is allows errors.Is to work with APIError, comparing based on StatusCode.
func (e *APIError) Is(target error) bool {
	t, ok := target.(*APIError)
	if !ok {
		return false
	}
	return e.StatusCode == t.StatusCode
}

func newSimpleAPIError(statusCode int) *APIError {
	return &APIError{
		StatusCode: statusCode,
		Message:    http.StatusText(statusCode),
	}
}

func newErrorFromErrors(statusCode int, details JSONAPIError) *APIError {
	result := newSimpleAPIError(statusCode)
	for _, err := range details.Errors {
		detail := err.Detail
		if detail != "" {
			result.Details = append(result.Details, fmt.Sprintf("%s: %s", err.Title, detail))
		} else {
			result.Details = append(result.Details, fmt.Sprintf("%s: %s", err.Status, err.Title))
		}
	}
	return result
}

// APIErrorFactory is a function that takes an HTTP response and a pipeline error, and returns
// an APIError. The response must have a status code of 400 or above.
func APIErrorFactory(resp *http.Response, pipelineErr error) error {
	if resp.StatusCode < 400 {
		return fmt.Errorf("status code %d is not an error. This is always a bug in the go-tfe package", resp.StatusCode)
	}

	// Some responses contain error details in the body
	var jsonErr JSONAPIError
	if err := json.NewDecoder(resp.Body).Decode(&jsonErr); err != nil {
		return newSimpleAPIError(resp.StatusCode)
	}
	return newErrorFromErrors(resp.StatusCode, jsonErr)
}
