package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

// Logger logs every HTTP request with structured fields.
// Includes request_id, trace_id, method, path, status, duration, and remote address.
//
// Reference: Master Prompt §38 — Observability, §80 — Logging Rule
func Logger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := &statusWriter{ResponseWriter: w, statusCode: http.StatusOK}

			next.ServeHTTP(ww, r)

			duration := time.Since(start)
			level := slog.LevelInfo
			if ww.statusCode >= 500 {
				level = slog.LevelError
			} else if ww.statusCode >= 400 {
				level = slog.LevelWarn
			}

			logger.Log(r.Context(), level, "http_request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", ww.statusCode,
				"duration_ms", duration.Milliseconds(),
				"bytes", ww.bytesWritten,
				"request_id", GetRequestID(r.Context()),
				"trace_id", GetTraceID(r.Context()),
				"remote_addr", r.RemoteAddr,
				"user_agent", r.UserAgent(),
			)
		})
	}
}

// statusWriter wraps http.ResponseWriter to capture status code and bytes.
type statusWriter struct {
	http.ResponseWriter
	statusCode   int
	wroteHeader  bool
	bytesWritten int
}

func (w *statusWriter) WriteHeader(code int) {
	if !w.wroteHeader {
		w.statusCode = code
		w.wroteHeader = true
		w.ResponseWriter.WriteHeader(code)
	}
}

func (w *statusWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	n, err := w.ResponseWriter.Write(b)
	w.bytesWritten += n
	return n, err
}

func (w *statusWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}
