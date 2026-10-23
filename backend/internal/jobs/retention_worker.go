package jobs

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/nenav/nenav/backend/internal/domain/privacy"
	"github.com/nenav/nenav/backend/internal/repository"
)

// SweepReport summarizes actions executed by a data retention sweep.
type SweepReport struct {
	PurgedOTPs        int64         `json:"purged_otps"`
	RevokedSessions   int64         `json:"revoked_sessions"`
	ProcessedErasures int64         `json:"processed_erasures"`
	ExecutedAt        time.Time     `json:"executed_at"`
	Duration          time.Duration `json:"duration"`
}

// RetentionWorker executes periodic automated data scrubbing and enforces DPDP retention schedules.
type RetentionWorker struct {
	store    repository.Store
	logger   *slog.Logger
	stopChan chan struct{}
	mu       sync.Mutex
	running  bool
}

// NewRetentionWorker initializes a retention job runner.
func NewRetentionWorker(store repository.Store, logger *slog.Logger) *RetentionWorker {
	return &RetentionWorker{
		store:    store,
		logger:   logger,
		stopChan: make(chan struct{}),
	}
}

// RunSweep performs a single pass over expired challenges, inactive sessions, and pending erasure DSRs.
func (w *RetentionWorker) RunSweep(ctx context.Context) (*SweepReport, error) {
	start := time.Now()
	report := &SweepReport{
		ExecutedAt: start,
	}

	// 1. Purge expired OTPs (Data Minimization: Section 8)
	purgedOTPs, err := w.store.PurgeExpiredOTPs(ctx)
	if err != nil && w.logger != nil {
		w.logger.Error("retention_sweep_otp_purge_failed", "error", err)
	} else {
		report.PurgedOTPs = purgedOTPs
	}

	// 2. Revoke expired sessions
	revokedSessions, err := w.store.PurgeExpiredSessions(ctx)
	if err != nil && w.logger != nil {
		w.logger.Error("retention_sweep_session_revoke_failed", "error", err)
	} else {
		report.RevokedSessions = revokedSessions
	}

	// 3. Process pending erasure requests approaching/past 72-hour statutory deadline
	pendingDSRs, err := w.store.ListPendingDSRs(ctx)
	if err == nil {
		now := time.Now()
		for _, dsr := range pendingDSRs {
			if dsr.RequestType == privacy.DSRTypeErasure {
				// Execute cascading PII anonymization
				if err := w.store.AnonymizeUserPII(ctx, dsr.UserID); err == nil {
					dsr.Status = privacy.DSRStatusCompleted
					dsr.CompletedAt = &now
					_ = w.store.UpdateDSR(ctx, &dsr)
					report.ProcessedErasures++
				}
			}
		}
	}

	report.Duration = time.Since(start)

	if w.logger != nil {
		w.logger.Info("retention_sweep_completed",
			"purged_otps", report.PurgedOTPs,
			"revoked_sessions", report.RevokedSessions,
			"processed_erasures", report.ProcessedErasures,
			"duration_ms", report.Duration.Milliseconds(),
		)
	}

	return report, nil
}

// Start begins the background cron ticker for automated sweeps.
func (w *RetentionWorker) Start(interval time.Duration) {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return
	}
	w.running = true
	w.stopChan = make(chan struct{})
	w.mu.Unlock()

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				_, _ = w.RunSweep(context.Background())
			case <-w.stopChan:
				return
			}
		}
	}()
}

// Stop terminates the background ticker gracefully.
func (w *RetentionWorker) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.running {
		return
	}
	w.running = false
	close(w.stopChan)
}
