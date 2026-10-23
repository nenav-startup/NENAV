package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/nenav/nenav/backend/internal/domain/booking"
	"github.com/nenav/nenav/backend/internal/domain/support"
	"github.com/nenav/nenav/backend/internal/events"
	"github.com/nenav/nenav/backend/internal/repository"
)

type SupportService struct {
	store     repository.Store
	payments  *PaymentService
	publisher events.Publisher
}

func NewSupportService(store repository.Store, payments *PaymentService, publisher events.Publisher) *SupportService {
	return &SupportService{
		store:     store,
		payments:  payments,
		publisher: publisher,
	}
}

// CreateDispute opens a formal dispute on a booking and updates booking status.
func (s *SupportService) CreateDispute(ctx context.Context, bookingID, raisedBy uuid.UUID, reason support.DisputeReason, desc string, evidence []string) (*support.Dispute, error) {
	d := &support.Dispute{
		ID:           uuid.New(),
		BookingID:    bookingID,
		RaisedBy:     raisedBy,
		Reason:       reason,
		Description:  desc,
		EvidenceURLs: evidence,
		Status:       support.StatusOpen,
		CreatedAt:    time.Now(),
	}

	if err := s.store.CreateDispute(ctx, d); err != nil {
		return nil, err
	}

	// Update booking status to disputed
	b, err := s.store.GetBookingByID(ctx, bookingID)
	if err == nil {
		b.Status = booking.StatusDisputed
		b.UpdatedAt = time.Now()
		_ = s.store.UpdateBooking(ctx, b)
	}

	if s.publisher != nil {
		_ = s.publisher.Publish(events.New(events.DisputeOpened, raisedBy.String(), "dispute", d.ID.String(), map[string]any{
			"booking_id": bookingID.String(),
			"reason":     reason,
		}))
	}

	return d, nil
}

// ResolveDispute enables admin mediation and optional refund execution.
func (s *SupportService) ResolveDispute(ctx context.Context, id, adminID uuid.UUID, notes string, shouldRefund bool) (*support.Dispute, error) {
	d, err := s.store.GetDisputeByID(ctx, id)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	d.Status = support.StatusResolved
	d.ResolutionNotes = &notes
	d.ResolvedBy = &adminID
	d.ResolvedAt = &now

	if err := s.store.UpdateDispute(ctx, d); err != nil {
		return nil, err
	}

	if shouldRefund && s.payments != nil {
		_, _ = s.payments.ProcessRefund(ctx, d.BookingID, "Dispute resolved in customer favor: "+notes)
	}

	if s.publisher != nil {
		_ = s.publisher.Publish(events.New(events.DisputeResolved, adminID.String(), "dispute", d.ID.String(), map[string]any{
			"notes": notes,
		}))
	}

	return d, nil
}

// ListDisputes returns all disputes for internal admin review.
func (s *SupportService) ListDisputes(ctx context.Context) ([]support.Dispute, error) {
	return s.store.ListDisputes(ctx)
}
