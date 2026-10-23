// Package response provides consistent JSON response helpers for HTTP handlers.
//
// All API responses use this package to ensure uniform structure.
// Error responses never expose internal details — they use the application
// error model from the errs package.
//
// Reference: Master Prompt §9 — Error Model, §11 — API Design
package response

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/nenav/nenav/backend/internal/errs"
)

// Envelope wraps all successful responses.
type Envelope struct {
	Data any    `json:"data,omitempty"`
	Meta *Meta  `json:"meta,omitempty"`
}

// Meta contains pagination and other response metadata.
type Meta struct {
	Page       int `json:"page,omitempty"`
	PageSize   int `json:"page_size,omitempty"`
	TotalCount int `json:"total_count,omitempty"`
	TotalPages int `json:"total_pages,omitempty"`
}

// ErrorBody is the standard error response structure.
type ErrorBody struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail contains error information safe for API clients.
type ErrorDetail struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

// JSON writes a successful JSON response.
func JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		json.NewEncoder(w).Encode(Envelope{Data: data})
	}
}

// JSONWithMeta writes a successful JSON response with pagination metadata.
func JSONWithMeta(w http.ResponseWriter, status int, data any, meta *Meta) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(Envelope{Data: data, Meta: meta})
}

// Err writes an error response. It uses the application error model to
// determine the HTTP status code and constructs a safe error body.
// Internal errors are logged but never exposed to clients.
func Err(w http.ResponseWriter, r *http.Request, logger *slog.Logger, err error) {
	appErr := errs.AsAppError(err)
	if appErr == nil {
		// Unknown error — treat as internal and log it
		logger.Error("unhandled_error",
			"error", err.Error(),
			"path", r.URL.Path,
			"method", r.Method,
		)
		appErr = errs.Internal("an unexpected error occurred", err)
	}

	status := appErr.HTTPStatus()

	// Log server errors at error level, client errors at warn
	if status >= 500 {
		logger.Error("server_error",
			"code", string(appErr.Code()),
			"message", appErr.Message(),
			"error", appErr.Error(),
			"path", r.URL.Path,
		)
	} else if status >= 400 {
		logger.Warn("client_error",
			"code", string(appErr.Code()),
			"message", appErr.Message(),
			"path", r.URL.Path,
		)
	}

	body := ErrorBody{
		Error: ErrorDetail{
			Code:    string(appErr.Code()),
			Message: appErr.Message(),
			Fields:  appErr.Fields(),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}

// NoContent writes a 204 No Content response.
func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// Created writes a 201 Created response with the given data.
func Created(w http.ResponseWriter, data any) {
	JSON(w, http.StatusCreated, data)
}
