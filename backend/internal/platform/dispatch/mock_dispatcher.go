package dispatch

import (
	"context"

	"github.com/google/uuid"
	"github.com/nenav/nenav/backend/internal/service"
)

type MockDispatcher struct {
	matchingSvc *service.MatchingService
}

func NewMockDispatcher(matchingSvc *service.MatchingService) *MockDispatcher {
	return &MockDispatcher{
		matchingSvc: matchingSvc,
	}
}

func (d *MockDispatcher) QueueBooking(ctx context.Context, bookingID uuid.UUID) error {
	// Synchronously match the worker for testing purposes
	_, err := d.matchingSvc.MatchWorker(ctx, bookingID)
	return err
}
