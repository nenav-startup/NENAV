// Package observability provides structured logging setup.
//
// Reference: Master Prompt §38 — Observability (Logs)
package observability

import (
	"io"
	"log/slog"
	"os"
)

// NewLogger creates a structured JSON logger at the given level.
// All NENAV logs use structured JSON for machine parsing.
func NewLogger(level string, w io.Writer) *slog.Logger {
	if w == nil {
		w = os.Stdout
	}

	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "info":
		lvl = slog.LevelInfo
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	handler := slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level:     lvl,
		AddSource: lvl == slog.LevelDebug,
	})

	return slog.New(handler)
}
