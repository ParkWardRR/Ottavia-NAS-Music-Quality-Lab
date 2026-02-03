package jobs

import (
	"context"
	"database/sql"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/ottavia-music/ottavia/internal/analyzer"
	"github.com/ottavia-music/ottavia/internal/database"
	"github.com/ottavia-music/ottavia/internal/models"
)

type Worker struct {
	db          *database.DB
	analyzer    *analyzer.Analyzer
	workerCount int
	pollInterval time.Duration

	running   bool
	runningMu sync.Mutex
	cancel    context.CancelFunc
	wg        sync.WaitGroup
}

func NewWorker(db *database.DB, analyzer *analyzer.Analyzer, workerCount int) *Worker {
	return &Worker{
		db:           db,
		analyzer:     analyzer,
		workerCount:  workerCount,
		pollInterval: 5 * time.Second,
	}
}

func (w *Worker) Start(ctx context.Context) {
	w.runningMu.Lock()
	if w.running {
		w.runningMu.Unlock()
		return
	}
	w.running = true
	ctx, w.cancel = context.WithCancel(ctx)
	w.runningMu.Unlock()

	log.Info().Int("workers", w.workerCount).Msg("Starting job workers")

	// Start worker goroutines
	for i := 0; i < w.workerCount; i++ {
		w.wg.Add(1)
		go w.workerLoop(ctx, i)
	}
}

func (w *Worker) Stop() {
	w.runningMu.Lock()
	if !w.running {
		w.runningMu.Unlock()
		return
	}
	w.running = false
	w.runningMu.Unlock()

	if w.cancel != nil {
		w.cancel()
	}

	w.wg.Wait()
	log.Info().Msg("Job workers stopped")
}

func (w *Worker) workerLoop(ctx context.Context, id int) {
	defer w.wg.Done()

	log.Debug().Int("worker_id", id).Msg("Worker started")

	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Debug().Int("worker_id", id).Msg("Worker stopping")
			return
		case <-ticker.C:
			w.processNextJob(ctx, id)
		}
	}
}

func (w *Worker) processNextJob(ctx context.Context, workerID int) {
	// Try to get an analyze job
	job, err := w.db.GetNextJob(ctx, "analyze")
	if err != nil {
		if err != sql.ErrNoRows {
			log.Error().Err(err).Msg("Failed to get next job")
		}
		return
	}

	log.Info().
		Str("job_id", job.ID).
		Str("type", job.Type).
		Str("target", job.TargetID).
		Int("worker", workerID).
		Msg("Processing job")

	// Job is already marked as running by GetNextJob, just increment attempts
	job.Attempts++

	// Process based on type
	var processErr error
	switch job.Type {
	case "analyze":
		processErr = w.analyzer.AnalyzeFile(ctx, job.TargetID)
	default:
		log.Warn().Str("type", job.Type).Msg("Unknown job type")
		return
	}

	// Update job status
	if processErr != nil {
		log.Error().Err(processErr).Str("job_id", job.ID).Msg("Job failed")
		job.LastError = sql.NullString{String: processErr.Error(), Valid: true}

		if job.Attempts >= job.MaxAttempts {
			job.Status = models.StatusFailed
			job.FinishedAt = sql.NullTime{Time: time.Now(), Valid: true}
		} else {
			// Exponential backoff for retry
			backoff := time.Duration(1<<uint(job.Attempts)) * time.Minute
			job.Status = models.StatusQueued
			job.ScheduledAt = time.Now().Add(backoff)
		}
	} else {
		log.Info().Str("job_id", job.ID).Msg("Job completed")
		job.Status = models.StatusSuccess
		job.FinishedAt = sql.NullTime{Time: time.Now(), Valid: true}
	}

	w.db.UpdateJob(ctx, job)
}

// Scheduler handles periodic library scans
type Scheduler struct {
	db       *database.DB
	scanFunc func(ctx context.Context, libraryID string)

	running   bool
	runningMu sync.Mutex
	cancel    context.CancelFunc
}

func NewScheduler(db *database.DB, scanFunc func(ctx context.Context, libraryID string)) *Scheduler {
	return &Scheduler{
		db:       db,
		scanFunc: scanFunc,
	}
}

func (s *Scheduler) Start(ctx context.Context) {
	s.runningMu.Lock()
	if s.running {
		s.runningMu.Unlock()
		return
	}
	s.running = true
	ctx, s.cancel = context.WithCancel(ctx)
	s.runningMu.Unlock()

	log.Info().Msg("Starting scheduler")

	go s.schedulerLoop(ctx)
}

func (s *Scheduler) Stop() {
	s.runningMu.Lock()
	if !s.running {
		s.runningMu.Unlock()
		return
	}
	s.running = false
	s.runningMu.Unlock()

	if s.cancel != nil {
		s.cancel()
	}

	log.Info().Msg("Scheduler stopped")
}

func (s *Scheduler) schedulerLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.checkLibraries(ctx)
		}
	}
}

func (s *Scheduler) checkLibraries(ctx context.Context) {
	libraries, err := s.db.ListLibraries(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to list libraries for scheduling")
		return
	}

	now := time.Now()
	for _, lib := range libraries {
		if lib.Status == models.StatusRunning {
			continue
		}

		interval, err := time.ParseDuration(lib.ScanInterval)
		if err != nil {
			interval = 15 * time.Minute
		}

		var nextScan time.Time
		if lib.LastScanAt.Valid {
			nextScan = lib.LastScanAt.Time.Add(interval)
		} else {
			nextScan = lib.CreatedAt.Add(time.Minute) // First scan after 1 minute
		}

		if now.After(nextScan) {
			log.Info().Str("library", lib.Name).Msg("Triggering scheduled scan")
			go s.scanFunc(ctx, lib.ID)
		}
	}
}
