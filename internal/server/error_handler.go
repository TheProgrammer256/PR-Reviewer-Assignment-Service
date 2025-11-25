package server

import (
	"errors"
	"net/http"

	openapi "github.com/TheProgrammer256/PR-Reviewer-Assignment-Service/go"

	"github.com/avito/pr-reviewer-assignment-service/internal/apperr"
)

// ErrorHandler renders APIError instances as JSON envelopes while delegating
// parsing/validation issues to the default handler.
func ErrorHandler(w http.ResponseWriter, r *http.Request, err error, result *openapi.ImplResponse) {
	var apiErr *apperr.APIError
	if errors.As(err, &apiErr) {
		status := apiErr.Status
		_ = openapi.EncodeJSONResponse(apiErr.Response(), &status, w)
		return
	}

	openapi.DefaultErrorHandler(w, r, err, result)
}

