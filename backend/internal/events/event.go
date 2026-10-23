// Package events defines the domain event contract for NENAV.
//
// All domain events use this structure. The contracts are designed to be
// transport-agnostic: currently published via Redis queue, but portable
// to Kafka or any streaming system without changing domain code.
//
// Reference: Master Prompt §34 — Event System, §35 — Outbox Pattern
package events

import (
	"time"

	"github.com/google/uuid"
)

// Event represents a domain event.
type Event struct {
	ID         string    `json:"event_id"`
	Type       string    `json:"event_type"`
	Version    int       `json:"event_version"`
	OccurredAt time.Time `json:"occurred_at"`
	ActorID    string    `json:"actor_id"`
	EntityType string    `json:"entity_type"`
	EntityID   string    `json:"entity_id"`
	RequestID  string    `json:"request_id,omitempty"`
	Payload    any       `json:"payload"`
}

// New creates a new domain event with a generated ID and current timestamp.
func New(eventType string, actorID, entityType, entityID string, payload any) Event {
	return Event{
		ID:         uuid.New().String(),
		Type:       eventType,
		Version:    1,
		OccurredAt: time.Now().UTC(),
		ActorID:    actorID,
		EntityType: entityType,
		EntityID:   entityID,
		Payload:    payload,
	}
}

// WithRequestID returns a copy of the event with the request ID set.
func (e Event) WithRequestID(requestID string) Event {
	e.RequestID = requestID
	return e
}

// --- Well-known event types ---
// These constants define the event type namespace.
// New domains register their event types here.

const (
	// Identity
	UserCreated      = "user.created"
	UserUpdated      = "user.updated"
	UserDeactivated  = "user.deactivated"
	OTPSent          = "auth.otp.sent"
	LoginSucceeded   = "auth.login.succeeded"
	LoginFailed      = "auth.login.failed"
	SessionRevoked   = "auth.session.revoked"

	// PROF
	ProfCreated      = "prof.created"
	ProfVerified     = "prof.verified"
	ProfSuspended    = "prof.suspended"

	// Job + Booking
	JobCreated                  = "job.created"
	BookingCreated              = "booking.created"
	BookingAccepted             = "booking.accepted"
	BookingEnRoute              = "booking.en_route"
	BookingArrived              = "booking.arrived"
	BookingCancelled            = "booking.cancelled"
	BookingCheckedIn            = "booking.checked_in"
	BookingCheckedOut           = "booking.checked_out"
	AdditionalWorkRequested     = "booking.additional_work_requested"
	AdditionalWorkApproved      = "booking.additional_work_approved"
	AdditionalWorkDeclined      = "booking.additional_work_declined"
	BookingLocationUpdated      = "booking.location_updated"
	JobCompleted                = "job.completed"

	// Payment
	PaymentCreated   = "payment.created"
	PaymentCompleted = "payment.completed"
	PaymentFailed    = "payment.failed"
	RefundCreated    = "refund.created"

	// Rating + Dispute
	RatingCreated    = "rating.created"
	DisputeOpened    = "dispute.opened"
	DisputeResolved  = "dispute.resolved"
	SafetyAlertTriggered = "safety.alert_triggered"
)

// Publisher is the interface for publishing domain events.
// Implementations may use Redis, Kafka, or an outbox table.
type Publisher interface {
	Publish(event Event) error
}
