package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/nenav/nenav/backend/internal/domain/booking"
	"github.com/nenav/nenav/backend/internal/service"
	"github.com/nenav/nenav/backend/internal/transport/http/response"
	"github.com/nenav/nenav/backend/internal/transport/middleware"
)

type BookingHandler struct {
	bookingService *service.BookingService
}

func NewBookingHandler(b *service.BookingService) *BookingHandler {
	return &BookingHandler{bookingService: b}
}

type QuoteRequest struct {
	TradeID     string `json:"trade_id"`
	Hours       int    `json:"hours"`
	WorkerCount int    `json:"worker_count"`
}

func (h *BookingHandler) HandleQuote(w http.ResponseWriter, r *http.Request) {
	var req QuoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request payload"}`, http.StatusBadRequest)
		return
	}

	tradeUUID, _ := uuid.Parse(req.TradeID)
	quote, err := h.bookingService.GetQuote(r.Context(), tradeUUID, req.Hours, req.WorkerCount)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	response.JSON(w, http.StatusOK, quote)
}

type CreateBookingRequest struct {
	CustomerID     string `json:"customer_id"`
	TradeID        string `json:"trade_id"`
	TradeName      string `json:"trade_name"`
	ScheduledDate  string `json:"scheduled_date"`
	TimeSlot       string `json:"time_slot"`
	DurationType   string `json:"duration_type"`
	Hours          int    `json:"hours"`
	WorkerCount    int    `json:"worker_count"`
	AddressLine    string `json:"address_line"`
	JobDescription string `json:"job_description"`
	PaymentMethod  string `json:"payment_method"`
}

func (h *BookingHandler) HandleCreateBooking(w http.ResponseWriter, r *http.Request) {
	var req CreateBookingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request payload"}`, http.StatusBadRequest)
		return
	}

	userIDVal := r.Context().Value(middleware.UserIDKey)
	if userIDVal == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	custID := userIDVal.(uuid.UUID)

	tradeID, _ := uuid.Parse(req.TradeID)

	b, err := h.bookingService.CreateBooking(r.Context(), service.CreateBookingInput{
		CustomerID:     custID,
		TradeID:        tradeID,
		TradeName:      req.TradeName,
		ScheduledDate:  req.ScheduledDate,
		TimeSlot:       req.TimeSlot,
		DurationType:   booking.DurationType(req.DurationType),
		Hours:          req.Hours,
		WorkerCount:    req.WorkerCount,
		AddressLine:    req.AddressLine,
		JobDescription: req.JobDescription,
		PaymentMethod:  req.PaymentMethod,
	})

	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	response.Created(w, b)
}

func (h *BookingHandler) HandleGetBooking(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, `{"error":"invalid booking id"}`, http.StatusBadRequest)
		return
	}

	b, err := h.bookingService.GetBooking(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"booking not found"}`, http.StatusNotFound)
		return
	}

	response.JSON(w, http.StatusOK, b)
}

func (h *BookingHandler) HandleListBookings(w http.ResponseWriter, r *http.Request) {
	userIDVal := r.Context().Value(middleware.UserIDKey)
	if userIDVal == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	custID := userIDVal.(uuid.UUID)

	list, err := h.bookingService.ListBookings(r.Context(), custID)
	if err != nil {
		http.Error(w, `{"error":"failed to list bookings"}`, http.StatusInternalServerError)
		return
	}

	response.JSON(w, http.StatusOK, list)
}

type CancelRequest struct {
	Reason string `json:"reason"`
}

func (h *BookingHandler) HandleCancelBooking(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, `{"error":"invalid booking id"}`, http.StatusBadRequest)
		return
	}

	var req CancelRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.Reason == "" {
		req.Reason = "Customer requested cancellation"
	}

	b, err := h.bookingService.CancelBooking(r.Context(), id, req.Reason)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	response.JSON(w, http.StatusOK, b)
}
