package payment

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrDuplicateIdempotency = errors.New("duplicate request: payment with this idempotency key already processed")
	ErrInvalidPaymentMethod = errors.New("unsupported payment method")
	ErrPaymentAlreadySettled = errors.New("payment already settled")
	ErrBilateralUnconfirmed  = errors.New("cash payment requires bilateral confirmation from both customer and professional")
)

type PaymentMethod string

const (
	MethodUPI        PaymentMethod = "upi"
	MethodCard       PaymentMethod = "card"
	MethodNetBanking PaymentMethod = "netbanking"
	MethodCash       PaymentMethod = "cash"
)

type PaymentStatus string

const (
	StatusPending  PaymentStatus = "pending"
	StatusEscrowed PaymentStatus = "escrowed"
	StatusCaptured PaymentStatus = "captured"
	StatusFailed   PaymentStatus = "failed"
	StatusRefunded PaymentStatus = "refunded"
)

// Payment represents a financial transaction for a booking.
type Payment struct {
	ID                    uuid.UUID     `json:"id"`
	BookingID             uuid.UUID     `json:"booking_id"`
	UserID                uuid.UUID     `json:"user_id"`
	Amount                int           `json:"amount"` // in INR
	Currency              string        `json:"currency"`
	Method                PaymentMethod `json:"method"`
	Status                PaymentStatus `json:"status"`
	IdempotencyKey        string        `json:"idempotency_key"`
	GatewayOrderID        string        `json:"gateway_order_id,omitempty"`
	GatewayPaymentID      string        `json:"gateway_payment_id,omitempty"`
	CashCustomerConfirmed bool          `json:"cash_customer_confirmed"`
	CashProfConfirmed     bool          `json:"cash_prof_confirmed"`
	CreatedAt             time.Time     `json:"created_at"`
	UpdatedAt             time.Time     `json:"updated_at"`
}

// Refund represents an escrow or transaction reversal.
type Refund struct {
	ID              uuid.UUID `json:"id"`
	PaymentID       uuid.UUID `json:"payment_id"`
	BookingID       uuid.UUID `json:"booking_id"`
	Amount          int       `json:"amount"`
	Reason          string    `json:"reason"`
	Status          string    `json:"status"` // initiated, processed, failed
	GatewayRefundID string    `json:"gateway_refund_id,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

// Processor defines the internal interface for triggering refunds and payment actions.
type Processor interface {
	ProcessRefund(ctx context.Context, bookingID uuid.UUID, reason string) (*Refund, error)
}

// ConfirmCash records bilateral cash payment confirmation.
// Returns true if both parties have confirmed, advancing payment to Captured.
func (p *Payment) ConfirmCash(isCustomer bool) bool {
	if isCustomer {
		p.CashCustomerConfirmed = true
	} else {
		p.CashProfConfirmed = true
	}

	if p.CashCustomerConfirmed && p.CashProfConfirmed {
		p.Status = StatusCaptured
		p.UpdatedAt = time.Now()
		return true
	}
	return false
}
