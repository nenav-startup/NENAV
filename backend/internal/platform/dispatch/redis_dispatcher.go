package dispatch

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/nenav/nenav/backend/internal/platform/cache"
)

const dispatchQueueChannel = "nenav:dispatch:queue"

type RedisDispatcher struct {
	redis  *cache.Redis
	logger *slog.Logger
}

func NewRedisDispatcher(r *cache.Redis, logger *slog.Logger) *RedisDispatcher {
	return &RedisDispatcher{
		redis:  r,
		logger: logger,
	}
}

// QueueBooking pushes the booking ID into the Redis pub/sub channel for background processing.
func (d *RedisDispatcher) QueueBooking(ctx context.Context, bookingID uuid.UUID) error {
	if d.redis == nil || d.redis.Client == nil {
		return fmt.Errorf("redis client not initialized")
	}

	err := d.redis.Client.Publish(ctx, dispatchQueueChannel, bookingID.String()).Err()
	if err != nil {
		d.logger.Error("failed to queue booking", "booking_id", bookingID, "error", err)
		return fmt.Errorf("queue booking: %w", err)
	}

	d.logger.Info("booking queued for matching", "booking_id", bookingID)
	return nil
}
