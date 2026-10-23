// Package middleware provides HTTP middleware for the NENAV API.
//
// Each middleware is in its own file for focused responsibility.
// Middleware is applied deliberately per route group, not blindly globally.
//
// Reference: Master Prompt §14 — Security Middleware
package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

type contextKey string

const (
	requestIDKey contextKey = "request_id"
	traceIDKey   contextKey = "trace_id"
)

// RequestID injects a unique request_id and trace_id into every request.
// If X-Request-ID is provided by the client (or upstream proxy), it is reused.
//
// Reference: Master Prompt §10 — Request ID / Trace ID
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := r.Header.Get("X-Request-ID")
		if reqID == "" {
			reqID = uuid.New().String()
		}
		traceID := r.Header.Get("X-Trace-ID")
		if traceID == "" {
			traceID = uuid.New().String()
		}

		w.Header().Set("X-Request-ID", reqID)
		w.Header().Set("X-Trace-ID", traceID)

		ctx := context.WithValue(r.Context(), requestIDKey, reqID)
		ctx = context.WithValue(ctx, traceIDKey, traceID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetRequestID extracts the request ID from context.
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey).(string); ok {
		return id
	}
	return ""
}

// GetTraceID extracts the trace ID from context.
func GetTraceID(ctx context.Context) string {
	if id, ok := ctx.Value(traceIDKey).(string); ok {
		return id
	}
	return ""
}
