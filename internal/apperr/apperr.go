package apperr

import (
	"fmt"

	openapi "github.com/TheProgrammer256/PR-Reviewer-Assignment-Service/go"
)

type APIError struct {
	Status  int
	Code    string
	Message string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *APIError) Response() openapi.ErrorResponse {
	return openapi.ErrorResponse{
		Error: openapi.ErrorResponseError{
			Code:    e.Code,
			Message: e.Message,
		},
	}
}

func New(status int, code, message string) *APIError {
	return &APIError{
		Status:  status,
		Code:    code,
		Message: message,
	}
}

