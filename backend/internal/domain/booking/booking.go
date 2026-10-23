package booking

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidStatusTransition = errors.New("invalid booking status transition")
	ErrBookingNotFound          = errors.New("booking not found")
	ErrInvalidOTPCheckin        = errors.New("invalid 4-digit check-in otp")
	ErrAlreadyCompleted         = errors.New("booking is already completed")
	ErrCannotCancel             = errors.New("booking cannot be cancelled in current status")
	ErrAlreadyClaimed           = errors.New("this job was just accepted by another professional")
	ErrInvalidAdditionalWork    = errors.New("cannot request additional work in current status")
)

type BookingStatus string

const (
	StatusDraft          BookingStatus = "draft"
	StatusPendingPayment BookingStatus = "pending_payment"
	StatusSearching      BookingStatus = "searching"
	StatusAssigned       BookingStatus = "assigned"
	StatusEnRoute        BookingStatus = "en_route"
	StatusArrived        BookingStatus = "arrived"
	StatusInProgress             BookingStatus = "in_progress"
	StatusAdditionalWorkPending  BookingStatus = "additional_work_pending"
	StatusCompleted              BookingStatus = "completed"
	StatusCancelled              BookingStatus = "cancelled"
	StatusDisputed               BookingStatus = "disputed"
)

type DurationType string

const (
	DurationHourly    DurationType = "hourly"
	DurationHalfDay   DurationType = "half_day"
	DurationFullDay   DurationType = "full_day"
	DurationMultiDay  DurationType = "multi_day"
	DurationRecurring DurationType = "recurring"
)

// AdditionalWorkQuote represents an on-site scope change requiring customer authorization.
type AdditionalWorkQuote struct {
	ID           string     `json:"id"`
	LaborCost    int        `json:"labor_cost"`
	MaterialCost int        `json:"material_cost"`
	TotalAmount  int        `json:"total_amount"`
	Description  string     `json:"description"`
	Status       string     `json:"status"` // "pending", "approved", "declined"
	RequestedAt  time.Time  `json:"requested_at"`
	RespondedAt  *time.Time `json:"responded_at,omitempty"`
}

// QuoteBreakdown represents dynamic itemized pricing before confirmation.
type QuoteBreakdown struct {
	BaseWagePerUnit int `json:"base_wage_per_unit"`
	Hours           int `json:"hours"`
	WorkerCount     int `json:"worker_count"`
	BaseWage        int `json:"base_wage"`
	PlatformFee     int `json:"platform_fee"` // 10%
	SafetyCover     int `json:"safety_cover"` // ₹39 per worker
	Subtotal        int `json:"subtotal"`
	GST             int `json:"gst"` // 18% on subtotal
	TotalAmount     int `json:"total_amount"`
	Currency        string `json:"currency"`
}

// Booking represents the authoritative marketplace transaction.
type Booking struct {
	ID                          uuid.UUID            `json:"id"`
	BookingReference            string               `json:"booking_reference"`
	CustomerID                  uuid.UUID            `json:"customer_id"`
	TradeID                     uuid.UUID            `json:"trade_id"`
	TradeName                   string               `json:"trade_name,omitempty"`
	AssignedProfID              *uuid.UUID           `json:"assigned_prof_id,omitempty"`
	Status                      BookingStatus        `json:"status"`
	DurationType                DurationType         `json:"duration_type"`
	ScheduledDate               string               `json:"scheduled_date"`
	TimeSlot                    string               `json:"time_slot"`
	Hours                       int                  `json:"hours"`
	WorkerCount                 int                  `json:"worker_count"`
	AddressID                   *uuid.UUID           `json:"address_id,omitempty"`
	AddressLine                 string               `json:"address_line,omitempty"`
	JobDescription              string               `json:"job_description"`
	CheckinOTP                  string               `json:"checkin_otp"` // 4-digit code (e.g. "4829")
	Quote                       QuoteBreakdown       `json:"quote"`
	AdditionalWork              *AdditionalWorkQuote `json:"additional_work,omitempty"`
	LocationLat                 *float64             `json:"location_lat,omitempty"`
	LocationLng                 *float64             `json:"location_lng,omitempty"`
	LocationUpdatedAt           *time.Time           `json:"location_updated_at,omitempty"`
	PaymentStatus               string               `json:"payment_status"`
	PaymentMethod               string               `json:"payment_method"`
	CustomerConfirmedCompletion bool                 `json:"customer_confirmed_completion"`
	ProfConfirmedCompletion     bool                 `json:"prof_confirmed_completion"`
	CancelledAt                 *time.Time           `json:"cancelled_at,omitempty"`
	CancellationReason          *string              `json:"cancellation_reason,omitempty"`
	StartedAt                   *time.Time           `json:"started_at,omitempty"`
	CompletedAt                 *time.Time           `json:"completed_at,omitempty"`
	CreatedAt                   time.Time            `json:"created_at"`
	UpdatedAt                   time.Time            `json:"updated_at"`
}

// ValidTransitions maps allowed state progressions.
var ValidTransitions = map[BookingStatus][]BookingStatus{
	StatusDraft:                 {StatusPendingPayment, StatusCancelled},
	StatusPendingPayment:        {StatusSearching, StatusCancelled},
	StatusSearching:             {StatusAssigned, StatusCancelled},
	StatusAssigned:              {StatusEnRoute, StatusSearching, StatusCancelled},
	StatusEnRoute:               {StatusArrived, StatusCancelled},
	StatusArrived:               {StatusInProgress, StatusDisputed},
	StatusInProgress:            {StatusCompleted, StatusAdditionalWorkPending, StatusDisputed},
	StatusAdditionalWorkPending: {StatusInProgress, StatusDisputed, StatusCancelled},
	StatusCompleted:             {StatusDisputed},
	StatusCancelled:             {},
	StatusDisputed:              {StatusCompleted, StatusCancelled},
}

// CanTransition validates whether a state transition is legal.
func CanTransition(from, to BookingStatus) bool {
	allowed, ok := ValidTransitions[from]
	if !ok {
		return false
	}
	for _, s := range allowed {
		if s == to {
			return true
		}
	}
	return false
}

// CalculateQuote computes the authoritative itemized price.
func CalculateQuote(baseHourlyRate, hours, workerCount int) QuoteBreakdown {
	if hours < 2 {
		hours = 2 // Minimum 2 hours
	}
	if workerCount < 1 {
		workerCount = 1
	}

	baseWage := baseHourlyRate * hours * workerCount
	platformFee := int(float64(baseWage) * 0.10) // 10%
	safetyCover := 39 * workerCount              // Flat ₹39 per worker
	subtotal := baseWage + platformFee + safetyCover
	gst := int(float64(subtotal) * 0.18)         // 18% GST
	total := subtotal + gst

	return QuoteBreakdown{
		BaseWagePerUnit: baseHourlyRate,
		Hours:           hours,
		WorkerCount:     workerCount,
		BaseWage:        baseWage,
		PlatformFee:     platformFee,
		SafetyCover:     safetyCover,
		Subtotal:        subtotal,
		GST:             gst,
		TotalAmount:     total,
		Currency:        "INR",
	}
}

// GenerateBookingReference creates human-readable reference, e.g. NV-2026-8842
func GenerateBookingReference(sequence int) string {
	return fmt.Sprintf("NV-2026-%04d", sequence%10000)
}
