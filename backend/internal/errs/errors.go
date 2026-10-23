// Package errs defines the application error model for NENAV.
//
// All application errors are typed so that transport layers can map them
// to appropriate HTTP status codes without leaking internal details.
// Domain and service code returns these errors; handlers translate them.
//
// Reference: Master Prompt §9 — Error Model
package errs

import (
	"errors"
	"fmt"
	"net/http"
)

// Code represents an application-level error classification.
type Code string

const (
	CodeValidation      Code = "validation_error"
	CodeAuthentication  Code = "authentication_error"
	CodeAuthorization   Code = "authorization_error"
	CodeNotFound        Code = "not_found"
	CodeConflict        Code = "conflict"
	CodeRateLimited     Code = "rate_limited"
	CodeBusinessRule    Code = "business_rule_violation"
	CodeDependency      Code = "dependency_failure"
	CodeInternal        Code = "internal_error"
	CodeTimeout         Code = "timeout"
	CodeIdempotencyHit  Code = "idempotency_hit"
)

// Error is the standard application error. It carries a machine-readable code,
// a human-readable message safe for API clients, and an optional wrapped cause
// that is logged but never exposed.
type Error struct {
	code    Code
	message string
	cause   error
	fields  map[string]string // validation field errors
}

// New creates an application error.
func New(code Code, message string) *Error {
	return &Error{code: code, message: message}
}

// Wrap creates an application error that wraps a lower-level cause.
// The cause is used for logging and debugging but never exposed to clients.
func Wrap(code Code, message string, cause error) *Error {
	return &Error{code: code, message: message, cause: cause}
}

// Validation creates a validation error with field-level details.
func Validation(message string, fields map[string]string) *Error {
	return &Error{
		code:    CodeValidation,
		message: message,
		fields:  fields,
	}
}

// Error implements the error interface.
func (e *Error) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.code, e.message, e.cause)
	}
	return fmt.Sprintf("%s: %s", e.code, e.message)
}

// Unwrap supports errors.Is and errors.As.
func (e *Error) Unwrap() error {
	return e.cause
}

// Code returns the error classification.
func (e *Error) Code() Code { return e.code }

// Message returns the client-safe message.
func (e *Error) Message() string { return e.message }

// Fields returns validation field errors (may be nil).
func (e *Error) Fields() map[string]string { return e.fields }

// HTTPStatus maps the error code to an HTTP status code.
func (e *Error) HTTPStatus() int {
	switch e.code {
	case CodeValidation:
		return http.StatusBadRequest
	case CodeAuthentication:
		return http.StatusUnauthorized
	case CodeAuthorization:
		return http.StatusForbidden
	case CodeNotFound:
		return http.StatusNotFound
	case CodeConflict:
		return http.StatusConflict
	case CodeRateLimited:
		return http.StatusTooManyRequests
	case CodeBusinessRule:
		return http.StatusUnprocessableEntity
	case CodeTimeout:
		return http.StatusGatewayTimeout
	case CodeIdempotencyHit:
		return http.StatusOK // idempotent replay returns the original result
	case CodeDependency:
		return http.StatusBadGateway
	default:
		return http.StatusInternalServerError
	}
}

// --- Helper constructors for common errors ---

func NotFound(entity, id string) *Error {
	return New(CodeNotFound, fmt.Sprintf("%s %s not found", entity, id))
}

func Unauthorized(message string) *Error {
	return New(CodeAuthentication, message)
}

func Forbidden(message string) *Error {
	return New(CodeAuthorization, message)
}

func Conflict(message string) *Error {
	return New(CodeConflict, message)
}

func Internal(message string, cause error) *Error {
	return Wrap(CodeInternal, message, cause)
}

func BusinessRule(message string) *Error {
	return New(CodeBusinessRule, message)
}

func DependencyFailure(service string, cause error) *Error {
	return Wrap(CodeDependency, fmt.Sprintf("%s unavailable", service), cause)
}

// --- Extraction helpers ---

// AsAppError extracts an *Error from an error chain.
// Returns nil if the error is not an application error.
func AsAppError(err error) *Error {
	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr
	}
	return nil
}

// IsCode checks if an error in the chain has the given application error code.
func IsCode(err error, code Code) bool {
	appErr := AsAppError(err)
	return appErr != nil && appErr.code == code
}
