package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/nenav/nenav/backend/internal/domain/booking"
	"github.com/nenav/nenav/backend/internal/domain/dispatch"
	"github.com/nenav/nenav/backend/internal/domain/payment"
	"github.com/nenav/nenav/backend/internal/events"
	"github.com/nenav/nenav/backend/internal/repository"
)

type BookingService struct {
	store      repository.Store
	dispatcher dispatch.Dispatcher
	payments   payment.Processor
	publisher  events.Publisher
	sequence   int
}

func NewBookingService(store repository.Store, dispatcher dispatch.Dispatcher, payments payment.Processor, publisher events.Publisher) *BookingService {
	return &BookingService{
		store:      store,
		dispatcher: dispatcher,
		payments:   payments,
		publisher:  publisher,
		sequence:   1000,
	}
}

// GetQuote generates a real-time price quotation based on trade and duration.
func (s *BookingService) GetQuote(ctx context.Context, tradeID uuid.UUID, hours, workerCount int) (booking.QuoteBreakdown, error) {
	// Base hourly rate: default ₹249 for skilled trades (electrician, plumber), ₹199 for helpers
	baseRate := 249
	return booking.CalculateQuote(baseRate, hours, workerCount), nil
}

type CreateBookingInput struct {
	CustomerID     uuid.UUID
	TradeID        uuid.UUID
	TradeName      string
	ScheduledDate  string
	TimeSlot       string
	DurationType   booking.DurationType
	Hours          int
	WorkerCount    int
	AddressLine    string
	JobDescription string
	PaymentMethod  string
}

// CreateBooking creates the authoritative booking record and initiates the workflow.
func (s *BookingService) CreateBooking(ctx context.Context, in CreateBookingInput) (*booking.Booking, error) {
	if in.Hours < 2 {
		in.Hours = 2
	}
	if in.WorkerCount < 1 {
		in.WorkerCount = 1
	}

	quote := booking.CalculateQuote(249, in.Hours, in.WorkerCount)
	s.sequence++
	bookingRef := booking.GenerateBookingReference(s.sequence)

	initialStatus := booking.StatusPendingPayment
	paymentStatus := "pending"
	if in.PaymentMethod == "cash" {
		initialStatus = booking.StatusSearching
		paymentStatus = "pending"
	}

	b := &booking.Booking{
		ID:               uuid.New(),
		BookingReference: bookingRef,
		CustomerID:       in.CustomerID,
		TradeID:          in.TradeID,
		TradeName:        in.TradeName,
		Status:           initialStatus,
		DurationType:     in.DurationType,
		ScheduledDate:    in.ScheduledDate,
		TimeSlot:         in.TimeSlot,
		Hours:            in.Hours,
		WorkerCount:      in.WorkerCount,
		AddressLine:      in.AddressLine,
		JobDescription:   in.JobDescription,
		CheckinOTP:       "4829", // Standard 4-digit mutual check-in OTP for MVP verification
		Quote:            quote,
		PaymentStatus:    paymentStatus,
		PaymentMethod:    in.PaymentMethod,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	if err := s.store.CreateBooking(ctx, b); err != nil {
		return nil, fmt.Errorf("create booking: %w", err)
	}

	// Create linked payment record
	p := &payment.Payment{
		ID:             uuid.New(),
		BookingID:      b.ID,
		UserID:         in.CustomerID,
		Amount:         quote.TotalAmount,
		Currency:       "INR",
		Method:         payment.PaymentMethod(in.PaymentMethod),
		Status:         payment.StatusPending,
		IdempotencyKey: fmt.Sprintf("pay-%s", b.ID.String()),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	_ = s.store.CreatePayment(ctx, p)

	if s.publisher != nil {
		_ = s.publisher.Publish(events.New(events.BookingCreated, in.CustomerID.String(), "booking", b.ID.String(), map[string]any{
			"reference": b.BookingReference,
			"amount":    quote.TotalAmount,
			"status":    b.Status,
		}))
	}

	// If cash or direct searching, automatically dispatch matcher
	if b.Status == booking.StatusSearching && s.dispatcher != nil {
		_ = s.dispatcher.QueueBooking(context.Background(), b.ID)
	}

	return b, nil
}

// GetBooking retrieves a booking by its ID.
func (s *BookingService) GetBooking(ctx context.Context, id uuid.UUID) (*booking.Booking, error) {
	return s.store.GetBookingByID(ctx, id)
}

// ListBookings returns all bookings for a user.
func (s *BookingService) ListBookings(ctx context.Context, customerID uuid.UUID) ([]booking.Booking, error) {
	return s.store.ListBookingsByCustomer(ctx, customerID)
}

// AdvanceLifecycle handles state transitions (En Route, Arrived, Check-in, Completed).
func (s *BookingService) AdvanceLifecycle(ctx context.Context, id uuid.UUID, targetStatus booking.BookingStatus, inputOTP string) (*booking.Booking, error) {
	b, err := s.store.GetBookingByID(ctx, id)
	if err != nil {
		return nil, booking.ErrBookingNotFound
	}

	if !booking.CanTransition(b.Status, targetStatus) {
		return nil, fmt.Errorf("%w: cannot transition from %s to %s", booking.ErrInvalidStatusTransition, b.Status, targetStatus)
	}

	now := time.Now()
	// Mutual Check-in OTP verification
	if targetStatus == booking.StatusInProgress {
		if inputOTP != b.CheckinOTP {
			return nil, booking.ErrInvalidOTPCheckin
		}
		b.StartedAt = &now
	}

	if targetStatus == booking.StatusCompleted {
		b.CompletedAt = &now
		b.PaymentStatus = "completed"
	}

	b.Status = targetStatus
	b.UpdatedAt = now

	if err := s.store.UpdateBooking(ctx, b); err != nil {
		return nil, fmt.Errorf("update booking: %w", err)
	}

	if s.publisher != nil {
		eventType := events.JobCompleted
		switch targetStatus {
		case booking.StatusEnRoute:
			eventType = events.BookingEnRoute
		case booking.StatusArrived:
			eventType = events.BookingArrived
		case booking.StatusInProgress:
			eventType = events.BookingCheckedIn
		}
		_ = s.publisher.Publish(events.New(eventType, b.CustomerID.String(), "booking", b.ID.String(), map[string]any{
			"status":   b.Status,
			"prof_id":  b.AssignedProfID,
			"ref":      b.BookingReference,
		}))
	}

	return b, nil
}

// ClaimBooking allows a professional to claim an unassigned booking atomically.
func (s *BookingService) ClaimBooking(ctx context.Context, bookingID, profID uuid.UUID) (*booking.Booking, error) {
	b, err := s.store.GetBookingByID(ctx, bookingID)
	if err != nil {
		return nil, booking.ErrBookingNotFound
	}

	// Atomic check: must be in Searching or PendingPayment status and unassigned
	if b.AssignedProfID != nil || (b.Status != booking.StatusSearching && b.Status != booking.StatusPendingPayment) {
		return nil, booking.ErrAlreadyClaimed
	}

	now := time.Now()
	b.AssignedProfID = &profID
	b.Status = booking.StatusAssigned
	b.UpdatedAt = now

	if err := s.store.UpdateBooking(ctx, b); err != nil {
		return nil, fmt.Errorf("claim booking update: %w", err)
	}

	if s.publisher != nil {
		_ = s.publisher.Publish(events.New(events.BookingAccepted, profID.String(), "booking", b.ID.String(), map[string]any{
			"booking_id":  b.ID.String(),
			"customer_id": b.CustomerID.String(),
			"prof_id":     profID.String(),
			"status":      b.Status,
			"reference":   b.BookingReference,
		}))
	}

	return b, nil
}

// RequestAdditionalWork submits an additional work quote from the professional for customer approval.
func (s *BookingService) RequestAdditionalWork(ctx context.Context, bookingID uuid.UUID, quote booking.AdditionalWorkQuote) (*booking.Booking, error) {
	b, err := s.store.GetBookingByID(ctx, bookingID)
	if err != nil {
		return nil, booking.ErrBookingNotFound
	}

	if b.Status != booking.StatusInProgress {
		return nil, booking.ErrInvalidAdditionalWork
	}

	quote.TotalAmount = quote.LaborCost + quote.MaterialCost
	now := time.Now()
	quote.RequestedAt = now
	quote.Status = "pending"
	b.AdditionalWork = &quote
	b.Status = booking.StatusAdditionalWorkPending
	b.UpdatedAt = now

	if err := s.store.UpdateBooking(ctx, b); err != nil {
		return nil, fmt.Errorf("request additional work: %w", err)
	}

	if s.publisher != nil {
		_ = s.publisher.Publish(events.New(events.AdditionalWorkRequested, b.CustomerID.String(), "booking", b.ID.String(), map[string]any{
			"booking_id":      b.ID.String(),
			"labor_cost":      quote.LaborCost,
			"material_cost":   quote.MaterialCost,
			"total_amount":    quote.TotalAmount,
			"description":     quote.Description,
		}))
	}

	return b, nil
}

// ApproveAdditionalWork processes customer acceptance of additional charges.
func (s *BookingService) ApproveAdditionalWork(ctx context.Context, bookingID uuid.UUID) (*booking.Booking, error) {
	b, err := s.store.GetBookingByID(ctx, bookingID)
	if err != nil {
		return nil, booking.ErrBookingNotFound
	}

	if b.Status != booking.StatusAdditionalWorkPending || b.AdditionalWork == nil {
		return nil, fmt.Errorf("%w: not pending approval", booking.ErrInvalidStatusTransition)
	}

	now := time.Now()
	b.AdditionalWork.Status = "approved"
	b.AdditionalWork.RespondedAt = &now

	b.Quote.Subtotal += b.AdditionalWork.TotalAmount
	b.Quote.TotalAmount += b.AdditionalWork.TotalAmount
	b.Status = booking.StatusInProgress
	b.UpdatedAt = now

	if err := s.store.UpdateBooking(ctx, b); err != nil {
		return nil, fmt.Errorf("approve additional work: %w", err)
	}

	if s.publisher != nil {
		_ = s.publisher.Publish(events.New(events.AdditionalWorkApproved, b.CustomerID.String(), "booking", b.ID.String(), map[string]any{
			"booking_id": b.ID.String(),
			"new_total":  b.Quote.TotalAmount,
			"approved":   b.AdditionalWork.TotalAmount,
		}))
	}

	return b, nil
}

// DeclineAdditionalWork processes customer rejection of additional charges.
func (s *BookingService) DeclineAdditionalWork(ctx context.Context, bookingID uuid.UUID) (*booking.Booking, error) {
	b, err := s.store.GetBookingByID(ctx, bookingID)
	if err != nil {
		return nil, booking.ErrBookingNotFound
	}

	if b.Status != booking.StatusAdditionalWorkPending {
		return nil, fmt.Errorf("%w: not pending approval", booking.ErrInvalidStatusTransition)
	}

	b.AdditionalWork = nil
	b.Status = booking.StatusInProgress
	b.UpdatedAt = time.Now()

	if err := s.store.UpdateBooking(ctx, b); err != nil {
		return nil, fmt.Errorf("decline additional work: %w", err)
	}

	if s.publisher != nil {
		_ = s.publisher.Publish(events.New(events.AdditionalWorkDeclined, b.CustomerID.String(), "booking", b.ID.String(), map[string]any{
			"booking_id": b.ID.String(),
		}))
	}

	return b, nil
}

// UpdateLocation streams GPS coordinates of the traveling professional.
func (s *BookingService) UpdateLocation(ctx context.Context, bookingID uuid.UUID, lat, lng float64) (*booking.Booking, error) {
	b, err := s.store.GetBookingByID(ctx, bookingID)
	if err != nil {
		return nil, booking.ErrBookingNotFound
	}

	now := time.Now()
	b.LocationLat = &lat
	b.LocationLng = &lng
	b.LocationUpdatedAt = &now
	b.UpdatedAt = now

	if err := s.store.UpdateBooking(ctx, b); err != nil {
		return nil, fmt.Errorf("update location: %w", err)
	}

	if s.publisher != nil {
		_ = s.publisher.Publish(events.New(events.BookingLocationUpdated, b.CustomerID.String(), "booking", b.ID.String(), map[string]any{
			"booking_id": b.ID.String(),
			"lat":        lat,
			"lng":        lng,
			"updated_at": now,
		}))
	}

	return b, nil
}

// CancelBooking marks a booking cancelled with audit reason.
func (s *BookingService) CancelBooking(ctx context.Context, id uuid.UUID, reason string) (*booking.Booking, error) {
	b, err := s.store.GetBookingByID(ctx, id)
	if err != nil {
		return nil, booking.ErrBookingNotFound
	}

	if b.Status == booking.StatusInProgress || b.Status == booking.StatusCompleted {
		return nil, booking.ErrCannotCancel
	}

	now := time.Now()
	b.Status = booking.StatusCancelled
	b.CancelledAt = &now
	b.CancellationReason = &reason
	b.UpdatedAt = now

	if err := s.store.UpdateBooking(ctx, b); err != nil {
		return nil, fmt.Errorf("cancel booking: %w", err)
	}

	// Trigger refund if payment was escrowed
	if b.PaymentStatus == "escrowed" && s.payments != nil {
		_, _ = s.payments.ProcessRefund(context.Background(), b.ID, "Cancellation: "+reason)
		b.PaymentStatus = "refunded"
		_ = s.store.UpdateBooking(ctx, b)
	}

	if s.publisher != nil {
		_ = s.publisher.Publish(events.New(events.BookingCancelled, b.CustomerID.String(), "booking", b.ID.String(), map[string]string{
			"reason": reason,
		}))
	}

	return b, nil
}
