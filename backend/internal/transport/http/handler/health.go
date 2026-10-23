// Package handler contains HTTP handlers grouped by concern.
//
// Handlers are thin: they decode the request, call a service, and encode
// the response. Business logic does not belong here.
package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/nenav/nenav/backend/internal/transport/http/response"
)

// Pinger is the interface for dependency health checks.
type Pinger interface {
	Ping(ctx context.Context) error
}

// Health handles /health, /ready, and /live endpoints.
//
// Reference: Master Prompt §39 — Health Endpoints
type Health struct {
	db        Pinger
	cache     Pinger
	startTime time.Time
}

// NewHealth creates a new health handler.
func NewHealth(db, cache Pinger) *Health {
	return &Health{
		db:        db,
		cache:     cache,
		startTime: time.Now(),
	}
}

type healthResponse struct {
	Status    string            `json:"status"`
	Uptime    string            `json:"uptime"`
	Timestamp string            `json:"timestamp"`
	Checks    map[string]string `json:"checks,omitempty"`
}

// HandleHealth returns basic health status. Always responds if the process is alive.
// GET /health
func (h *Health) HandleHealth(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, healthResponse{
		Status:    "ok",
		Uptime:    time.Since(h.startTime).Round(time.Second).String(),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

// HandleReady verifies that all dependencies are available.
// Returns 503 if any dependency is unhealthy.
// GET /ready
func (h *Health) HandleReady(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	checks := make(map[string]string)
	allOK := true

	if err := h.db.Ping(ctx); err != nil {
		checks["postgres"] = "unhealthy"
		allOK = false
	} else {
		checks["postgres"] = "healthy"
	}

	if err := h.cache.Ping(ctx); err != nil {
		checks["redis"] = "unhealthy"
		allOK = false
	} else {
		checks["redis"] = "healthy"
	}

	status := "ready"
	statusCode := http.StatusOK
	if !allOK {
		status = "not_ready"
		statusCode = http.StatusServiceUnavailable
	}

	resp := healthResponse{
		Status:    status,
		Uptime:    time.Since(h.startTime).Round(time.Second).String(),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Checks:    checks,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	response.JSON(w, statusCode, resp)
}

// HandleLive indicates the process is alive (for liveness probes).
// GET /live
func (h *Health) HandleLive(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, map[string]string{"status": "alive"})
}
