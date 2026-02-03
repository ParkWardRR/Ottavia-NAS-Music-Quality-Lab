package models

import (
	"database/sql"
	"encoding/json"
	"time"
)

// Library represents a music library root folder
type Library struct {
	ID            string         `db:"id" json:"id"`
	Name          string         `db:"name" json:"name"`
	RootPath      string         `db:"root_path" json:"rootPath"`
	ScanInterval  string         `db:"scan_interval" json:"scanInterval"`
	ReadOnly      bool           `db:"read_only" json:"readOnly"`
	OutputPath    sql.NullString `db:"output_path" json:"outputPath,omitempty"`
	AllowedFormats sql.NullString `db:"allowed_formats" json:"allowedFormats,omitempty"`
	LastScanAt    sql.NullTime   `db:"last_scan_at" json:"lastScanAt,omitempty"`
	Status        string         `db:"status" json:"status"`
	CreatedAt     time.Time      `db:"created_at" json:"createdAt"`
	UpdatedAt     time.Time      `db:"updated_at" json:"updatedAt"`

	// Computed fields (populated by queries with aggregates)
	TrackCount   int   `db:"track_count" json:"trackCount,omitempty"`
	IssueCount   int   `db:"issue_count" json:"issueCount,omitempty"`
	TotalSize    int64 `db:"total_size" json:"totalSize,omitempty"`
}

// MediaFile represents a file on disk
type MediaFile struct {
	ID          string         `db:"id" json:"id"`
	LibraryID   string         `db:"library_id" json:"libraryId"`
	Path        string         `db:"path" json:"path"`
	Filename    string         `db:"filename" json:"filename"`
	Extension   string         `db:"extension" json:"extension"`
	Size        int64          `db:"size" json:"size"`
	Mtime       time.Time      `db:"mtime" json:"mtime"`
	QuickHash   sql.NullString `db:"quick_hash" json:"quickHash,omitempty"`
	FullHash    sql.NullString `db:"full_hash" json:"fullHash,omitempty"`
	Status      string         `db:"status" json:"status"`
	ErrorMsg    sql.NullString `db:"error_msg" json:"errorMsg,omitempty"`
	CreatedAt   time.Time      `db:"created_at" json:"createdAt"`
	UpdatedAt   time.Time      `db:"updated_at" json:"updatedAt"`
}

// Track represents audio track metadata
type Track struct {
	ID            string         `db:"id" json:"id"`
	MediaFileID   string         `db:"media_file_id" json:"mediaFileId"`
	Duration      float64        `db:"duration" json:"duration"`
	Codec         string         `db:"codec" json:"codec"`
	SampleRate    int            `db:"sample_rate" json:"sampleRate"`
	BitDepth      int            `db:"bit_depth" json:"bitDepth"`
	Channels      int            `db:"channels" json:"channels"`
	Bitrate       int            `db:"bitrate" json:"bitrate"`

	// Metadata tags
	Title         sql.NullString `db:"title" json:"title,omitempty"`
	Artist        sql.NullString `db:"artist" json:"artist,omitempty"`
	Album         sql.NullString `db:"album" json:"album,omitempty"`
	AlbumArtist   sql.NullString `db:"album_artist" json:"albumArtist,omitempty"`
	TrackNumber   sql.NullInt32  `db:"track_number" json:"trackNumber,omitempty"`
	DiscNumber    sql.NullInt32  `db:"disc_number" json:"discNumber,omitempty"`
	Year          sql.NullInt32  `db:"year" json:"year,omitempty"`
	Genre         sql.NullString `db:"genre" json:"genre,omitempty"`

	HasArtwork    bool           `db:"has_artwork" json:"hasArtwork"`
	ArtworkWidth  sql.NullInt32  `db:"artwork_width" json:"artworkWidth,omitempty"`
	ArtworkHeight sql.NullInt32  `db:"artwork_height" json:"artworkHeight,omitempty"`

	CreatedAt     time.Time      `db:"created_at" json:"createdAt"`
	UpdatedAt     time.Time      `db:"updated_at" json:"updatedAt"`

	// Computed/joined fields (populated by queries with JOINs)
	Path          string         `db:"path" json:"path,omitempty"`
	LibraryID     string         `db:"library_id" json:"libraryId,omitempty"`
	LibraryName   string         `db:"library_name" json:"libraryName,omitempty"`
}

// ScanRun represents a library scan session
type ScanRun struct {
	ID           string       `db:"id" json:"id"`
	LibraryID    string       `db:"library_id" json:"libraryId"`
	Status       string       `db:"status" json:"status"`
	FilesFound   int          `db:"files_found" json:"filesFound"`
	FilesNew     int          `db:"files_new" json:"filesNew"`
	FilesChanged int          `db:"files_changed" json:"filesChanged"`
	FilesDeleted int          `db:"files_deleted" json:"filesDeleted"`
	FilesFailed  int          `db:"files_failed" json:"filesFailed"`
	StartedAt    time.Time    `db:"started_at" json:"startedAt"`
	FinishedAt   sql.NullTime `db:"finished_at" json:"finishedAt,omitempty"`
	ErrorMsg     sql.NullString `db:"error_msg" json:"errorMsg,omitempty"`
}

// AnalysisResult represents audio analysis results for a track
type AnalysisResult struct {
	ID             string    `db:"id" json:"id"`
	TrackID        string    `db:"track_id" json:"trackId"`
	Version        int       `db:"version" json:"version"`

	// Lossless detection
	LosslessScore  float64   `db:"lossless_score" json:"losslessScore"`
	LosslessStatus string    `db:"lossless_status" json:"losslessStatus"` // pass/warn/fail

	// Integrity
	IntegrityOK    bool      `db:"integrity_ok" json:"integrityOk"`
	DecodeErrors   int       `db:"decode_errors" json:"decodeErrors"`

	// Audio stats
	PeakLevel      float64   `db:"peak_level" json:"peakLevel"`
	TruePeak       float64   `db:"true_peak" json:"truePeak"`
	CrestFactor    float64   `db:"crest_factor" json:"crestFactor"`
	ClippedSamples int       `db:"clipped_samples" json:"clippedSamples"`
	DCOffset       float64   `db:"dc_offset" json:"dcOffset"`

	// Loudness
	IntegratedLoudness float64 `db:"integrated_loudness" json:"integratedLoudness"`
	LoudnessRange      float64 `db:"loudness_range" json:"loudnessRange"`

	// Frequency analysis
	HighFreqCutoff   float64 `db:"high_freq_cutoff" json:"highFreqCutoff"`
	SpectralRolloff  float64 `db:"spectral_rolloff" json:"spectralRolloff"`

	// Phase/stereo
	PhaseCorrelation float64 `db:"phase_correlation" json:"phaseCorrelation"`

	// Issues stored as JSON
	IssuesJSON     string    `db:"issues_json" json:"-"`
	StatsJSON      string    `db:"stats_json" json:"-"`

	CreatedAt      time.Time `db:"created_at" json:"createdAt"`

	// Parsed fields
	Issues         []Issue   `db:"-" json:"issues,omitempty"`
	Stats          map[string]interface{} `db:"-" json:"stats,omitempty"`
}

// Issue represents a detected problem
type Issue struct {
	Type        string  `json:"type"`
	Severity    string  `json:"severity"` // info/warning/error
	Message     string  `json:"message"`
	Confidence  float64 `json:"confidence"`
	ArtifactID  string  `json:"artifactId,omitempty"`
}

// Artifact represents evidence files (spectrograms, waveforms, etc.)
type Artifact struct {
	ID           string         `db:"id" json:"id"`
	TrackID      string         `db:"track_id" json:"trackId"`
	Type         string         `db:"type" json:"type"` // spectrogram/waveform/hf_energy/etc
	Path         string         `db:"path" json:"path"`
	MimeType     string         `db:"mime_type" json:"mimeType"`
	Width        sql.NullInt32  `db:"width" json:"width,omitempty"`
	Height       sql.NullInt32  `db:"height" json:"height,omitempty"`
	MetadataJSON sql.NullString `db:"metadata_json" json:"-"`
	CreatedAt    time.Time      `db:"created_at" json:"createdAt"`

	Metadata     map[string]interface{} `db:"-" json:"metadata,omitempty"`
}

// ActionLog represents a user or system action
type ActionLog struct {
	ID         string    `db:"id" json:"id"`
	Type       string    `db:"type" json:"type"` // tag_edit/convert/delete/etc
	TargetType string    `db:"target_type" json:"targetType"`
	TargetID   string    `db:"target_id" json:"targetId"`
	Actor      string    `db:"actor" json:"actor"`
	BeforeJSON string    `db:"before_json" json:"-"`
	AfterJSON  string    `db:"after_json" json:"-"`
	CreatedAt  time.Time `db:"created_at" json:"createdAt"`

	Before     map[string]interface{} `db:"-" json:"before,omitempty"`
	After      map[string]interface{} `db:"-" json:"after,omitempty"`
}

// ConversionJob represents a queued conversion task
type ConversionJob struct {
	ID          string         `db:"id" json:"id"`
	SourceType  string         `db:"source_type" json:"sourceType"` // track/album/library
	SourceID    string         `db:"source_id" json:"sourceId"`
	Profile     string         `db:"profile" json:"profile"`
	OutputPath  string         `db:"output_path" json:"outputPath"`
	Status      string         `db:"status" json:"status"` // queued/running/success/failed/cancelled
	Progress    float64        `db:"progress" json:"progress"`
	LogsPath    sql.NullString `db:"logs_path" json:"logsPath,omitempty"`
	ErrorMsg    sql.NullString `db:"error_msg" json:"errorMsg,omitempty"`
	QueuedAt    time.Time      `db:"queued_at" json:"queuedAt"`
	StartedAt   sql.NullTime   `db:"started_at" json:"startedAt,omitempty"`
	FinishedAt  sql.NullTime   `db:"finished_at" json:"finishedAt,omitempty"`
}

// Job represents a generic background job
type Job struct {
	ID          string         `db:"id" json:"id"`
	Type        string         `db:"type" json:"type"`
	TargetType  string         `db:"target_type" json:"targetType"`
	TargetID    string         `db:"target_id" json:"targetId"`
	Status      string         `db:"status" json:"status"`
	Priority    int            `db:"priority" json:"priority"`
	Attempts    int            `db:"attempts" json:"attempts"`
	MaxAttempts int            `db:"max_attempts" json:"maxAttempts"`
	LastError   sql.NullString `db:"last_error" json:"lastError,omitempty"`
	PayloadJSON sql.NullString `db:"payload_json" json:"-"`
	ScheduledAt time.Time      `db:"scheduled_at" json:"scheduledAt"`
	StartedAt   sql.NullTime   `db:"started_at" json:"startedAt,omitempty"`
	FinishedAt  sql.NullTime   `db:"finished_at" json:"finishedAt,omitempty"`
	CreatedAt   time.Time      `db:"created_at" json:"createdAt"`

	Payload     map[string]interface{} `db:"-" json:"payload,omitempty"`
}

// Settings represents user/app settings
type Setting struct {
	Key       string    `db:"key" json:"key"`
	Value     string    `db:"value" json:"value"`
	Type      string    `db:"type" json:"type"` // string/int/bool/json
	Category  string    `db:"category" json:"category"`
	UpdatedAt time.Time `db:"updated_at" json:"updatedAt"`
}

// ConversionProfile represents a conversion preset
type ConversionProfile struct {
	ID          string    `db:"id" json:"id"`
	Name        string    `db:"name" json:"name"`
	Description string    `db:"description" json:"description"`
	Codec       string    `db:"codec" json:"codec"`
	SampleRate  int       `db:"sample_rate" json:"sampleRate"`
	BitDepth    int       `db:"bit_depth" json:"bitDepth"`
	Bitrate     int       `db:"bitrate" json:"bitrate,omitempty"`
	Options     string    `db:"options" json:"-"`
	IsBuiltin   bool      `db:"is_builtin" json:"isBuiltin"`
	CreatedAt   time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt   time.Time `db:"updated_at" json:"updatedAt"`
}

// Helper methods

func (r *AnalysisResult) ParseIssues() error {
	if r.IssuesJSON != "" {
		return json.Unmarshal([]byte(r.IssuesJSON), &r.Issues)
	}
	return nil
}

func (r *AnalysisResult) ParseStats() error {
	if r.StatsJSON != "" {
		return json.Unmarshal([]byte(r.StatsJSON), &r.Stats)
	}
	return nil
}

// Status constants
const (
	StatusPending   = "pending"
	StatusQueued    = "queued"
	StatusRunning   = "running"
	StatusSuccess   = "success"
	StatusFailed    = "failed"
	StatusCancelled = "cancelled"
	StatusRetry     = "retry"

	SeverityInfo    = "info"
	SeverityWarning = "warning"
	SeverityError   = "error"

	LosslessPass    = "pass"
	LosslessWarn    = "warn"
	LosslessFail    = "fail"
)
