package tests

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/nenav/nenav/backend/internal/domain/booking"
	"github.com/nenav/nenav/backend/internal/domain/payment"
	"github.com/nenav/nenav/backend/internal/domain/prof"
	"github.com/nenav/nenav/backend/internal/events"
	"github.com/nenav/nenav/backend/internal/repository"
	"github.com/nenav/nenav/backend/internal/service"
)

func TestPricingEngine(t *testing.T) {
	// Base wage for 2 hours @ 249/hr with 1 worker:
	// Base wage = 249 * 2 * 1 = 498
	// Platform fee (10%) = 49
	// Safety cover = 39
	// Subtotal = 498 + 49 + 39 = 586
	// GST (18%) = 105
	// Total = 586 + 105 = 691
	quote := booking.CalculateQuote(249, 2, 1)

	if quote.BaseWage != 498 {
		t.Errorf("expected BaseWage 498, got %d", quote.BaseWage)
	}
	if quote.PlatformFee != 49 {
		t.Errorf("expected PlatformFee 49, got %d", quote.PlatformFee)
	}
	if quote.SafetyCover != 39 {
		t.Errorf("expected SafetyCover 39, got %d", quote.SafetyCover)
	}
	if quote.Subtotal != 586 {
		t.Errorf("expected Subtotal 586, got %d", quote.Subtotal)
	}
	if quote.GST != 105 {
		t.Errorf("expected GST 105, got %d", quote.GST)
	}
	if quote.TotalAmount != 691 {
		t.Errorf("expected TotalAmount 691, got %d", quote.TotalAmount)
	}
}

func TestBookingStateMachine(t *testing.T) {
	// Valid transitions
	if !booking.CanTransition(booking.StatusDraft, booking.StatusPendingPayment) {
		t.Errorf("draft -> pending_payment should be valid")
	}
	if !booking.CanTransition(booking.StatusPendingPayment, booking.StatusSearching) {
		t.Errorf("pending_payment -> searching should be valid")
	}
	if !booking.CanTransition(booking.StatusSearching, booking.StatusAssigned) {
		t.Errorf("searching -> assigned should be valid")
	}
	if !booking.CanTransition(booking.StatusAssigned, booking.StatusEnRoute) {
		t.Errorf("assigned -> en_route should be valid")
	}
	if !booking.CanTransition(booking.StatusEnRoute, booking.StatusArrived) {
		t.Errorf("en_route -> arrived should be valid")
	}
	if !booking.CanTransition(booking.StatusArrived, booking.StatusInProgress) {
		t.Errorf("arrived -> in_progress should be valid")
	}
	if !booking.CanTransition(booking.StatusInProgress, booking.StatusCompleted) {
		t.Errorf("in_progress -> completed should be valid")
	}

	// Invalid transitions
	if booking.CanTransition(booking.StatusDraft, booking.StatusCompleted) {
		t.Errorf("draft -> completed should be invalid")
	}
	if booking.CanTransition(booking.StatusCancelled, booking.StatusInProgress) {
		t.Errorf("cancelled -> in_progress should be invalid")
	}
}

func TestMatchingAlgorithm15kmRadius(t *testing.T) {
	tradeElectrician := uuid.MustParse("10000000-0000-0000-0000-000000000001")
	custLat := 17.385044 // Charminar
	custLon := 78.486671

	profs := []prof.ProfProfile{
		{
			ID:              uuid.New(),
			TradeID:         tradeElectrician,
			Name:            "Nearby Worker",
			KYCStatus:       prof.KYCVerified,
			IsAvailable:     true,
			Rating:          4.8,
			ExperienceYears: 5,
			Latitude:        17.390000, // ~1 km away
			Longitude:       78.490000,
		},
		{
			ID:              uuid.New(),
			TradeID:         tradeElectrician,
			Name:            "Far Worker",
			KYCStatus:       prof.KYCVerified,
			IsAvailable:     true,
			Rating:          5.0,
			ExperienceYears: 10,
			Latitude:        17.700000, // ~35 km away (outside 15 km Hyderabad cutoff)
			Longitude:       78.700000,
		},
	}

	ranked := prof.RankPROFs(profs, tradeElectrician, custLat, custLon, 15.0)
	if len(ranked) != 1 {
		t.Fatalf("expected exactly 1 candidate within 15km radius, got %d", len(ranked))
	}
	if ranked[0].Name != "Nearby Worker" {
		t.Errorf("expected Nearby Worker, got %s", ranked[0].Name)
	}
}

func TestAuthServiceOTPAndSession(t *testing.T) {
	ctx := context.Background()
	store := repository.NewMemoryStore()
	bus := events.NewBus(nil)
	auth := service.NewAuthService(store, bus, "test-salt")

	phone := "9876543210"
	code, err := auth.SendOTP(ctx, "+91", phone, "login")
	if err != nil {
		t.Fatalf("send otp failed: %v", err)
	}

	if len(code) != 6 {
		t.Errorf("expected 6-digit otp, got %s", code)
	}

	// Verify with wrong code should fail
	_, _, err = auth.VerifyOTP(ctx, "+91", phone, "000000")
	if err == nil {
		t.Fatalf("expected error on wrong OTP, got nil")
	}

	// Verify with correct code
	user, session, err := auth.VerifyOTP(ctx, "+91", phone, code)
	if err != nil {
		t.Fatalf("verify otp failed: %v", err)
	}
	if user.Phone != phone {
		t.Errorf("expected user phone %s, got %s", phone, user.Phone)
	}
	if session.Token == "" {
		t.Errorf("expected session token, got empty string")
	}
}

func TestBilateralCashConfirmation(t *testing.T) {
	p := &payment.Payment{
		ID:        uuid.New(),
		BookingID: uuid.New(),
		UserID:    uuid.New(),
		Amount:    691,
		Currency:  "INR",
		Method:    payment.MethodCash,
		Status:    payment.StatusPending,
	}

	// Customer confirms
	both := p.ConfirmCash(true)
	if both {
		t.Errorf("should not be complete after only customer confirms")
	}
	if !p.CashCustomerConfirmed || p.CashProfConfirmed {
		t.Errorf("expected only customer confirmed")
	}

	// Worker confirms
	both = p.ConfirmCash(false)
	if !both {
		t.Errorf("should be complete after both parties confirm")
	}
	if p.Status != payment.StatusCaptured {
		t.Errorf("expected StatusCaptured, got %s", p.Status)
	}
}
