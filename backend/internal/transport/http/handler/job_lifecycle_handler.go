package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/nenav/nenav/backend/internal/domain/booking"
	"github.com/nenav/nenav/backend/internal/service"
	"github.com/nenav/nenav/backend/internal/transport/http/response"
)

type JobLifecycleHandler struct {
	bookingService *service.BookingService
}

func NewJobLifecycleHandler(b *service.BookingService) *JobLifecycleHandler {
	return &JobLifecycleHandler{bookingService: b}
}

func (h *JobLifecycleHandler) HandleEnRoute(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, `{"error":"invalid booking id"}`, http.StatusBadRequest)
		return
	}

	b, err := h.bookingService.AdvanceLifecycle(r.Context(), id, booking.StatusEnRoute, "")
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	response.JSON(w, http.StatusOK, b)
}

func (h *JobLifecycleHandler) HandleArrived(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, `{"error":"invalid booking id"}`, http.StatusBadRequest)
		return
	}

	b, err := h.bookingService.AdvanceLifecycle(r.Context(), id, booking.StatusArrived, "")
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	response.JSON(w, http.StatusOK, b)
}

type CheckinRequest struct {
	OTP string `json:"otp"`
}

func (h *JobLifecycleHandler) HandleCheckin(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, `{"error":"invalid booking id"}`, http.StatusBadRequest)
		return
	}

	var req CheckinRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request payload"}`, http.StatusBadRequest)
		return
	}

	b, err := h.bookingService.AdvanceLifecycle(r.Context(), id, booking.StatusInProgress, req.OTP)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusUnauthorized)
		return
	}

	response.JSON(w, http.StatusOK, b)
}

func (h *JobLifecycleHandler) HandleComplete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, `{"error":"invalid booking id"}`, http.StatusBadRequest)
		return
	}

	b, err := h.bookingService.AdvanceLifecycle(r.Context(), id, booking.StatusCompleted, "")
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	response.JSON(w, http.StatusOK, b)
}

type ClaimRequest struct {
	ProfID uuid.UUID `json:"prof_id"`
}

func (h *JobLifecycleHandler) HandleClaim(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, `{"error":"invalid booking id"}`, http.StatusBadRequest)
		return
	}

	var req ClaimRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request payload"}`, http.StatusBadRequest)
		return
	}

	b, err := h.bookingService.ClaimBooking(r.Context(), id, req.ProfID)
	if err != nil {
		if err == booking.ErrAlreadyClaimed {
			http.Error(w, `{"error":"This job was just accepted by another professional"}`, http.StatusConflict)
			return
		}
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	response.JSON(w, http.StatusOK, b)
}

type AdditionalWorkRequest struct {
	LaborCost    int    `json:"labor_cost"`
	MaterialCost int    `json:"material_cost"`
	Description  string `json:"description"`
}

func (h *JobLifecycleHandler) HandleAdditionalWork(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, `{"error":"invalid booking id"}`, http.StatusBadRequest)
		return
	}

	var req AdditionalWorkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request payload"}`, http.StatusBadRequest)
		return
	}

	quote := booking.AdditionalWorkQuote{
		LaborCost:    req.LaborCost,
		MaterialCost: req.MaterialCost,
		Description:  req.Description,
	}

	b, err := h.bookingService.RequestAdditionalWork(r.Context(), id, quote)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	response.JSON(w, http.StatusOK, b)
}

func (h *JobLifecycleHandler) HandleApproveAdditionalWork(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, `{"error":"invalid booking id"}`, http.StatusBadRequest)
		return
	}

	b, err := h.bookingService.ApproveAdditionalWork(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	response.JSON(w, http.StatusOK, b)
}

func (h *JobLifecycleHandler) HandleDeclineAdditionalWork(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, `{"error":"invalid booking id"}`, http.StatusBadRequest)
		return
	}

	b, err := h.bookingService.DeclineAdditionalWork(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	response.JSON(w, http.StatusOK, b)
}

type LocationUpdateRequest struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

func (h *JobLifecycleHandler) HandleLocation(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, `{"error":"invalid booking id"}`, http.StatusBadRequest)
		return
	}

	var req LocationUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request payload"}`, http.StatusBadRequest)
		return
	}

	b, err := h.bookingService.UpdateLocation(r.Context(), id, req.Lat, req.Lng)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	response.JSON(w, http.StatusOK, b)
}
