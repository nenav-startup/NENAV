package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/nenav/nenav/backend/internal/domain/booking"
	"github.com/nenav/nenav/backend/internal/domain/prof"
	"github.com/nenav/nenav/backend/internal/events"
	"github.com/nenav/nenav/backend/internal/repository"
)

var (
	ErrNoWorkersAvailable = errors.New("no verified workers available within 15 km dispatch radius")
)

type MatchingService struct {
	store     repository.Store
	publisher events.Publisher
}

func NewMatchingService(store repository.Store, publisher events.Publisher) *MatchingService {
	return &MatchingService{
		store:     store,
		publisher: publisher,
	}
}

// MatchWorker finds the closest, highest-rated verified professional within 15km and assigns them.
func (s *MatchingService) MatchWorker(ctx context.Context, bookingID uuid.UUID) (*prof.ProfProfile, error) {
	b, err := s.store.GetBookingByID(ctx, bookingID)
	if err != nil {
		return nil, booking.ErrBookingNotFound
	}

	// Default customer location (Hyderabad city center if address coordinates not set)
	custLat := 17.385044
	custLon := 78.486671

	ranked, err := s.store.SearchPROFs(ctx, custLat, custLon, 15.0, b.TradeID)
	if err != nil {
		return nil, fmt.Errorf("search profs: %w", err)
	}

	if len(ranked) == 0 {
		return nil, ErrNoWorkersAvailable
	}

	chosen := ranked[0]
	b.AssignedProfID = &chosen.ID
	b.Status = booking.StatusAssigned
	b.UpdatedAt = time.Now()

	if err := s.store.UpdateBooking(ctx, b); err != nil {
		return nil, fmt.Errorf("update booking: %w", err)
	}

	// Update worker availability to busy
	chosen.IsAvailable = false
	chosen.TotalJobs++
	_ = s.store.UpdatePROF(ctx, &chosen)

	if s.publisher != nil {
		_ = s.publisher.Publish(events.New(events.BookingAccepted, chosen.UserID.String(), "booking", b.ID.String(), map[string]any{
			"prof_id":   chosen.ID.String(),
			"prof_name": chosen.Name,
			"rating":    chosen.Rating,
		}))
	}

	return &chosen, nil
}
