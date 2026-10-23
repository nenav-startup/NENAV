package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"
)

// Recovery recovers from panics, logs the stack trace, and returns 500.
// Stack traces are logged but never exposed to the client.
func Recovery(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					stack := debug.Stack()
					logger.Error("panic_recovered",
						"error", err,
						"stack", string(stack),
						"method", r.Method,
						"path", r.URL.Path,
						"request_id", GetRequestID(r.Context()),
					)
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(`{"error":{"code":"internal_error","message":"internal server error"}}`))
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
