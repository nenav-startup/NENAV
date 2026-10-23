package handler

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/nenav/nenav/backend/internal/service"
	"github.com/nenav/nenav/backend/internal/transport/http/response"
)

type PaymentHandler struct {
	paymentService *service.PaymentService
}

func NewPaymentHandler(p *service.PaymentService) *PaymentHandler {
	return &PaymentHandler{paymentService: p}
}

type OnlineConfirmRequest struct {
	BookingID        string `json:"booking_id"`
	GatewayPaymentID string `json:"gateway_payment_id"`
	Signature        string `json:"signature"`
}

func (h *PaymentHandler) HandleOnlineConfirm(w http.ResponseWriter, r *http.Request) {
	var req OnlineConfirmRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request payload"}`, http.StatusBadRequest)
		return
	}

	bID, err := uuid.Parse(req.BookingID)
	if err != nil {
		http.Error(w, `{"error":"invalid booking id"}`, http.StatusBadRequest)
		return
	}

	p, err := h.paymentService.ConfirmOnlinePayment(r.Context(), bID, req.GatewayPaymentID, req.Signature)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	response.JSON(w, http.StatusOK, map[string]any{
		"message": "Payment captured, searching for nearest verified professional",
		"payment": p,
	})
}

type CashConfirmRequest struct {
	BookingID  string `json:"booking_id"`
	IsCustomer bool   `json:"is_customer"` // true if customer, false if worker
}

func (h *PaymentHandler) HandleCashConfirm(w http.ResponseWriter, r *http.Request) {
	var req CashConfirmRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request payload"}`, http.StatusBadRequest)
		return
	}

	bID, err := uuid.Parse(req.BookingID)
	if err != nil {
		http.Error(w, `{"error":"invalid booking id"}`, http.StatusBadRequest)
		return
	}

	p, bothConfirmed, err := h.paymentService.ConfirmCashPayment(r.Context(), bID, req.IsCustomer)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	msg := "Cash confirmation recorded. Awaiting confirmation from other party."
	if bothConfirmed {
		msg = "Bilateral cash confirmation verified. Job settled and completed."
	}

	response.JSON(w, http.StatusOK, map[string]any{
		"message":        msg,
		"both_confirmed": bothConfirmed,
		"payment":        p,
	})
}
