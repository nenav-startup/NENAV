package payment

import "context"

type GatewayOrder struct {
	OrderID string
	Amount  int
	Receipt string
}

type GatewayRefund struct {
	RefundID string
	Status   string
}

// Gateway defines the contract for external payment processors (e.g. Razorpay, Cashfree).
type Gateway interface {
	CreateOrder(ctx context.Context, amount int, currency, receipt string) (*GatewayOrder, error)
	VerifySignature(ctx context.Context, orderID, paymentID, signature string) error
	ProcessRefund(ctx context.Context, paymentID string, amount int) (*GatewayRefund, error)
}
