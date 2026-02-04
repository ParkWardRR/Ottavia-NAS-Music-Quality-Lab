package jobs

import (
	"sync"
	"time"
)

// LogEntry represents a single log entry for a job
type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"` // info, warn, error, debug
	Module    string    `json:"module,omitempty"`
	Message   string    `json:"message"`
	Details   string    `json:"details,omitempty"`
}

// JobLog holds all log entries for a single job
type JobLog struct {
	JobID     string     `json:"jobId"`
	TrackID   string     `json:"trackId,omitempty"`
	Status    string     `json:"status"` // running, completed, failed
	StartedAt time.Time  `json:"startedAt"`
	EndedAt   *time.Time `json:"endedAt,omitempty"`
	Entries   []LogEntry `json:"entries"`
}

// Logger provides verbose logging for jobs
type Logger struct {
	mu   sync.RWMutex
	logs map[string]*JobLog
	// Keep only last N jobs to avoid memory bloat
	maxJobs int
	order   []string // Track order for cleanup
}

// NewLogger creates a new job logger
func NewLogger(maxJobs int) *Logger {
	if maxJobs <= 0 {
		maxJobs = 100
	}
	return &Logger{
		logs:    make(map[string]*JobLog),
		maxJobs: maxJobs,
		order:   make([]string, 0),
	}
}

// StartJob begins logging for a new job
func (l *Logger) StartJob(jobID, trackID string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Clean up old jobs if needed
	for len(l.order) >= l.maxJobs {
		oldID := l.order[0]
		l.order = l.order[1:]
		delete(l.logs, oldID)
	}

	l.logs[jobID] = &JobLog{
		JobID:     jobID,
		TrackID:   trackID,
		Status:    "running",
		StartedAt: time.Now(),
		Entries:   make([]LogEntry, 0),
	}
	l.order = append(l.order, jobID)

	l.addEntryLocked(jobID, "info", "", "Job started", "")
}

// EndJob marks a job as completed or failed
func (l *Logger) EndJob(jobID string, success bool, errorMsg string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if job, ok := l.logs[jobID]; ok {
		now := time.Now()
		job.EndedAt = &now
		if success {
			job.Status = "completed"
			l.addEntryLocked(jobID, "info", "", "Job completed successfully", "")
		} else {
			job.Status = "failed"
			l.addEntryLocked(jobID, "error", "", "Job failed", errorMsg)
		}
	}
}

// Log adds a log entry for a job
func (l *Logger) Log(jobID, level, module, message, details string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.addEntryLocked(jobID, level, module, message, details)
}

// Info logs an info message
func (l *Logger) Info(jobID, module, message string) {
	l.Log(jobID, "info", module, message, "")
}

// Debug logs a debug message
func (l *Logger) Debug(jobID, module, message, details string) {
	l.Log(jobID, "debug", module, message, details)
}

// Warn logs a warning message
func (l *Logger) Warn(jobID, module, message, details string) {
	l.Log(jobID, "warn", module, message, details)
}

// Error logs an error message
func (l *Logger) Error(jobID, module, message, details string) {
	l.Log(jobID, "error", module, message, details)
}

func (l *Logger) addEntryLocked(jobID, level, module, message, details string) {
	if job, ok := l.logs[jobID]; ok {
		job.Entries = append(job.Entries, LogEntry{
			Timestamp: time.Now(),
			Level:     level,
			Module:    module,
			Message:   message,
			Details:   details,
		})
	}
}

// GetLog returns the log for a specific job (returns interface{} for handler compatibility)
func (l *Logger) GetLog(jobID string) interface{} {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if job, ok := l.logs[jobID]; ok {
		// Return a copy
		entriesCopy := make([]LogEntry, len(job.Entries))
		copy(entriesCopy, job.Entries)
		return &JobLog{
			JobID:     job.JobID,
			TrackID:   job.TrackID,
			Status:    job.Status,
			StartedAt: job.StartedAt,
			EndedAt:   job.EndedAt,
			Entries:   entriesCopy,
		}
	}
	return nil
}

// GetLogSince returns log entries since a given index (returns interface{} for handler compatibility)
func (l *Logger) GetLogSince(jobID string, sinceIndex int) ([]interface{}, int, string) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if job, ok := l.logs[jobID]; ok {
		if sinceIndex < 0 {
			sinceIndex = 0
		}
		if sinceIndex >= len(job.Entries) {
			return []interface{}{}, len(job.Entries), job.Status
		}
		entries := job.Entries[sinceIndex:]
		result := make([]interface{}, len(entries))
		for i, e := range entries {
			result[i] = e
		}
		return result, len(job.Entries), job.Status
	}
	return nil, 0, ""
}

// GetRecentJobs returns the most recent job logs (returns interface{} for handler compatibility)
func (l *Logger) GetRecentJobs(limit int) interface{} {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if limit <= 0 || limit > len(l.order) {
		limit = len(l.order)
	}

	result := make([]*JobLog, 0, limit)
	for i := len(l.order) - 1; i >= 0 && len(result) < limit; i-- {
		if job, ok := l.logs[l.order[i]]; ok {
			result = append(result, &JobLog{
				JobID:     job.JobID,
				TrackID:   job.TrackID,
				Status:    job.Status,
				StartedAt: job.StartedAt,
				EndedAt:   job.EndedAt,
				Entries:   nil, // Don't include entries in list view
			})
		}
	}
	return result
}

// Global logger instance
var globalLogger = NewLogger(100)

// GetGlobalLogger returns the global job logger
func GetGlobalLogger() *Logger {
	return globalLogger
}
