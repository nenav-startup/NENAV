package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/nenav/nenav/backend/internal/domain/booking"
	"github.com/nenav/nenav/backend/internal/domain/dispatch"
	"github.com/nenav/nenav/backend/internal/domain/payment"
	"github.com/nenav/nenav/backend/internal/events"
	"github.com/nenav/nenav/backend/internal/repository"
)

type PaymentService struct {
	store      repository.Store
	dispatcher dispatch.Dispatcher
	gateway    payment.Gateway
	publisher  events.Publisher
}

func NewPaymentService(store repository.Store, dispatcher dispatch.Dispatcher, gateway payment.Gateway, publisher events.Publisher) *PaymentService {
	return &PaymentService{
		store:      store,
		dispatcher: dispatcher,
		gateway:    gateway,
		publisher:  publisher,
	}
}

// ConfirmOnlinePayment simulates gateway callback verification (UPI, NetBanking, Card).
func (s *PaymentService) ConfirmOnlinePayment(ctx context.Context, bookingID uuid.UUID, gatewayTxnID, signature string) (*payment.Payment, error) {
	p, err := s.store.GetPaymentByBookingID(ctx, bookingID)
	if err != nil {
		return nil, fmt.Errorf("payment not found for booking: %w", err)
	}

	if s.gateway != nil {
		if err := s.gateway.VerifySignature(ctx, p.GatewayOrderID, gatewayTxnID, signature); err != nil {
			return nil, fmt.Errorf("invalid payment signature: %w", err)
		}
	}

	p.Status = payment.StatusCaptured
	p.GatewayPaymentID = gatewayTxnID
	p.UpdatedAt = time.Now()

	if err := s.store.UpdatePayment(ctx, p); err != nil {
		return nil, err
	}

	// Advance booking from pending_payment to searching and trigger worker match
	b, err := s.store.GetBookingByID(ctx, bookingID)
	if err == nil && b.Status == booking.StatusPendingPayment {
		b.Status = booking.StatusSearching
		b.PaymentStatus = "escrowed"
		_ = s.store.UpdateBooking(ctx, b)

		if s.dispatcher != nil {
			_ = s.dispatcher.QueueBooking(context.Background(), b.ID)
		}
	}

	if s.publisher != nil {
		_ = s.publisher.Publish(events.New(events.PaymentCompleted, p.UserID.String(), "payment", p.ID.String(), map[string]any{
			"booking_id": bookingID.String(),
			"amount":     p.Amount,
		}))
	}

	return p, nil
}

// ConfirmCashPayment records bilateral confirmation. Advances to completed when both confirm.
func (s *PaymentService) ConfirmCashPayment(ctx context.Context, bookingID uuid.UUID, isCustomer bool) (*payment.Payment, bool, error) {
	p, err := s.store.GetPaymentByBookingID(ctx, bookingID)
	if err != nil {
		return nil, false, fmt.Errorf("payment not found: %w", err)
	}

	bothConfirmed := p.ConfirmCash(isCustomer)
	if err := s.store.UpdatePayment(ctx, p); err != nil {
		return nil, false, err
	}

	if bothConfirmed {
		// Both parties confirmed cash handover — advance booking to completed
		b, err := s.store.GetBookingByID(ctx, bookingID)
		if err == nil {
			now := time.Now()
			b.Status = booking.StatusCompleted
			b.PaymentStatus = "completed"
			b.CompletedAt = &now
			_ = s.store.UpdateBooking(ctx, b)

			if s.publisher != nil {
				_ = s.publisher.Publish(events.New(events.JobCompleted, b.CustomerID.String(), "booking", b.ID.String(), map[string]string{
					"settlement": "cash_bilateral",
				}))
			}
		}
	}

	return p, bothConfirmed, nil
}

// ProcessRefund initiates refund and records reversal ledger.
func (s *PaymentService) ProcessRefund(ctx context.Context, bookingID uuid.UUID, reason string) (*payment.Refund, error) {
	p, err := s.store.GetPaymentByBookingID(ctx, bookingID)
	if err != nil {
		return nil, errors.New("payment not found")
	}

	p.Status = payment.StatusRefunded
	p.UpdatedAt = time.Now()
	_ = s.store.UpdatePayment(ctx, p)

	var gatewayRefundID string
	var status = "processed"
	
	if s.gateway != nil && p.GatewayPaymentID != "" {
		gRefund, err := s.gateway.ProcessRefund(ctx, p.GatewayPaymentID, p.Amount)
		if err == nil {
			gatewayRefundID = gRefund.RefundID
			status = gRefund.Status
		} else {
			status = "failed"
		}
	} else {
		gatewayRefundID = fmt.Sprintf("rfnd-%s", uuid.New().String()[:8])
	}

	ref := &payment.Refund{
		ID:              uuid.New(),
		PaymentID:       p.ID,
		BookingID:       bookingID,
		Amount:          p.Amount,
		Reason:          reason,
		Status:          status,
		GatewayRefundID: gatewayRefundID,
		CreatedAt:       time.Now(),
	}

	if s.publisher != nil {
		_ = s.publisher.Publish(events.New(events.RefundCreated, p.UserID.String(), "refund", ref.ID.String(), map[string]any{
			"amount": ref.Amount,
			"reason": reason,
		}))
	}

	return ref, nil
}
