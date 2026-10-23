package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/nenav/nenav/backend/internal/domain/privacy"
)

// RecordConsent inserts an immutable record into the consents table.
func (db *DB) RecordConsent(ctx context.Context, record *privacy.ConsentRecord) error {
	if record.ID == uuid.Nil {
		record.ID = uuid.New()
	}
	if record.CreatedAt.IsZero() {
		record.CreatedAt = time.Now()
	}
	metaJSON, err := json.Marshal(record.Metadata)
	if err != nil {
		metaJSON = []byte("{}")
	}

	query := `
		INSERT INTO consents (id, user_id, purpose, status, notice_version, ip_address, user_agent, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, NULLIF($6, '')::inet, $7, $8, $9)
	`
	_, err = db.Pool.Exec(ctx, query,
		record.ID,
		record.UserID,
		record.Purpose,
		record.Status,
		record.NoticeVersion,
		record.IPAddress,
		record.UserAgent,
		metaJSON,
		record.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("record consent: %w", err)
	}
	return nil
}

// GetActiveConsents retrieves the latest status for each purpose for the user.
func (db *DB) GetActiveConsents(ctx context.Context, userID uuid.UUID) (map[privacy.ConsentPurpose]bool, error) {
	query := `
		SELECT DISTINCT ON (purpose) purpose, status
		FROM consents
		WHERE user_id = $1
		ORDER BY purpose, created_at DESC
	`
	rows, err := db.Pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("get active consents: %w", err)
	}
	defer rows.Close()

	active := make(map[privacy.ConsentPurpose]bool)
	for rows.Next() {
		var purpose string
		var status string
		if err := rows.Scan(&purpose, &status); err != nil {
			return nil, fmt.Errorf("scan active consent: %w", err)
		}
		active[privacy.ConsentPurpose(purpose)] = (status == string(privacy.StatusGranted))
	}
	return active, nil
}

// CreateDSR creates a new statutory data subject request.
func (db *DB) CreateDSR(ctx context.Context, dsr *privacy.DataSubjectRequest) error {
	if dsr.ID == uuid.Nil {
		dsr.ID = uuid.New()
	}
	now := time.Now()
	if dsr.CreatedAt.IsZero() {
		dsr.CreatedAt = now
	}
	dsr.UpdatedAt = now

	detailsJSON, err := json.Marshal(dsr.Details)
	if err != nil {
		detailsJSON = []byte("{}")
	}

	query := `
		INSERT INTO data_subject_requests (
			id, user_id, request_type, status, details, sla_deadline, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err = db.Pool.Exec(ctx, query,
		dsr.ID,
		dsr.UserID,
		dsr.RequestType,
		dsr.Status,
		detailsJSON,
		dsr.SLADeadline,
		dsr.CreatedAt,
		dsr.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create dsr: %w", err)
	}
	return nil
}

// AnonymizeUserPII executes permanent PII scrubbing under DPDP Section 12 while maintaining financial integrity.
func (db *DB) AnonymizeUserPII(ctx context.Context, userID uuid.UUID) error {
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin anonymize tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Anonymize user table
	userQuery := `
		UPDATE users
		SET phone = '0000000000',
		    email = NULL,
		    display_name = 'Anonymized Data Subject',
		    status = 'deactivated',
		    updated_at = NOW()
		WHERE id = $1
	`
	if _, err := tx.Exec(ctx, userQuery, userID); err != nil {
		return fmt.Errorf("anonymize user: %w", err)
	}

	// Revoke sessions
	sessionQuery := `
		UPDATE sessions
		SET revoked_at = NOW()
		WHERE user_id = $1 AND revoked_at IS NULL
	`
	if _, err := tx.Exec(ctx, sessionQuery, userID); err != nil {
		return fmt.Errorf("revoke sessions: %w", err)
	}

	// Remove saved customer addresses (PII)
	addrQuery := `DELETE FROM addresses WHERE user_id = $1`
	if _, err := tx.Exec(ctx, addrQuery, userID); err != nil {
		return fmt.Errorf("delete addresses: %w", err)
	}

	return tx.Commit(ctx)
}

// CreateGrievance records a formal DPO complaint.
func (db *DB) CreateGrievance(ctx context.Context, g *privacy.Grievance) error {
	if g.ID == uuid.Nil {
		g.ID = uuid.New()
	}
	now := time.Now()
	if g.CreatedAt.IsZero() {
		g.CreatedAt = now
	}
	g.UpdatedAt = now

	query := `
		INSERT INTO grievances (
			id, ticket_number, user_id, subject, description, dpo_name, status, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := db.Pool.Exec(ctx, query,
		g.ID,
		g.TicketNumber,
		g.UserID,
		g.Subject,
		g.Description,
		g.DPOName,
		g.Status,
		g.CreatedAt,
		g.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create grievance: %w", err)
	}
	return nil
}

// GetGrievanceByID fetches a grievance by primary key.
func (db *DB) GetGrievanceByID(ctx context.Context, id uuid.UUID) (*privacy.Grievance, error) {
	query := `
		SELECT id, ticket_number, user_id, subject, description, dpo_name, status, resolution_notes, resolved_at, created_at, updated_at
		FROM grievances
		WHERE id = $1
	`
	g := &privacy.Grievance{}
	err := db.Pool.QueryRow(ctx, query, id).Scan(
		&g.ID,
		&g.TicketNumber,
		&g.UserID,
		&g.Subject,
		&g.Description,
		&g.DPOName,
		&g.Status,
		&g.ResolutionNotes,
		&g.ResolvedAt,
		&g.CreatedAt,
		&g.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("grievance not found")
		}
		return nil, fmt.Errorf("get grievance: %w", err)
	}
	return g, nil
}
