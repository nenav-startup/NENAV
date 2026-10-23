package dispatch

import (
	"context"

	"github.com/google/uuid"
)

// Dispatcher handles the asynchronous queueing of booking matching.
type Dispatcher interface {
	QueueBooking(ctx context.Context, bookingID uuid.UUID) error
}
