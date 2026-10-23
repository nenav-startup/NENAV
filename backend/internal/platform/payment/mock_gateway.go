package payment

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/nenav/nenav/backend/internal/domain/payment"
)

type MockRazorpayGateway struct{}

func NewMockRazorpayGateway() *MockRazorpayGateway {
	return &MockRazorpayGateway{}
}

func (m *MockRazorpayGateway) CreateOrder(ctx context.Context, amount int, currency, receipt string) (*payment.GatewayOrder, error) {
	return &payment.GatewayOrder{
		OrderID: fmt.Sprintf("order_%s", uuid.New().String()[:8]),
		Amount:  amount,
		Receipt: receipt,
	}, nil
}

func (m *MockRazorpayGateway) VerifySignature(ctx context.Context, orderID, paymentID, signature string) error {
	if signature == "invalid" {
		return errors.New("invalid signature")
	}
	return nil
}

func (m *MockRazorpayGateway) ProcessRefund(ctx context.Context, paymentID string, amount int) (*payment.GatewayRefund, error) {
	return &payment.GatewayRefund{
		RefundID: fmt.Sprintf("rfnd_%s", uuid.New().String()[:8]),
		Status:   "processed",
	}, nil
}
