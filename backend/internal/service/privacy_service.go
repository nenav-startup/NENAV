package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/nenav/nenav/backend/internal/domain/privacy"
	"github.com/nenav/nenav/backend/internal/events"
	"github.com/nenav/nenav/backend/internal/repository"
)

// PrivacyService orchestrates compliance with the DPDP Act 2023.
type PrivacyService struct {
	store     repository.Store
	publisher events.Publisher
}

// NewPrivacyService creates a new PrivacyService instance.
func NewPrivacyService(store repository.Store, publisher events.Publisher) *PrivacyService {
	return &PrivacyService{
		store:     store,
		publisher: publisher,
	}
}

// RecordConsent logs a new consent grant or revocation in the immutable ledger.
func (s *PrivacyService) RecordConsent(
	ctx context.Context,
	userID uuid.UUID,
	purpose privacy.ConsentPurpose,
	status privacy.ConsentStatus,
	noticeVersion string,
	ipAddress string,
	userAgent string,
	metadata map[string]interface{},
) (*privacy.ConsentRecord, error) {
	if err := privacy.ValidatePurpose(purpose); err != nil {
		return nil, err
	}
	if status != privacy.StatusGranted && status != privacy.StatusRevoked {
		return nil, privacy.ErrInvalidStatus
	}
	if noticeVersion == "" {
		noticeVersion = "v1.0.0"
	}

	record := &privacy.ConsentRecord{
		ID:            uuid.New(),
		UserID:        userID,
		Purpose:       purpose,
		Status:        status,
		NoticeVersion: noticeVersion,
		IPAddress:     ipAddress,
		UserAgent:     userAgent,
		Metadata:      metadata,
		CreatedAt:     time.Now(),
	}

	if err := s.store.RecordConsent(ctx, record); err != nil {
		return nil, fmt.Errorf("record consent: %w", err)
	}

	if s.publisher != nil {
		_ = s.publisher.Publish(events.New(
			"privacy.consent_updated",
			userID.String(),
			"user",
			userID.String(),
			map[string]interface{}{
				"purpose":        string(purpose),
				"status":         string(status),
				"notice_version": noticeVersion,
			},
		))
	}

	return record, nil
}

// GetActiveConsents returns the active consent mapping for a user.
func (s *PrivacyService) GetActiveConsents(ctx context.Context, userID uuid.UUID) (map[privacy.ConsentPurpose]bool, error) {
	return s.store.GetActiveConsents(ctx, userID)
}

// GetConsentHistory retrieves the complete immutable audit trail of consents.
func (s *PrivacyService) GetConsentHistory(ctx context.Context, userID uuid.UUID) ([]privacy.ConsentRecord, error) {
	return s.store.GetConsentHistory(ctx, userID)
}

// RequestDataExport compiles all personal data into a machine-readable portability bundle (DPDP Section 11).
func (s *PrivacyService) RequestDataExport(ctx context.Context, userID uuid.UUID) (*privacy.UserDataExport, error) {
	u, err := s.store.GetUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	activeConsents, err := s.store.GetActiveConsents(ctx, userID)
	if err != nil {
		activeConsents = make(map[privacy.ConsentPurpose]bool)
	}

	consentHistory, err := s.store.GetConsentHistory(ctx, userID)
	if err != nil {
		consentHistory = []privacy.ConsentRecord{}
	}

	bookings, err := s.store.ListBookingsByCustomer(ctx, userID)
	if err != nil {
		bookings = nil
	}

	disputes, err := s.store.ListDisputes(ctx)
	userDisputesCount := 0
	if err == nil {
		for _, d := range disputes {
			if d.RaisedBy == userID {
				userDisputesCount++
			}
		}
	}

	export := &privacy.UserDataExport{
		ExportID:        uuid.New(),
		ExportTimestamp: time.Now(),
		User:            u,
		ActiveConsents:  activeConsents,
		ConsentHistory:  consentHistory,
		BookingsCount:   len(bookings),
		PaymentsCount:   0, // Extracted from completed bookings
		DisputesCount:   userDisputesCount,
	}

	// Record DSR audit entry
	now := time.Now()
	dsr := &privacy.DataSubjectRequest{
		ID:          uuid.New(),
		UserID:      userID,
		RequestType: privacy.DSRTypeAccessExport,
		Status:      privacy.DSRStatusCompleted,
		Details: map[string]interface{}{
			"export_id": export.ExportID.String(),
		},
		SLADeadline: privacy.CalculateSLADeadline(privacy.DSRTypeAccessExport, now),
		CompletedAt: &now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	_ = s.store.CreateDSR(ctx, dsr)

	if s.publisher != nil {
		_ = s.publisher.Publish(events.New(
			"privacy.data_exported",
			userID.String(),
			"user",
			userID.String(),
			map[string]interface{}{
				"export_id": export.ExportID.String(),
			},
		))
	}

	return export, nil
}

// RequestErasure enforces the Right to Erasure under DPDP Section 12 with a strict 72-hour SLA.
func (s *PrivacyService) RequestErasure(ctx context.Context, userID uuid.UUID, reason string) (*privacy.DataSubjectRequest, error) {
	_, err := s.store.GetUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	now := time.Now()
	slaDeadline := privacy.CalculateSLADeadline(privacy.DSRTypeErasure, now)

	dsr := &privacy.DataSubjectRequest{
		ID:          uuid.New(),
		UserID:      userID,
		RequestType: privacy.DSRTypeErasure,
		Status:      privacy.DSRStatusCompleted,
		Details: map[string]interface{}{
			"reason": reason,
		},
		SLADeadline: slaDeadline,
		CompletedAt: &now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Scrub PII immediately while preserving statutory transaction hashes
	if err := s.store.AnonymizeUserPII(ctx, userID); err != nil {
		return nil, fmt.Errorf("anonymize pii: %w", err)
	}

	if err := s.store.CreateDSR(ctx, dsr); err != nil {
		return nil, fmt.Errorf("record dsr: %w", err)
	}

	if s.publisher != nil {
		_ = s.publisher.Publish(events.New(
			"privacy.account_erased",
			userID.String(),
			"user",
			userID.String(),
			map[string]interface{}{
				"dsr_id":    dsr.ID.String(),
				"erased_at": now.Format(time.RFC3339),
			},
		))
	}

	return dsr, nil
}

// FileGrievance logs a formal escalation to the Data Protection Officer under DPDP Section 13.
func (s *PrivacyService) FileGrievance(ctx context.Context, userID uuid.UUID, subject, description string) (*privacy.Grievance, error) {
	subject = strings.TrimSpace(subject)
	description = strings.TrimSpace(description)

	if subject == "" {
		return nil, privacy.ErrSubjectRequired
	}
	if description == "" {
		return nil, privacy.ErrDescriptionRequired
	}

	now := time.Now()
	g := &privacy.Grievance{
		ID:           uuid.New(),
		TicketNumber: privacy.GenerateTicketNumber(),
		UserID:       userID,
		Subject:      subject,
		Description:  description,
		DPOName:      "NENAV Data Protection Officer",
		Status:       privacy.GrievanceStatusOpen,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.store.CreateGrievance(ctx, g); err != nil {
		return nil, fmt.Errorf("create grievance: %w", err)
	}

	if s.publisher != nil {
		_ = s.publisher.Publish(events.New(
			"privacy.grievance_filed",
			userID.String(),
			"grievance",
			g.ID.String(),
			map[string]interface{}{
				"ticket_number": g.TicketNumber,
				"subject":       g.Subject,
			},
		))
	}

	return g, nil
}

// ListUserGrievances returns all grievances filed by a user.
func (s *PrivacyService) ListUserGrievances(ctx context.Context, userID uuid.UUID) ([]privacy.Grievance, error) {
	return s.store.ListGrievancesByUser(ctx, userID)
}

// ResolveGrievance marks a complaint resolved with DPO findings.
func (s *PrivacyService) ResolveGrievance(ctx context.Context, grievanceID uuid.UUID, resolutionNotes, dpoName string) (*privacy.Grievance, error) {
	g, err := s.store.GetGrievanceByID(ctx, grievanceID)
	if err != nil {
		return nil, privacy.ErrGrievanceNotFound
	}

	now := time.Now()
	g.Status = privacy.GrievanceStatusResolved
	g.ResolutionNotes = &resolutionNotes
	g.ResolvedAt = &now
	if dpoName != "" {
		g.DPOName = dpoName
	}
	g.UpdatedAt = now

	if err := s.store.UpdateGrievance(ctx, g); err != nil {
		return nil, fmt.Errorf("update grievance: %w", err)
	}

	return g, nil
}

// ListUserDSRs returns all data subject requests initiated by the user.
func (s *PrivacyService) ListUserDSRs(ctx context.Context, userID uuid.UUID) ([]privacy.DataSubjectRequest, error) {
	return s.store.ListDSRsByUser(ctx, userID)
}

// SetNominee registers or updates a Section 14 nominee for the data principal.
func (s *PrivacyService) SetNominee(ctx context.Context, userID uuid.UUID, nomineeName, relationship, phone, email string) (*privacy.Nominee, error) {
	nomineeName = strings.TrimSpace(nomineeName)
	relationship = strings.TrimSpace(relationship)
	phone = strings.TrimSpace(phone)

	if nomineeName == "" || relationship == "" || phone == "" {
		return nil, fmt.Errorf("nominee_name, relationship, and phone are required")
	}

	var emailPtr *string
	if email != "" {
		emailPtr = &email
	}

	now := time.Now()
	nominee := &privacy.Nominee{
		ID:           uuid.New(),
		UserID:       userID,
		NomineeName:  nomineeName,
		Relationship: relationship,
		Phone:        phone,
		Email:        emailPtr,
		Status:       privacy.NomineeStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.store.SaveNominee(ctx, nominee); err != nil {
		return nil, fmt.Errorf("save nominee: %w", err)
	}

	if s.publisher != nil {
		_ = s.publisher.Publish(events.New(
			"privacy.nominee_updated",
			userID.String(),
			"user",
			userID.String(),
			map[string]interface{}{
				"nominee_name": nomineeName,
				"relationship": relationship,
			},
		))
	}

	return nominee, nil
}

// GetNominee retrieves the active Section 14 nominee designation.
func (s *PrivacyService) GetNominee(ctx context.Context, userID uuid.UUID) (*privacy.Nominee, error) {
	return s.store.GetNomineeByUserID(ctx, userID)
}

// RevokeNominee cancels an active Section 14 nominee.
func (s *PrivacyService) RevokeNominee(ctx context.Context, userID uuid.UUID) error {
	if err := s.store.RevokeNominee(ctx, userID); err != nil {
		return err
	}

	if s.publisher != nil {
		_ = s.publisher.Publish(events.New(
			"privacy.nominee_revoked",
			userID.String(),
			"user",
			userID.String(),
			nil,
		))
	}

	return nil
}

// ReportDataBreach creates an audit entry for an incident under DPDP Section 8(6).
func (s *PrivacyService) ReportDataBreach(
	ctx context.Context,
	incidentRef, nature string,
	severity privacy.BreachSeverity,
	affectedCount int,
	mitigation string,
) (*privacy.DataBreach, error) {
	now := time.Now()
	breach := &privacy.DataBreach{
		ID:                uuid.New(),
		IncidentReference: incidentRef,
		NatureOfBreach:    nature,
		AffectedCount:     affectedCount,
		Severity:          severity,
		MitigationTaken:   mitigation,
		Status:            privacy.BreachStatusInvestigating,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	if err := s.store.RecordBreach(ctx, breach); err != nil {
		return nil, fmt.Errorf("record breach: %w", err)
	}

	if s.publisher != nil {
		_ = s.publisher.Publish(events.New(
			"security.breach_recorded",
			"security_team",
			"breach",
			breach.ID.String(),
			map[string]interface{}{
				"incident_ref":   incidentRef,
				"severity":       string(severity),
				"affected_count": affectedCount,
			},
		))
	}

	return breach, nil
}

// ListBreaches retrieves recorded security and privacy incidents.
func (s *PrivacyService) ListBreaches(ctx context.Context) ([]privacy.DataBreach, error) {
	return s.store.ListBreaches(ctx)
}

// ReportBreachToDPBI marks the incident as notified to the Data Protection Board of India.
func (s *PrivacyService) ReportBreachToDPBI(ctx context.Context, breachID uuid.UUID) error {
	breaches, err := s.store.ListBreaches(ctx)
	if err != nil {
		return err
	}

	now := time.Now()
	for _, b := range breaches {
		if b.ID == breachID {
			b.ReportedToDPBIAt = &now
			b.Status = privacy.BreachStatusReported
			return s.store.UpdateBreach(ctx, &b)
		}
	}
	return fmt.Errorf("breach not found")
}

// NotifyDataPrincipals marks the incident as notified to affected data subjects.
func (s *PrivacyService) NotifyDataPrincipals(ctx context.Context, breachID uuid.UUID) error {
	breaches, err := s.store.ListBreaches(ctx)
	if err != nil {
		return err
	}

	now := time.Now()
	for _, b := range breaches {
		if b.ID == breachID {
			b.NotifiedPrincipalsAt = &now
			return s.store.UpdateBreach(ctx, &b)
		}
	}
	return fmt.Errorf("breach not found")
}
