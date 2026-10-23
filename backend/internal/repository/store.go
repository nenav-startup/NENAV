package repository

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/nenav/nenav/backend/internal/domain/booking"
	"github.com/nenav/nenav/backend/internal/domain/identity"
	"github.com/nenav/nenav/backend/internal/domain/payment"
	"github.com/nenav/nenav/backend/internal/domain/privacy"
	"github.com/nenav/nenav/backend/internal/domain/prof"
	"github.com/nenav/nenav/backend/internal/domain/support"
)

var (
	ErrNotFound = errors.New("record not found")
)

// Store defines all data access methods across NENAV domains.
type Store interface {
	// Identity & Auth
	CreateUser(ctx context.Context, u *identity.User) error
	GetUserByID(ctx context.Context, id uuid.UUID) (*identity.User, error)
	GetUserByPhone(ctx context.Context, countryCode, phone string) (*identity.User, error)
	CreateOTP(ctx context.Context, otp *identity.OTPRequest) error
	GetLatestOTP(ctx context.Context, countryCode, phone string) (*identity.OTPRequest, error)
	UpdateOTP(ctx context.Context, otp *identity.OTPRequest) error
	CreateSession(ctx context.Context, s *identity.Session) error
	GetSessionByTokenHash(ctx context.Context, tokenHash string) (*identity.Session, error)
	RevokeSession(ctx context.Context, id uuid.UUID) error

	// Prof & Trades
	ListPROFs(ctx context.Context) ([]prof.ProfProfile, error)
	GetPROFByID(ctx context.Context, id uuid.UUID) (*prof.ProfProfile, error)
	GetPROFByUserID(ctx context.Context, userID uuid.UUID) (*prof.ProfProfile, error)
	UpdatePROF(ctx context.Context, p *prof.ProfProfile) error
	CreatePROF(ctx context.Context, p *prof.ProfProfile) error
	SearchPROFs(ctx context.Context, lat, lon, radiusKm float64, tradeID uuid.UUID) ([]prof.ProfProfile, error)

	// Bookings
	CreateBooking(ctx context.Context, b *booking.Booking) error
	GetBookingByID(ctx context.Context, id uuid.UUID) (*booking.Booking, error)
	GetBookingByReference(ctx context.Context, ref string) (*booking.Booking, error)
	ListBookingsByCustomer(ctx context.Context, customerID uuid.UUID) ([]booking.Booking, error)
	ListStuckBookings(ctx context.Context, threshold time.Duration) ([]booking.Booking, error)
	ListBookingsByProf(ctx context.Context, profID uuid.UUID) ([]booking.Booking, error)
	ListAllBookings(ctx context.Context) ([]booking.Booking, error)
	UpdateBooking(ctx context.Context, b *booking.Booking) error

	// Payments
	CreatePayment(ctx context.Context, p *payment.Payment) error
	GetPaymentByID(ctx context.Context, id uuid.UUID) (*payment.Payment, error)
	GetPaymentByBookingID(ctx context.Context, bookingID uuid.UUID) (*payment.Payment, error)
	GetPaymentByIdempotency(ctx context.Context, key string) (*payment.Payment, error)
	UpdatePayment(ctx context.Context, p *payment.Payment) error

	// Disputes & Support
	CreateDispute(ctx context.Context, d *support.Dispute) error
	GetDisputeByID(ctx context.Context, id uuid.UUID) (*support.Dispute, error)
	ListDisputes(ctx context.Context) ([]support.Dispute, error)
	UpdateDispute(ctx context.Context, d *support.Dispute) error

	// DPDP Privacy & Compliance
	RecordConsent(ctx context.Context, record *privacy.ConsentRecord) error
	GetActiveConsents(ctx context.Context, userID uuid.UUID) (map[privacy.ConsentPurpose]bool, error)
	GetConsentHistory(ctx context.Context, userID uuid.UUID) ([]privacy.ConsentRecord, error)
	CreateDSR(ctx context.Context, dsr *privacy.DataSubjectRequest) error
	GetDSRByID(ctx context.Context, id uuid.UUID) (*privacy.DataSubjectRequest, error)
	ListDSRsByUser(ctx context.Context, userID uuid.UUID) ([]privacy.DataSubjectRequest, error)
	ListPendingDSRs(ctx context.Context) ([]privacy.DataSubjectRequest, error)
	UpdateDSR(ctx context.Context, dsr *privacy.DataSubjectRequest) error
	AnonymizeUserPII(ctx context.Context, userID uuid.UUID) error
	CreateGrievance(ctx context.Context, g *privacy.Grievance) error
	GetGrievanceByID(ctx context.Context, id uuid.UUID) (*privacy.Grievance, error)
	ListGrievancesByUser(ctx context.Context, userID uuid.UUID) ([]privacy.Grievance, error)
	UpdateGrievance(ctx context.Context, g *privacy.Grievance) error

	// DPDP Retention, Nominees, & Breaches
	PurgeExpiredOTPs(ctx context.Context) (int64, error)
	PurgeExpiredSessions(ctx context.Context) (int64, error)
	SaveNominee(ctx context.Context, n *privacy.Nominee) error
	GetNomineeByUserID(ctx context.Context, userID uuid.UUID) (*privacy.Nominee, error)
	RevokeNominee(ctx context.Context, userID uuid.UUID) error
	RecordBreach(ctx context.Context, b *privacy.DataBreach) error
	ListBreaches(ctx context.Context) ([]privacy.DataBreach, error)
	UpdateBreach(ctx context.Context, b *privacy.DataBreach) error
}

// MemoryStore provides a high-performance, thread-safe in-memory store for testing and fallback.
type MemoryStore struct {
	mu            sync.RWMutex
	users         map[uuid.UUID]*identity.User
	usersByPhone  map[string]*identity.User
	otps          map[string]*identity.OTPRequest
	sessions      map[string]*identity.Session
	profs         map[uuid.UUID]*prof.ProfProfile
	bookings      map[uuid.UUID]*booking.Booking
	payments      map[uuid.UUID]*payment.Payment
	paymentsByKey map[string]*payment.Payment
	disputes      map[uuid.UUID]*support.Dispute
	consents      []privacy.ConsentRecord
	dsrs          map[uuid.UUID]*privacy.DataSubjectRequest
	grievances    map[uuid.UUID]*privacy.Grievance
	accessLogs    []privacy.PrivacyAccessLog
	nominees      map[uuid.UUID]*privacy.Nominee
	breaches      map[uuid.UUID]*privacy.DataBreach
}

// NewMemoryStore initializes an empty memory store with seed data.
func NewMemoryStore() *MemoryStore {
	m := &MemoryStore{
		users:         make(map[uuid.UUID]*identity.User),
		usersByPhone:  make(map[string]*identity.User),
		otps:          make(map[string]*identity.OTPRequest),
		sessions:      make(map[string]*identity.Session),
		profs:         make(map[uuid.UUID]*prof.ProfProfile),
		bookings:      make(map[uuid.UUID]*booking.Booking),
		payments:      make(map[uuid.UUID]*payment.Payment),
		paymentsByKey: make(map[string]*payment.Payment),
		disputes:      make(map[uuid.UUID]*support.Dispute),
		consents:      make([]privacy.ConsentRecord, 0),
		dsrs:          make(map[uuid.UUID]*privacy.DataSubjectRequest),
		grievances:    make(map[uuid.UUID]*privacy.Grievance),
		accessLogs:    make([]privacy.PrivacyAccessLog, 0),
		nominees:      make(map[uuid.UUID]*privacy.Nominee),
		breaches:      make(map[uuid.UUID]*privacy.DataBreach),
	}
	m.seedDefaultData()
	return m
}

func (m *MemoryStore) seedDefaultData() {
	// Seed demo customer
	demoCustID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	demoCustomer := &identity.User{
		ID:               demoCustID,
		Phone:            "9876543210",
		PhoneCountryCode: "+91",
		DisplayName:      "Vikram Reddy",
		Role:             identity.RoleUser,
		Status:           identity.StatusActive,
		Locale:           "en-IN",
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
	m.users[demoCustID] = demoCustomer
	m.usersByPhone["+91:9876543210"] = demoCustomer

	// Seed 8 Trades with Verified PROFs in Hyderabad
	tradeElectrician := uuid.MustParse("10000000-0000-0000-0000-000000000001")
	tradePlumber := uuid.MustParse("10000000-0000-0000-0000-000000000002")
	tradeCarpenter := uuid.MustParse("10000000-0000-0000-0000-000000000003")
	tradePainter := uuid.MustParse("10000000-0000-0000-0000-000000000004")
	tradeMason := uuid.MustParse("10000000-0000-0000-0000-000000000005")
	tradeCleaner := uuid.MustParse("10000000-0000-0000-0000-000000000006")
	tradeLoader := uuid.MustParse("10000000-0000-0000-0000-000000000007")
	tradeGeneral := uuid.MustParse("10000000-0000-0000-0000-000000000008")

	seedProfs := []prof.ProfProfile{
		{
			ID:                  uuid.MustParse("20000000-0000-0000-0000-000000000001"),
			UserID:              uuid.New(),
			Name:                "Ramesh Naidu",
			Phone:               "9849012345",
			TradeID:             tradeElectrician,
			TradeName:           "Electrician",
			KYCStatus:           prof.KYCVerified,
			ExperienceYears:     8,
			Rating:              4.85,
			TotalReviews:        142,
			TotalJobs:           189,
			IsAvailable:         true,
			Latitude:            17.385044, // Charminar, Hyderabad
			Longitude:           78.486671,
			BadgePoliceVerified: true,
			BadgeSkillCertified: true,
			BadgeNenavPro:       true,
		},
		{
			ID:                  uuid.MustParse("20000000-0000-0000-0000-000000000002"),
			UserID:              uuid.New(),
			Name:                "Suresh Goud",
			Phone:               "9849012346",
			TradeID:             tradePlumber,
			TradeName:           "Plumber",
			KYCStatus:           prof.KYCVerified,
			ExperienceYears:     6,
			Rating:              4.72,
			TotalReviews:        98,
			TotalJobs:           125,
			IsAvailable:         true,
			Latitude:            17.440081, // Hitec City, Hyderabad
			Longitude:           78.348915,
			BadgePoliceVerified: true,
			BadgeSkillCertified: true,
			BadgeNenavPro:       true,
		},
		{
			ID:                  uuid.MustParse("20000000-0000-0000-0000-000000000003"),
			UserID:              uuid.New(),
			Name:                "Mahesh Chari",
			Phone:               "9849012347",
			TradeID:             tradeCarpenter,
			TradeName:           "Carpenter",
			KYCStatus:           prof.KYCVerified,
			ExperienceYears:     10,
			Rating:              4.90,
			TotalReviews:        210,
			TotalJobs:           264,
			IsAvailable:         true,
			Latitude:            17.439930, // Jubilee Hills, Hyderabad
			Longitude:           78.408282,
			BadgePoliceVerified: true,
			BadgeSkillCertified: true,
			BadgeNenavPro:       true,
		},
		{
			ID:                  uuid.MustParse("20000000-0000-0000-0000-000000000004"),
			UserID:              uuid.New(),
			Name:                "Kishore Yadav",
			Phone:               "9849012348",
			TradeID:             tradePainter,
			TradeName:           "Painter",
			KYCStatus:           prof.KYCVerified,
			ExperienceYears:     5,
			Rating:              4.68,
			TotalReviews:        82,
			TotalJobs:           104,
			IsAvailable:         true,
			Latitude:            17.494793, // Kukatpally, Hyderabad
			Longitude:           78.399643,
			BadgePoliceVerified: true,
			BadgeSkillCertified: false,
			BadgeNenavPro:       false,
		},
		{
			ID:                  uuid.MustParse("20000000-0000-0000-0000-000000000005"),
			UserID:              uuid.New(),
			Name:                "Balaraju M.",
			Phone:               "9849012349",
			TradeID:             tradeMason,
			TradeName:           "Mason",
			KYCStatus:           prof.KYCVerified,
			ExperienceYears:     12,
			Rating:              4.88,
			TotalReviews:        165,
			TotalJobs:           210,
			IsAvailable:         true,
			Latitude:            17.361564, // Dilsukhnagar, Hyderabad
			Longitude:           78.528435,
			BadgePoliceVerified: true,
			BadgeSkillCertified: true,
			BadgeNenavPro:       true,
		},
		{
			ID:                  uuid.MustParse("20000000-0000-0000-0000-000000000006"),
			UserID:              uuid.New(),
			Name:                "Anitha K.",
			Phone:               "9849012350",
			TradeID:             tradeCleaner,
			TradeName:           "Cleaner",
			KYCStatus:           prof.KYCVerified,
			ExperienceYears:     4,
			Rating:              4.92,
			TotalReviews:        114,
			TotalJobs:           148,
			IsAvailable:         true,
			Latitude:            17.437462, // Banjara Hills, Hyderabad
			Longitude:           78.448288,
			BadgePoliceVerified: true,
			BadgeSkillCertified: true,
			BadgeNenavPro:       true,
		},
		{
			ID:                  uuid.MustParse("20000000-0000-0000-0000-000000000007"),
			UserID:              uuid.New(),
			Name:                "Venkat Rao",
			Phone:               "9849012351",
			TradeID:             tradeLoader,
			TradeName:           "Loader / Unloader",
			KYCStatus:           prof.KYCVerified,
			ExperienceYears:     7,
			Rating:              4.78,
			TotalReviews:        90,
			TotalJobs:           115,
			IsAvailable:         true,
			Latitude:            17.443500, // Secunderabad
			Longitude:           78.498300,
			BadgePoliceVerified: true,
			BadgeSkillCertified: false,
			BadgeNenavPro:       false,
		},
		{
			ID:                  uuid.MustParse("20000000-0000-0000-0000-000000000008"),
			UserID:              uuid.New(),
			Name:                "Mallesh T.",
			Phone:               "9849012352",
			TradeID:             tradeGeneral,
			TradeName:           "General Labour",
			KYCStatus:           prof.KYCVerified,
			ExperienceYears:     3,
			Rating:              4.65,
			TotalReviews:        72,
			TotalJobs:           95,
			IsAvailable:         true,
			Latitude:            17.398400, // Mehdipatnam
			Longitude:           78.438200,
			BadgePoliceVerified: true,
			BadgeSkillCertified: false,
			BadgeNenavPro:       false,
		},
	}

	for _, p := range seedProfs {
		cp := p
		m.profs[cp.ID] = &cp
	}
}

// Implementations for Store methods:

func (m *MemoryStore) CreateUser(ctx context.Context, u *identity.User) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.users[u.ID] = u
	m.usersByPhone[u.PhoneCountryCode+":"+u.Phone] = u
	return nil
}

func (m *MemoryStore) GetUserByID(ctx context.Context, id uuid.UUID) (*identity.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	u, ok := m.users[id]
	if !ok {
		return nil, ErrNotFound
	}
	return u, nil
}

func (m *MemoryStore) GetUserByPhone(ctx context.Context, countryCode, phone string) (*identity.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	u, ok := m.usersByPhone[countryCode+":"+phone]
	if !ok {
		return nil, ErrNotFound
	}
	return u, nil
}

func (m *MemoryStore) CreateOTP(ctx context.Context, otp *identity.OTPRequest) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.otps[otp.PhoneCountryCode+":"+otp.Phone] = otp
	return nil
}

func (m *MemoryStore) GetLatestOTP(ctx context.Context, countryCode, phone string) (*identity.OTPRequest, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	otp, ok := m.otps[countryCode+":"+phone]
	if !ok {
		return nil, ErrNotFound
	}
	return otp, nil
}

func (m *MemoryStore) UpdateOTP(ctx context.Context, otp *identity.OTPRequest) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.otps[otp.PhoneCountryCode+":"+otp.Phone] = otp
	return nil
}

func (m *MemoryStore) CreateSession(ctx context.Context, s *identity.Session) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[s.TokenHash] = s
	return nil
}

func (m *MemoryStore) GetSessionByTokenHash(ctx context.Context, tokenHash string) (*identity.Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.sessions[tokenHash]
	if !ok {
		return nil, ErrNotFound
	}
	return s, nil
}

func (m *MemoryStore) RevokeSession(ctx context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	for _, s := range m.sessions {
		if s.ID == id {
			s.RevokedAt = &now
			return nil
		}
	}
	return ErrNotFound
}

func (m *MemoryStore) ListPROFs(ctx context.Context) ([]prof.ProfProfile, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var list []prof.ProfProfile
	for _, p := range m.profs {
		list = append(list, *p)
	}
	return list, nil
}

func (m *MemoryStore) SearchPROFs(ctx context.Context, lat, lon, radiusKm float64, tradeID uuid.UUID) ([]prof.ProfProfile, error) {
	m.mu.RLock()
	var list []prof.ProfProfile
	for _, p := range m.profs {
		list = append(list, *p)
	}
	m.mu.RUnlock()
	return prof.RankPROFs(list, tradeID, lat, lon, radiusKm), nil
}

func (m *MemoryStore) GetPROFByID(ctx context.Context, id uuid.UUID) (*prof.ProfProfile, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.profs[id]
	if !ok {
		return nil, ErrNotFound
	}
	return p, nil
}

func (m *MemoryStore) GetPROFByUserID(ctx context.Context, userID uuid.UUID) (*prof.ProfProfile, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, p := range m.profs {
		if p.UserID == userID {
			return p, nil
		}
	}
	return nil, ErrNotFound
}

func (m *MemoryStore) UpdatePROF(ctx context.Context, p *prof.ProfProfile) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.profs[p.ID] = p
	return nil
}

func (m *MemoryStore) CreatePROF(ctx context.Context, p *prof.ProfProfile) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.profs[p.ID] = p
	return nil
}

func (m *MemoryStore) CreateBooking(ctx context.Context, b *booking.Booking) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	bCopy := *b
	m.bookings[bCopy.ID] = &bCopy
	return nil
}

func (m *MemoryStore) GetBookingByID(ctx context.Context, id uuid.UUID) (*booking.Booking, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	b, ok := m.bookings[id]
	if !ok {
		return nil, ErrNotFound
	}
	bCopy := *b
	return &bCopy, nil
}

func (m *MemoryStore) GetBookingByReference(ctx context.Context, ref string) (*booking.Booking, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, b := range m.bookings {
		if b.BookingReference == ref {
			bCopy := *b
			return &bCopy, nil
		}
	}
	return nil, ErrNotFound
}

func (m *MemoryStore) ListBookingsByCustomer(ctx context.Context, customerID uuid.UUID) ([]booking.Booking, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var list []booking.Booking
	for _, b := range m.bookings {
		if b.CustomerID == customerID {
			list = append(list, *b)
		}
	}
	return list, nil
}

func (m *MemoryStore) ListStuckBookings(ctx context.Context, threshold time.Duration) ([]booking.Booking, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var stuck []booking.Booking
	cutoff := time.Now().Add(-threshold)
	for _, b := range m.bookings {
		if b.Status == booking.StatusSearching && b.CreatedAt.Before(cutoff) {
			stuck = append(stuck, *b)
		}
	}
	return stuck, nil
}

func (m *MemoryStore) ListBookingsByProf(ctx context.Context, profID uuid.UUID) ([]booking.Booking, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var list []booking.Booking
	for _, b := range m.bookings {
		if b.AssignedProfID != nil && *b.AssignedProfID == profID {
			list = append(list, *b)
		}
	}
	return list, nil
}

func (m *MemoryStore) ListAllBookings(ctx context.Context) ([]booking.Booking, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var list []booking.Booking
	for _, b := range m.bookings {
		list = append(list, *b)
	}
	return list, nil
}

func (m *MemoryStore) UpdateBooking(ctx context.Context, b *booking.Booking) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	bCopy := *b
	m.bookings[bCopy.ID] = &bCopy
	return nil
}

func (m *MemoryStore) CreatePayment(ctx context.Context, p *payment.Payment) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.payments[p.ID] = p
	if p.IdempotencyKey != "" {
		m.paymentsByKey[p.IdempotencyKey] = p
	}
	return nil
}

func (m *MemoryStore) GetPaymentByID(ctx context.Context, id uuid.UUID) (*payment.Payment, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.payments[id]
	if !ok {
		return nil, ErrNotFound
	}
	return p, nil
}

func (m *MemoryStore) GetPaymentByBookingID(ctx context.Context, bookingID uuid.UUID) (*payment.Payment, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, p := range m.payments {
		if p.BookingID == bookingID {
			return p, nil
		}
	}
	return nil, ErrNotFound
}

func (m *MemoryStore) GetPaymentByIdempotency(ctx context.Context, key string) (*payment.Payment, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.paymentsByKey[key]
	if !ok {
		return nil, ErrNotFound
	}
	return p, nil
}

func (m *MemoryStore) UpdatePayment(ctx context.Context, p *payment.Payment) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.payments[p.ID] = p
	return nil
}

func (m *MemoryStore) CreateDispute(ctx context.Context, d *support.Dispute) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.disputes[d.ID] = d
	return nil
}

func (m *MemoryStore) GetDisputeByID(ctx context.Context, id uuid.UUID) (*support.Dispute, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	d, ok := m.disputes[id]
	if !ok {
		return nil, ErrNotFound
	}
	return d, nil
}

func (m *MemoryStore) ListDisputes(ctx context.Context) ([]support.Dispute, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var list []support.Dispute
	for _, d := range m.disputes {
		list = append(list, *d)
	}
	return list, nil
}

func (m *MemoryStore) UpdateDispute(ctx context.Context, d *support.Dispute) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.disputes[d.ID] = d
	return nil
}

// ==========================================
// DPDP Privacy & Compliance Implementations
// ==========================================

func (m *MemoryStore) RecordConsent(ctx context.Context, record *privacy.ConsentRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if record.ID == uuid.Nil {
		record.ID = uuid.New()
	}
	if record.CreatedAt.IsZero() {
		record.CreatedAt = time.Now()
	}
	m.consents = append(m.consents, *record)
	return nil
}

func (m *MemoryStore) GetActiveConsents(ctx context.Context, userID uuid.UUID) (map[privacy.ConsentPurpose]bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	active := make(map[privacy.ConsentPurpose]bool)
	// Consents are append-only; process chronologically so later events overwrite earlier
	for _, c := range m.consents {
		if c.UserID == userID {
			active[c.Purpose] = (c.Status == privacy.StatusGranted)
		}
	}
	return active, nil
}

func (m *MemoryStore) GetConsentHistory(ctx context.Context, userID uuid.UUID) ([]privacy.ConsentRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var history []privacy.ConsentRecord
	for _, c := range m.consents {
		if c.UserID == userID {
			history = append(history, c)
		}
	}
	return history, nil
}

func (m *MemoryStore) CreateDSR(ctx context.Context, dsr *privacy.DataSubjectRequest) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if dsr.ID == uuid.Nil {
		dsr.ID = uuid.New()
	}
	now := time.Now()
	if dsr.CreatedAt.IsZero() {
		dsr.CreatedAt = now
	}
	dsr.UpdatedAt = now
	m.dsrs[dsr.ID] = dsr
	return nil
}

func (m *MemoryStore) GetDSRByID(ctx context.Context, id uuid.UUID) (*privacy.DataSubjectRequest, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	dsr, ok := m.dsrs[id]
	if !ok {
		return nil, ErrNotFound
	}
	return dsr, nil
}

func (m *MemoryStore) ListDSRsByUser(ctx context.Context, userID uuid.UUID) ([]privacy.DataSubjectRequest, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var list []privacy.DataSubjectRequest
	for _, d := range m.dsrs {
		if d.UserID == userID {
			list = append(list, *d)
		}
	}
	return list, nil
}

func (m *MemoryStore) ListPendingDSRs(ctx context.Context) ([]privacy.DataSubjectRequest, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var list []privacy.DataSubjectRequest
	for _, d := range m.dsrs {
		if d.Status == privacy.DSRStatusPending || d.Status == privacy.DSRStatusProcessing {
			list = append(list, *d)
		}
	}
	return list, nil
}

func (m *MemoryStore) UpdateDSR(ctx context.Context, dsr *privacy.DataSubjectRequest) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	dsr.UpdatedAt = time.Now()
	m.dsrs[dsr.ID] = dsr
	return nil
}

func (m *MemoryStore) AnonymizeUserPII(ctx context.Context, userID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	u, ok := m.users[userID]
	if !ok {
		return ErrNotFound
	}

	// Delete from phone lookup to prevent re-login
	delete(m.usersByPhone, u.PhoneCountryCode+u.Phone)

	// Anonymize PII fields under DPDP Section 12
	u.Phone = "0000000000"
	u.DisplayName = "Anonymized Data Subject"
	u.Email = nil
	u.Status = identity.StatusDeactivated
	u.UpdatedAt = time.Now()

	// Revoke active sessions immediately
	now := time.Now()
	for _, s := range m.sessions {
		if s.UserID == userID {
			s.RevokedAt = &now
		}
	}

	// Scrub prof details if applicable
	if p, ok := m.profs[userID]; ok {
		p.Name = "Anonymized Professional"
		p.Phone = "0000000000"
		p.IsAvailable = false
		p.UpdatedAt = now
	}

	return nil
}

func (m *MemoryStore) CreateGrievance(ctx context.Context, g *privacy.Grievance) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if g.ID == uuid.Nil {
		g.ID = uuid.New()
	}
	now := time.Now()
	if g.CreatedAt.IsZero() {
		g.CreatedAt = now
	}
	g.UpdatedAt = now
	m.grievances[g.ID] = g
	return nil
}

func (m *MemoryStore) GetGrievanceByID(ctx context.Context, id uuid.UUID) (*privacy.Grievance, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	g, ok := m.grievances[id]
	if !ok {
		return nil, ErrNotFound
	}
	return g, nil
}

func (m *MemoryStore) ListGrievancesByUser(ctx context.Context, userID uuid.UUID) ([]privacy.Grievance, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var list []privacy.Grievance
	for _, g := range m.grievances {
		if g.UserID == userID {
			list = append(list, *g)
		}
	}
	return list, nil
}

func (m *MemoryStore) UpdateGrievance(ctx context.Context, g *privacy.Grievance) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	g.UpdatedAt = time.Now()
	m.grievances[g.ID] = g
	return nil
}

// PurgeExpiredOTPs permanently deletes OTP challenges past their expiry under Section 8 data minimization.
func (m *MemoryStore) PurgeExpiredOTPs(ctx context.Context) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var purged int64
	now := time.Now()
	for key, otp := range m.otps {
		if now.After(otp.ExpiresAt) {
			delete(m.otps, key)
			purged++
		}
	}
	return purged, nil
}

// PurgeExpiredSessions revokes sessions that have passed their expiration timestamp.
func (m *MemoryStore) PurgeExpiredSessions(ctx context.Context) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var revokedCount int64
	now := time.Now()
	for _, s := range m.sessions {
		if now.After(s.ExpiresAt) && s.RevokedAt == nil {
			s.RevokedAt = &now
			revokedCount++
		}
	}
	return revokedCount, nil
}

// SaveNominee stores or updates a Section 14 nominee designation.
func (m *MemoryStore) SaveNominee(ctx context.Context, n *privacy.Nominee) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if n.ID == uuid.Nil {
		n.ID = uuid.New()
	}
	now := time.Now()
	if n.CreatedAt.IsZero() {
		n.CreatedAt = now
	}
	n.UpdatedAt = now
	m.nominees[n.UserID] = n
	return nil
}

// GetNomineeByUserID fetches an active nominee designation for a user.
func (m *MemoryStore) GetNomineeByUserID(ctx context.Context, userID uuid.UUID) (*privacy.Nominee, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	n, ok := m.nominees[userID]
	if !ok || n.Status != privacy.NomineeStatusActive {
		return nil, ErrNotFound
	}
	return n, nil
}

// RevokeNominee cancels an active Section 14 nominee designation.
func (m *MemoryStore) RevokeNominee(ctx context.Context, userID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	n, ok := m.nominees[userID]
	if !ok {
		return ErrNotFound
	}
	n.Status = privacy.NomineeStatusRevoked
	n.UpdatedAt = time.Now()
	return nil
}

// RecordBreach creates a new personal data breach record under DPDP Section 8(6).
func (m *MemoryStore) RecordBreach(ctx context.Context, b *privacy.DataBreach) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	now := time.Now()
	if b.CreatedAt.IsZero() {
		b.CreatedAt = now
	}
	b.UpdatedAt = now
	m.breaches[b.ID] = b
	return nil
}

// ListBreaches returns all recorded security and privacy incident records.
func (m *MemoryStore) ListBreaches(ctx context.Context) ([]privacy.DataBreach, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var list []privacy.DataBreach
	for _, b := range m.breaches {
		list = append(list, *b)
	}
	return list, nil
}

// UpdateBreach updates status, DPBI notification, or mitigation details for an incident.
func (m *MemoryStore) UpdateBreach(ctx context.Context, b *privacy.DataBreach) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	b.UpdatedAt = time.Now()
	m.breaches[b.ID] = b
	return nil
}

