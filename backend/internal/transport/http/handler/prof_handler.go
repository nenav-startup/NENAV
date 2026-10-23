package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/nenav/nenav/backend/internal/repository"
	"github.com/nenav/nenav/backend/internal/transport/http/response"
)

type ProfHandler struct {
	store repository.Store
}

func NewProfHandler(s repository.Store) *ProfHandler {
	return &ProfHandler{store: s}
}

func (h *ProfHandler) HandleListPROFs(w http.ResponseWriter, r *http.Request) {
	profs, err := h.store.ListPROFs(r.Context())
	if err != nil {
		http.Error(w, `{"error":"failed to list professionals"}`, http.StatusInternalServerError)
		return
	}
	response.JSON(w, http.StatusOK, profs)
}

func (h *ProfHandler) HandleGetPROF(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, `{"error":"invalid prof id"}`, http.StatusBadRequest)
		return
	}

	p, err := h.store.GetPROFByID(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"professional not found"}`, http.StatusNotFound)
		return
	}

	response.JSON(w, http.StatusOK, p)
}
