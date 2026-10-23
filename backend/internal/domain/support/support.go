package support

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrDisputeNotFound = errors.New("dispute case not found")
	ErrAlreadyResolved = errors.New("dispute has already been resolved")
)

type DisputeReason string

const (
	ReasonLateArrival    DisputeReason = "late_arrival"
	ReasonNoShow         DisputeReason = "no_show"
	ReasonConduct        DisputeReason = "conduct"
	ReasonDamage         DisputeReason = "damage"
	ReasonIncomplete     DisputeReason = "incomplete"
	ReasonSkillMismatch  DisputeReason = "skill_mismatch"
	ReasonPaymentDispute DisputeReason = "payment_dispute"
)

type DisputeStatus string

const (
	StatusOpen        DisputeStatus = "open"
	StatusUnderReview DisputeStatus = "under_review"
	StatusResolved    DisputeStatus = "resolved"
	StatusDismissed   DisputeStatus = "dismissed"
)

// Dispute represents a contested service execution requiring mediation.
type Dispute struct {
	ID              uuid.UUID     `json:"id"`
	BookingID       uuid.UUID     `json:"booking_id"`
	RaisedBy        uuid.UUID     `json:"raised_by"`
	Reason          DisputeReason `json:"reason"`
	Description     string        `json:"description"`
	EvidenceURLs    []string      `json:"evidence_urls"`
	Status          DisputeStatus `json:"status"`
	ResolutionNotes *string       `json:"resolution_notes,omitempty"`
	ResolvedBy      *uuid.UUID    `json:"resolved_by,omitempty"`
	ResolvedAt      *time.Time    `json:"resolved_at,omitempty"`
	CreatedAt       time.Time     `json:"created_at"`
}

// SupportTicket represents a customer or professional inquiry.
type SupportTicket struct {
	ID          uuid.UUID `json:"id"`
	UserID      uuid.UUID `json:"user_id"`
	Subject     string    `json:"subject"`
	Category    string    `json:"category"`
	Priority    string    `json:"priority"` // low, normal, urgent, emergency
	Status      string    `json:"status"`   // open, in_progress, resolved, closed
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}
