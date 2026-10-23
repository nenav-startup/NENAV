package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/nenav/nenav/backend/internal/domain/privacy"
	"github.com/nenav/nenav/backend/internal/service"
	"github.com/nenav/nenav/backend/internal/transport/http/response"
	"github.com/nenav/nenav/backend/internal/transport/middleware"
)

// PrivacyHandler handles DPDP compliance endpoints.
type PrivacyHandler struct {
	privacyService *service.PrivacyService
}

// NewPrivacyHandler creates a new PrivacyHandler instance.
func NewPrivacyHandler(privacyService *service.PrivacyService) *PrivacyHandler {
	return &PrivacyHandler{
		privacyService: privacyService,
	}
}

type recordConsentRequest struct {
	Purpose       string                 `json:"purpose"`
	Status        string                 `json:"status"`
	NoticeVersion string                 `json:"notice_version"`
	Metadata      map[string]interface{} `json:"metadata"`
}

// HandleRecordConsent records a user's explicit consent grant or revocation.
func (h *PrivacyHandler) HandleRecordConsent(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var req recordConsentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	record, err := h.privacyService.RecordConsent(
		r.Context(),
		userID,
		privacy.ConsentPurpose(req.Purpose),
		privacy.ConsentStatus(req.Status),
		req.NoticeVersion,
		r.RemoteAddr,
		r.UserAgent(),
		req.Metadata,
	)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	response.JSON(w, http.StatusCreated, record)
}

// HandleGetConsents lists active consents and audit history for the authenticated user.
func (h *PrivacyHandler) HandleGetConsents(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	active, err := h.privacyService.GetActiveConsents(r.Context(), userID)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	history, err := h.privacyService.GetConsentHistory(r.Context(), userID)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"active_consents": active,
		"history":         history,
	})
}

// HandleExportData complies with the Right to Data Portability (Section 11).
func (h *PrivacyHandler) HandleExportData(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	export, err := h.privacyService.RequestDataExport(r.Context(), userID)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	response.JSON(w, http.StatusOK, export)
}

type erasureRequest struct {
	Reason string `json:"reason"`
}

// HandleErasure executes the statutory Right to Erasure (Section 12) with a 72-hour SLA.
func (h *PrivacyHandler) HandleErasure(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var req erasureRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	dsr, err := h.privacyService.RequestErasure(r.Context(), userID, req.Reason)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"message":      "Data erasure completed under DPDP Section 12. PII scrubbed.",
		"dsr_id":       dsr.ID,
		"sla_deadline": dsr.SLADeadline,
		"status":       dsr.Status,
	})
}

// HandleListDSRs returns all statutory rights requests for the user.
func (h *PrivacyHandler) HandleListDSRs(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	dsrs, err := h.privacyService.ListUserDSRs(r.Context(), userID)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	response.JSON(w, http.StatusOK, dsrs)
}

type grievanceRequest struct {
	Subject     string `json:"subject"`
	Description string `json:"description"`
}

// HandleFileGrievance files a complaint to the Data Protection Officer under Section 13.
func (h *PrivacyHandler) HandleFileGrievance(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var req grievanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	g, err := h.privacyService.FileGrievance(r.Context(), userID, req.Subject, req.Description)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	response.JSON(w, http.StatusCreated, g)
}

// HandleListGrievances returns all grievances filed by the authenticated user.
func (h *PrivacyHandler) HandleListGrievances(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	grievances, err := h.privacyService.ListUserGrievances(r.Context(), userID)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	response.JSON(w, http.StatusOK, grievances)
}

type resolveGrievanceRequest struct {
	ResolutionNotes string `json:"resolution_notes"`
	DPOName         string `json:"dpo_name"`
}

// HandleResolveGrievance resolves a grievance as the DPO (Admin route).
func (h *PrivacyHandler) HandleResolveGrievance(w http.ResponseWriter, r *http.Request) {
	grievanceIDStr := chi.URLParam(r, "id")
	grievanceID, err := uuid.Parse(grievanceIDStr)
	if err != nil {
		http.Error(w, `{"error":"invalid grievance id"}`, http.StatusBadRequest)
		return
	}

	var req resolveGrievanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	g, err := h.privacyService.ResolveGrievance(r.Context(), grievanceID, req.ResolutionNotes, req.DPOName)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	response.JSON(w, http.StatusOK, g)
}

type setNomineeRequest struct {
	NomineeName  string `json:"nominee_name"`
	Relationship string `json:"relationship"`
	Phone        string `json:"phone"`
	Email        string `json:"email"`
}

// HandleSetNominee designates a Section 14 nominee for the data principal.
func (h *PrivacyHandler) HandleSetNominee(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var req setNomineeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	n, err := h.privacyService.SetNominee(r.Context(), userID, req.NomineeName, req.Relationship, req.Phone, req.Email)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	response.JSON(w, http.StatusCreated, n)
}

// HandleGetNominee returns the active Section 14 nominee for the user.
func (h *PrivacyHandler) HandleGetNominee(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	n, err := h.privacyService.GetNominee(r.Context(), userID)
	if err != nil {
		http.Error(w, `{"error":"no active nominee found"}`, http.StatusNotFound)
		return
	}

	response.JSON(w, http.StatusOK, n)
}

// HandleRevokeNominee revokes the active Section 14 nominee.
func (h *PrivacyHandler) HandleRevokeNominee(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	if err := h.privacyService.RevokeNominee(r.Context(), userID); err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{
		"message": "Nominee designation revoked successfully under DPDP Section 14.",
	})
}

type reportBreachRequest struct {
	IncidentReference string                 `json:"incident_reference"`
	NatureOfBreach    string                 `json:"nature_of_breach"`
	Severity          privacy.BreachSeverity `json:"severity"`
	AffectedCount     int                    `json:"affected_count"`
	MitigationTaken   string                 `json:"mitigation_taken"`
}

// HandleReportBreach records a personal data breach under Section 8(6) (Ops route).
func (h *PrivacyHandler) HandleReportBreach(w http.ResponseWriter, r *http.Request) {
	var req reportBreachRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	b, err := h.privacyService.ReportDataBreach(
		r.Context(),
		req.IncidentReference,
		req.NatureOfBreach,
		req.Severity,
		req.AffectedCount,
		req.MitigationTaken,
	)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	response.JSON(w, http.StatusCreated, b)
}

// HandleListBreaches lists recorded incidents (Ops route).
func (h *PrivacyHandler) HandleListBreaches(w http.ResponseWriter, r *http.Request) {
	breaches, err := h.privacyService.ListBreaches(r.Context())
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	response.JSON(w, http.StatusOK, breaches)
}

// HandleReportBreachToDPBI records notification to the Data Protection Board of India (Ops route).
func (h *PrivacyHandler) HandleReportBreachToDPBI(w http.ResponseWriter, r *http.Request) {
	breachIDStr := chi.URLParam(r, "id")
	breachID, err := uuid.Parse(breachIDStr)
	if err != nil {
		http.Error(w, `{"error":"invalid breach id"}`, http.StatusBadRequest)
		return
	}

	if err := h.privacyService.ReportBreachToDPBI(r.Context(), breachID); err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{
		"message": "Breach reported to DPBI under Section 8(6) statutory mandate.",
	})
}
