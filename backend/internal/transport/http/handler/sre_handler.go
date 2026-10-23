package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/nenav/nenav/backend/internal/repository"
	"github.com/nenav/nenav/backend/internal/service"
	"github.com/nenav/nenav/backend/internal/transport/http/response"
)

type SREHandler struct {
	store       repository.Store
	matchingSvc *service.MatchingService
}

func NewSREHandler(store repository.Store, matchingSvc *service.MatchingService) *SREHandler {
	return &SREHandler{
		store:       store,
		matchingSvc: matchingSvc,
	}
}

func (h *SREHandler) HandleGetStuckBookings(w http.ResponseWriter, r *http.Request) {
	stuck, err := h.store.ListStuckBookings(r.Context(), 5*time.Minute)
	if err != nil {
		http.Error(w, `{"error":"failed to fetch stuck bookings"}`, http.StatusInternalServerError)
		return
	}
	response.JSON(w, http.StatusOK, stuck)
}

type ForceAssignRequest struct {
	ProfID string `json:"prof_id"`
}

func (h *SREHandler) HandleForceAssign(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	bookingID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, `{"error":"invalid booking id"}`, http.StatusBadRequest)
		return
	}

	var req ForceAssignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request payload"}`, http.StatusBadRequest)
		return
	}

	profID, err := uuid.Parse(req.ProfID)
	if err != nil {
		http.Error(w, `{"error":"invalid prof id"}`, http.StatusBadRequest)
		return
	}

	b, err := h.store.GetBookingByID(r.Context(), bookingID)
	if err != nil {
		http.Error(w, `{"error":"booking not found"}`, http.StatusNotFound)
		return
	}

	// In a real MVP we would probably implement this logic inside MatchingService.
	// For now, we will just call a force assign method if it exists, 
	// or we update the booking directly.
	// We should just use MatchingService since it holds the logic.
	
	p, err := h.store.GetPROFByID(r.Context(), profID)
	if err != nil {
		http.Error(w, `{"error":"professional not found"}`, http.StatusNotFound)
		return
	}

	// Manually force assign
	b.AssignedProfID = &p.ID
	b.Status = "assigned"
	b.UpdatedAt = time.Now()
	b.CheckinOTP = "0000" // Reset/override OTP

	if err := h.store.UpdateBooking(r.Context(), b); err != nil {
		http.Error(w, `{"error":"failed to force assign"}`, http.StatusInternalServerError)
		return
	}

	response.JSON(w, http.StatusOK, map[string]any{
		"message": "Successfully force assigned booking",
		"booking": b,
	})
}
