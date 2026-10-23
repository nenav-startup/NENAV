package privacy

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/nenav/nenav/backend/internal/domain/identity"
)

var (
	ErrInvalidPurpose     = errors.New("invalid consent purpose: must be analytics, marketing, precise_location, service_delivery, or ai_processing")
	ErrInvalidStatus      = errors.New("invalid consent status: must be granted or revoked")
	ErrInvalidDSRType     = errors.New("invalid dsr type: must be access_export, correction, erasure, nominee, or grievance")
	ErrDSRNotFound        = errors.New("data subject request not found")
	ErrGrievanceNotFound  = errors.New("privacy grievance not found")
	ErrSubjectRequired    = errors.New("grievance subject is required")
	ErrDescriptionRequired = errors.New("grievance description is required")
)

// ConsentPurpose defines standard processing purposes under DPDP Act 2023.
type ConsentPurpose string

const (
	PurposeAnalytics       ConsentPurpose = "analytics"
	PurposeMarketing       ConsentPurpose = "marketing"
	PurposePreciseLocation ConsentPurpose = "precise_location"
	PurposeServiceDelivery ConsentPurpose = "service_delivery"
	PurposeAIProcessing    ConsentPurpose = "ai_processing"
)

// ConsentStatus indicates current agreement state.
type ConsentStatus string

const (
	StatusGranted ConsentStatus = "granted"
	StatusRevoked ConsentStatus = "revoked"
)

// ConsentRecord models an immutable entry in the consent audit ledger.
type ConsentRecord struct {
	ID            uuid.UUID              `json:"id"`
	UserID        uuid.UUID              `json:"user_id"`
	Purpose       ConsentPurpose         `json:"purpose"`
	Status        ConsentStatus          `json:"status"`
	NoticeVersion string                 `json:"notice_version"`
	IPAddress     string                 `json:"ip_address,omitempty"`
	UserAgent     string                 `json:"user_agent,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt     time.Time              `json:"created_at"`
}

// DSRType categorizes Data Subject Rights under DPDP Act 2023 (Sections 11, 12, 13, 14).
type DSRType string

const (
	DSRTypeAccessExport DSRType = "access_export"
	DSRTypeCorrection   DSRType = "correction"
	DSRTypeErasure      DSRType = "erasure"
	DSRTypeNominee      DSRType = "nominee"
	DSRTypeGrievance    DSRType = "grievance"
)

// DSRStatus tracks processing stages for a data subject request.
type DSRStatus string

const (
	DSRStatusPending    DSRStatus = "pending"
	DSRStatusProcessing DSRStatus = "processing"
	DSRStatusCompleted  DSRStatus = "completed"
	DSRStatusRejected   DSRStatus = "rejected"
)

// DataSubjectRequest tracks statutory rights requests.
type DataSubjectRequest struct {
	ID              uuid.UUID              `json:"id"`
	UserID          uuid.UUID              `json:"user_id"`
	RequestType     DSRType                `json:"request_type"`
	Status          DSRStatus              `json:"status"`
	Details         map[string]interface{} `json:"details,omitempty"`
	SLADeadline     time.Time              `json:"sla_deadline"`
	CompletedAt     *time.Time             `json:"completed_at,omitempty"`
	RejectionReason *string                `json:"rejection_reason,omitempty"`
	ExportFileURL   *string                `json:"export_file_url,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

// GrievanceStatus models the resolution stages for DPO complaints.
type GrievanceStatus string

const (
	GrievanceStatusOpen               GrievanceStatus = "open"
	GrievanceStatusUnderInvestigation GrievanceStatus = "under_investigation"
	GrievanceStatusResolved           GrievanceStatus = "resolved"
	GrievanceStatusEscalatedToDPBI    GrievanceStatus = "escalated_to_dpbi"
)

// Grievance models a user's direct escalation to the Data Protection Officer.
type Grievance struct {
	ID              uuid.UUID       `json:"id"`
	TicketNumber    string          `json:"ticket_number"`
	UserID          uuid.UUID       `json:"user_id"`
	Subject         string          `json:"subject"`
	Description     string          `json:"description"`
	DPOName         string          `json:"dpo_name"`
	Status          GrievanceStatus `json:"status"`
	ResolutionNotes *string         `json:"resolution_notes,omitempty"`
	ResolvedAt      *time.Time      `json:"resolved_at,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

// PrivacyAccessLog records whenever sensitive customer/prof data is accessed.
type PrivacyAccessLog struct {
	ID             uuid.UUID `json:"id"`
	AccessorID     uuid.UUID `json:"accessor_id"`
	AccessorRole   string    `json:"accessor_role"`
	TargetUserID   uuid.UUID `json:"target_user_id"`
	AccessedFields []string  `json:"accessed_fields"`
	Reason         string    `json:"reason"`
	IPAddress      string    `json:"ip_address,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

// UserDataExport defines the comprehensive machine-readable portability payload (DPDP Section 11).
type UserDataExport struct {
	ExportID        uuid.UUID               `json:"export_id"`
	ExportTimestamp time.Time               `json:"export_timestamp"`
	User            *identity.User          `json:"user"`
	ActiveConsents  map[ConsentPurpose]bool `json:"active_consents"`
	ConsentHistory  []ConsentRecord         `json:"consent_history"`
	BookingsCount   int                     `json:"bookings_count"`
	PaymentsCount   int                     `json:"payments_count"`
	DisputesCount   int                     `json:"disputes_count"`
}

// CalculateSLADeadline returns statutory deadline based on request type.
// Erasure has strict 72-hour SLA under NENAV DPDP compliance rules.
// General DSRs have 30-day statutory resolution timelines.
func CalculateSLADeadline(reqType DSRType, from time.Time) time.Time {
	if reqType == DSRTypeErasure {
		return from.Add(72 * time.Hour)
	}
	return from.Add(30 * 24 * time.Hour)
}

// GenerateTicketNumber produces a formatted grievance ticket string.
func GenerateTicketNumber() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return fmt.Sprintf("DPDP-GRV-%s-%s", time.Now().Format("20060102"), hex.EncodeToString(b))
}

// ValidatePurpose ensures purpose is one of the approved DPDP categories.
func ValidatePurpose(p ConsentPurpose) error {
	switch p {
	case PurposeAnalytics, PurposeMarketing, PurposePreciseLocation, PurposeServiceDelivery, PurposeAIProcessing:
		return nil
	default:
		return ErrInvalidPurpose
	}
}

// ValidateDSRType ensures the request type complies with statutory rights.
func ValidateDSRType(t DSRType) error {
	switch t {
	case DSRTypeAccessExport, DSRTypeCorrection, DSRTypeErasure, DSRTypeNominee, DSRTypeGrievance:
		return nil
	default:
		return ErrInvalidDSRType
	}
}
