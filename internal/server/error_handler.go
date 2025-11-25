package server

import (
	"errors"
	"net/http"

	openapi "github.com/TheProgrammer256/PR-Reviewer-Assignment-Service/go"

	"github.com/avito/pr-reviewer-assignment-service/internal/apperr"
)

func ErrorHandler(w http.ResponseWriter, r *http.Request, err error, result *openapi.ImplResponse) {
	var apiErr *apperr.APIError
	if errors.As(err, &apiErr) {
		status := apiErr.Status
		_ = openapi.EncodeJSONResponse(apiErr.Response(), &status, w)
		return
	}

	openapi.DefaultErrorHandler(w, r, err, result)
}

