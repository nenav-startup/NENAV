package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/nenav/nenav/backend/internal/domain/support"
	"github.com/nenav/nenav/backend/internal/service"
	"github.com/nenav/nenav/backend/internal/transport/http/response"
)

type SupportHandler struct {
	supportService *service.SupportService
}

func NewSupportHandler(s *service.SupportService) *SupportHandler {
	return &SupportHandler{supportService: s}
}

type CreateDisputeRequest struct {
	BookingID   string   `json:"booking_id"`
	RaisedBy    string   `json:"raised_by"`
	Reason      string   `json:"reason"`
	Description string   `json:"description"`
	Evidence    []string `json:"evidence"`
}

func (h *SupportHandler) HandleCreateDispute(w http.ResponseWriter, r *http.Request) {
	var req CreateDisputeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request payload"}`, http.StatusBadRequest)
		return
	}

	bID, err := uuid.Parse(req.BookingID)
	if err != nil {
		http.Error(w, `{"error":"invalid booking id"}`, http.StatusBadRequest)
		return
	}

	raisedBy, _ := uuid.Parse(req.RaisedBy)
	if raisedBy == uuid.Nil {
		raisedBy = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	}

	d, err := h.supportService.CreateDispute(r.Context(), bID, raisedBy, support.DisputeReason(req.Reason), req.Description, req.Evidence)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	response.Created(w, d)
}

func (h *SupportHandler) HandleListDisputes(w http.ResponseWriter, r *http.Request) {
	list, err := h.supportService.ListDisputes(r.Context())
	if err != nil {
		http.Error(w, `{"error":"failed to list disputes"}`, http.StatusInternalServerError)
		return
	}
	response.JSON(w, http.StatusOK, list)
}

type ResolveDisputeRequest struct {
	AdminID      string `json:"admin_id"`
	Notes        string `json:"notes"`
	ShouldRefund bool   `json:"should_refund"`
}

func (h *SupportHandler) HandleResolveDispute(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, `{"error":"invalid dispute id"}`, http.StatusBadRequest)
		return
	}

	var req ResolveDisputeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request payload"}`, http.StatusBadRequest)
		return
	}

	adminUUID, _ := uuid.Parse(req.AdminID)
	if adminUUID == uuid.Nil {
		adminUUID = uuid.New()
	}

	d, err := h.supportService.ResolveDispute(r.Context(), id, adminUUID, req.Notes, req.ShouldRefund)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	response.JSON(w, http.StatusOK, d)
}
