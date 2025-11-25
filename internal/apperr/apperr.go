package apperr

import (
	"fmt"

	openapi "github.com/TheProgrammer256/PR-Reviewer-Assignment-Service/go"
)

// APIError is propagated up to the custom error handler so that we can reuse
// the same domain errors across every controller.
type APIError struct {
	Status  int
	Code    string
	Message string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Response serialises the error into the OpenAPI error envelope.
func (e *APIError) Response() openapi.ErrorResponse {
	return openapi.ErrorResponse{
		Error: openapi.ErrorResponseError{
			Code:    e.Code,
			Message: e.Message,
		},
	}
}

// New constructs a new APIError.
func New(status int, code, message string) *APIError {
	return &APIError{
		Status:  status,
		Code:    code,
		Message: message,
	}
}

