package database

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"

	"github.com/ottavia-music/ottavia/internal/models"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

type DB struct {
	*sqlx.DB
}

func New(dsn string) (*DB, error) {
	db, err := sqlx.Connect("sqlite3", dsn+"?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(time.Hour)

	return &DB{db}, nil
}

func (db *DB) Migrate() error {
	migrations, err := migrationsFS.ReadFile("migrations/001_initial.sql")
	if err != nil {
		return fmt.Errorf("failed to read migrations: %w", err)
	}

	_, err = db.Exec(string(migrations))
	if err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return db.seedDefaults()
}

func (db *DB) seedDefaults() error {
	// Seed default conversion profiles
	profiles := []models.ConversionProfile{
		{
			ID:          "ipod-max",
			Name:        "iPod Max Compatibility",
			Description: "16-bit/44.1kHz ALAC for maximum compatibility with iPod and iTunes",
			Codec:       "alac",
			SampleRate:  44100,
			BitDepth:    16,
			IsBuiltin:   true,
		},
		{
			ID:          "redbook",
			Name:        "Red Book CD Quality",
			Description: "16-bit/44.1kHz FLAC - Standard CD quality",
			Codec:       "flac",
			SampleRate:  44100,
			BitDepth:    16,
			IsBuiltin:   true,
		},
		{
			ID:          "aac-256",
			Name:        "AAC 256kbps",
			Description: "High quality AAC for portable devices",
			Codec:       "aac",
			SampleRate:  44100,
			BitDepth:    0,
			Bitrate:     256000,
			IsBuiltin:   true,
		},
	}

	for _, p := range profiles {
		_, err := db.Exec(`
			INSERT OR IGNORE INTO conversion_profiles (id, name, description, codec, sample_rate, bit_depth, bitrate, is_builtin, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, p.ID, p.Name, p.Description, p.Codec, p.SampleRate, p.BitDepth, p.Bitrate, p.IsBuiltin, time.Now(), time.Now())
		if err != nil {
			return err
		}
	}

	// Seed default settings
	settings := []models.Setting{
		{Key: "theme", Value: "system", Type: "string", Category: "appearance"},
		{Key: "accent_color", Value: "blue", Type: "string", Category: "appearance"},
		{Key: "sidebar_collapsed", Value: "false", Type: "bool", Category: "appearance"},
		{Key: "scan_interval", Value: "15m", Type: "string", Category: "scanner"},
		{Key: "worker_count", Value: "4", Type: "int", Category: "scanner"},
		{Key: "auto_scan_enabled", Value: "true", Type: "bool", Category: "scanner"},
		{Key: "notifications_enabled", Value: "true", Type: "bool", Category: "notifications"},
		{Key: "notify_scan_complete", Value: "true", Type: "bool", Category: "notifications"},
		{Key: "notify_issues_found", Value: "true", Type: "bool", Category: "notifications"},
	}

	for _, s := range settings {
		_, err := db.Exec(`
			INSERT OR IGNORE INTO settings (key, value, type, category, updated_at)
			VALUES (?, ?, ?, ?, ?)
		`, s.Key, s.Value, s.Type, s.Category, time.Now())
		if err != nil {
			return err
		}
	}

	return nil
}

// Library operations

func (db *DB) CreateLibrary(ctx context.Context, lib *models.Library) error {
	lib.ID = uuid.NewString()
	lib.CreatedAt = time.Now()
	lib.UpdatedAt = time.Now()
	lib.Status = models.StatusPending

	_, err := db.ExecContext(ctx, `
		INSERT INTO libraries (id, name, root_path, scan_interval, read_only, output_path, allowed_formats, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, lib.ID, lib.Name, lib.RootPath, lib.ScanInterval, lib.ReadOnly, lib.OutputPath, lib.AllowedFormats, lib.Status, lib.CreatedAt, lib.UpdatedAt)

	return err
}

func (db *DB) GetLibrary(ctx context.Context, id string) (*models.Library, error) {
	var lib models.Library
	err := db.GetContext(ctx, &lib, "SELECT * FROM libraries WHERE id = ?", id)
	if err != nil {
		return nil, err
	}
	return &lib, nil
}

func (db *DB) ListLibraries(ctx context.Context) ([]models.Library, error) {
	var libs []models.Library
	err := db.SelectContext(ctx, &libs, `
		SELECT l.*,
			   COALESCE((SELECT COUNT(*) FROM tracks t JOIN media_files m ON t.media_file_id = m.id WHERE m.library_id = l.id), 0) as track_count
		FROM libraries l
		ORDER BY l.name
	`)
	return libs, err
}

func (db *DB) UpdateLibrary(ctx context.Context, lib *models.Library) error {
	lib.UpdatedAt = time.Now()
	_, err := db.ExecContext(ctx, `
		UPDATE libraries SET name = ?, root_path = ?, scan_interval = ?, read_only = ?,
		output_path = ?, allowed_formats = ?, status = ?, last_scan_at = ?, updated_at = ?
		WHERE id = ?
	`, lib.Name, lib.RootPath, lib.ScanInterval, lib.ReadOnly, lib.OutputPath, lib.AllowedFormats, lib.Status, lib.LastScanAt, lib.UpdatedAt, lib.ID)
	return err
}

func (db *DB) DeleteLibrary(ctx context.Context, id string) error {
	_, err := db.ExecContext(ctx, "DELETE FROM libraries WHERE id = ?", id)
	return err
}

// MediaFile operations

func (db *DB) CreateMediaFile(ctx context.Context, mf *models.MediaFile) error {
	mf.ID = uuid.NewString()
	mf.CreatedAt = time.Now()
	mf.UpdatedAt = time.Now()
	mf.Status = models.StatusPending

	_, err := db.ExecContext(ctx, `
		INSERT INTO media_files (id, library_id, path, filename, extension, size, mtime, quick_hash, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, mf.ID, mf.LibraryID, mf.Path, mf.Filename, mf.Extension, mf.Size, mf.Mtime, mf.QuickHash, mf.Status, mf.CreatedAt, mf.UpdatedAt)

	return err
}

func (db *DB) GetMediaFileByPath(ctx context.Context, libraryID, path string) (*models.MediaFile, error) {
	var mf models.MediaFile
	err := db.GetContext(ctx, &mf, "SELECT * FROM media_files WHERE library_id = ? AND path = ?", libraryID, path)
	if err != nil {
		return nil, err
	}
	return &mf, nil
}

func (db *DB) UpdateMediaFile(ctx context.Context, mf *models.MediaFile) error {
	mf.UpdatedAt = time.Now()
	_, err := db.ExecContext(ctx, `
		UPDATE media_files SET size = ?, mtime = ?, quick_hash = ?, full_hash = ?, status = ?, error_msg = ?, updated_at = ?
		WHERE id = ?
	`, mf.Size, mf.Mtime, mf.QuickHash, mf.FullHash, mf.Status, mf.ErrorMsg, mf.UpdatedAt, mf.ID)
	return err
}

func (db *DB) ListMediaFiles(ctx context.Context, libraryID string) ([]models.MediaFile, error) {
	var files []models.MediaFile
	err := db.SelectContext(ctx, &files, "SELECT * FROM media_files WHERE library_id = ? ORDER BY path", libraryID)
	return files, err
}

// Track operations

func (db *DB) CreateTrack(ctx context.Context, track *models.Track) error {
	track.ID = uuid.NewString()
	track.CreatedAt = time.Now()
	track.UpdatedAt = time.Now()

	_, err := db.ExecContext(ctx, `
		INSERT INTO tracks (id, media_file_id, duration, codec, sample_rate, bit_depth, channels, bitrate,
		title, artist, album, album_artist, track_number, disc_number, year, genre, has_artwork, artwork_width, artwork_height, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, track.ID, track.MediaFileID, track.Duration, track.Codec, track.SampleRate, track.BitDepth, track.Channels, track.Bitrate,
		track.Title, track.Artist, track.Album, track.AlbumArtist, track.TrackNumber, track.DiscNumber, track.Year, track.Genre,
		track.HasArtwork, track.ArtworkWidth, track.ArtworkHeight, track.CreatedAt, track.UpdatedAt)

	return err
}

func (db *DB) GetTrack(ctx context.Context, id string) (*models.Track, error) {
	var track models.Track
	err := db.GetContext(ctx, &track, `
		SELECT t.*, m.path, m.library_id, l.name as library_name
		FROM tracks t
		JOIN media_files m ON t.media_file_id = m.id
		JOIN libraries l ON m.library_id = l.id
		WHERE t.id = ?
	`, id)
	if err != nil {
		return nil, err
	}
	return &track, nil
}

func (db *DB) ListTracks(ctx context.Context, libraryID string, filter string, limit, offset int) ([]models.Track, int, error) {
	baseQuery := `
		FROM tracks t
		JOIN media_files m ON t.media_file_id = m.id
		JOIN libraries l ON m.library_id = l.id
	`
	where := "WHERE 1=1"
	args := []interface{}{}

	if libraryID != "" {
		where += " AND m.library_id = ?"
		args = append(args, libraryID)
	}

	if filter == "issues" {
		where += " AND EXISTS (SELECT 1 FROM analysis_results ar WHERE ar.track_id = t.id AND ar.lossless_status != 'pass')"
	}

	// Get total count
	var total int
	countQuery := "SELECT COUNT(*) " + baseQuery + where
	err := db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	// Get tracks
	query := `SELECT t.*, m.path, m.library_id, l.name as library_name ` + baseQuery + where + ` ORDER BY t.album, t.disc_number, t.track_number LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	var tracks []models.Track
	err = db.SelectContext(ctx, &tracks, query, args...)

	return tracks, total, err
}

func (db *DB) GetTrackByMediaFile(ctx context.Context, mediaFileID string) (*models.Track, error) {
	var track models.Track
	err := db.GetContext(ctx, &track, "SELECT * FROM tracks WHERE media_file_id = ?", mediaFileID)
	if err != nil {
		return nil, err
	}
	return &track, nil
}

func (db *DB) UpdateTrack(ctx context.Context, track *models.Track) error {
	track.UpdatedAt = time.Now()
	_, err := db.ExecContext(ctx, `
		UPDATE tracks SET duration = ?, codec = ?, sample_rate = ?, bit_depth = ?, channels = ?, bitrate = ?,
		title = ?, artist = ?, album = ?, album_artist = ?, track_number = ?, disc_number = ?, year = ?, genre = ?,
		has_artwork = ?, artwork_width = ?, artwork_height = ?, updated_at = ?
		WHERE id = ?
	`, track.Duration, track.Codec, track.SampleRate, track.BitDepth, track.Channels, track.Bitrate,
		track.Title, track.Artist, track.Album, track.AlbumArtist, track.TrackNumber, track.DiscNumber, track.Year, track.Genre,
		track.HasArtwork, track.ArtworkWidth, track.ArtworkHeight, track.UpdatedAt, track.ID)
	return err
}

// AnalysisResult operations

func (db *DB) CreateAnalysisResult(ctx context.Context, result *models.AnalysisResult) error {
	result.ID = uuid.NewString()
	result.CreatedAt = time.Now()

	_, err := db.ExecContext(ctx, `
		INSERT INTO analysis_results (id, track_id, version, lossless_score, lossless_status, integrity_ok, decode_errors,
		peak_level, true_peak, crest_factor, clipped_samples, dc_offset, integrated_loudness, loudness_range,
		high_freq_cutoff, spectral_rolloff, phase_correlation, issues_json, stats_json, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, result.ID, result.TrackID, result.Version, result.LosslessScore, result.LosslessStatus, result.IntegrityOK, result.DecodeErrors,
		result.PeakLevel, result.TruePeak, result.CrestFactor, result.ClippedSamples, result.DCOffset,
		result.IntegratedLoudness, result.LoudnessRange, result.HighFreqCutoff, result.SpectralRolloff,
		result.PhaseCorrelation, result.IssuesJSON, result.StatsJSON, result.CreatedAt)

	return err
}

func (db *DB) GetAnalysisResult(ctx context.Context, trackID string) (*models.AnalysisResult, error) {
	var result models.AnalysisResult
	err := db.GetContext(ctx, &result, `
		SELECT * FROM analysis_results WHERE track_id = ? ORDER BY version DESC LIMIT 1
	`, trackID)
	if err != nil {
		return nil, err
	}
	result.ParseIssues()
	result.ParseStats()
	return &result, nil
}

// Artifact operations

func (db *DB) CreateArtifact(ctx context.Context, artifact *models.Artifact) error {
	artifact.ID = uuid.NewString()
	artifact.CreatedAt = time.Now()

	_, err := db.ExecContext(ctx, `
		INSERT INTO artifacts (id, track_id, type, path, mime_type, width, height, metadata_json, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, artifact.ID, artifact.TrackID, artifact.Type, artifact.Path, artifact.MimeType, artifact.Width, artifact.Height, artifact.MetadataJSON, artifact.CreatedAt)

	return err
}

func (db *DB) ListArtifacts(ctx context.Context, trackID string) ([]models.Artifact, error) {
	var artifacts []models.Artifact
	err := db.SelectContext(ctx, &artifacts, "SELECT * FROM artifacts WHERE track_id = ? ORDER BY type", trackID)
	return artifacts, err
}

// Job operations

func (db *DB) CreateJob(ctx context.Context, job *models.Job) error {
	job.ID = uuid.NewString()
	job.CreatedAt = time.Now()
	job.Status = models.StatusQueued

	_, err := db.ExecContext(ctx, `
		INSERT INTO jobs (id, type, target_type, target_id, status, priority, attempts, max_attempts, payload_json, scheduled_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, job.ID, job.Type, job.TargetType, job.TargetID, job.Status, job.Priority, job.Attempts, job.MaxAttempts, job.PayloadJSON, job.ScheduledAt, job.CreatedAt)

	return err
}

func (db *DB) GetNextJob(ctx context.Context, jobType string) (*models.Job, error) {
	// Use a transaction to atomically claim a job
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Find and claim the next job in one atomic operation
	var job models.Job
	err = tx.GetContext(ctx, &job, `
		SELECT * FROM jobs
		WHERE type = ? AND status = ? AND scheduled_at <= ?
		ORDER BY priority DESC, scheduled_at ASC
		LIMIT 1
	`, jobType, models.StatusQueued, time.Now())
	if err != nil {
		return nil, err
	}

	// Immediately mark it as running
	_, err = tx.ExecContext(ctx, `
		UPDATE jobs SET status = ?, started_at = ?
		WHERE id = ? AND status = ?
	`, models.StatusRunning, time.Now(), job.ID, models.StatusQueued)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	job.Status = models.StatusRunning
	return &job, nil
}

func (db *DB) UpdateJob(ctx context.Context, job *models.Job) error {
	_, err := db.ExecContext(ctx, `
		UPDATE jobs SET status = ?, attempts = ?, last_error = ?, started_at = ?, finished_at = ?, scheduled_at = ?
		WHERE id = ?
	`, job.Status, job.Attempts, job.LastError, job.StartedAt, job.FinishedAt, job.ScheduledAt, job.ID)
	return err
}

func (db *DB) ListJobs(ctx context.Context, status string, limit int) ([]models.Job, error) {
	var jobs []models.Job
	query := "SELECT * FROM jobs"
	args := []interface{}{}

	if status != "" {
		query += " WHERE status = ?"
		args = append(args, status)
	}

	query += " ORDER BY created_at DESC LIMIT ?"
	args = append(args, limit)

	err := db.SelectContext(ctx, &jobs, query, args...)
	return jobs, err
}

// Settings operations

func (db *DB) GetSetting(ctx context.Context, key string) (*models.Setting, error) {
	var setting models.Setting
	err := db.GetContext(ctx, &setting, "SELECT * FROM settings WHERE key = ?", key)
	if err != nil {
		return nil, err
	}
	return &setting, nil
}

func (db *DB) SetSetting(ctx context.Context, setting *models.Setting) error {
	setting.UpdatedAt = time.Now()
	_, err := db.ExecContext(ctx, `
		INSERT INTO settings (key, value, type, category, updated_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(key) DO UPDATE SET value = ?, updated_at = ?
	`, setting.Key, setting.Value, setting.Type, setting.Category, setting.UpdatedAt, setting.Value, setting.UpdatedAt)
	return err
}

func (db *DB) ListSettings(ctx context.Context, category string) ([]models.Setting, error) {
	var settings []models.Setting
	query := "SELECT * FROM settings"
	args := []interface{}{}

	if category != "" {
		query += " WHERE category = ?"
		args = append(args, category)
	}

	query += " ORDER BY category, key"
	err := db.SelectContext(ctx, &settings, query, args...)
	return settings, err
}

func (db *DB) GetAllSettings(ctx context.Context) (map[string]string, error) {
	settings, err := db.ListSettings(ctx, "")
	if err != nil {
		return nil, err
	}

	result := make(map[string]string)
	for _, s := range settings {
		result[s.Key] = s.Value
	}
	return result, nil
}

// ScanRun operations

func (db *DB) CreateScanRun(ctx context.Context, run *models.ScanRun) error {
	run.ID = uuid.NewString()
	run.StartedAt = time.Now()
	run.Status = models.StatusRunning

	_, err := db.ExecContext(ctx, `
		INSERT INTO scan_runs (id, library_id, status, files_found, files_new, files_changed, files_deleted, files_failed, started_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, run.ID, run.LibraryID, run.Status, run.FilesFound, run.FilesNew, run.FilesChanged, run.FilesDeleted, run.FilesFailed, run.StartedAt)

	return err
}

func (db *DB) UpdateScanRun(ctx context.Context, run *models.ScanRun) error {
	_, err := db.ExecContext(ctx, `
		UPDATE scan_runs SET status = ?, files_found = ?, files_new = ?, files_changed = ?, files_deleted = ?, files_failed = ?, finished_at = ?, error_msg = ?
		WHERE id = ?
	`, run.Status, run.FilesFound, run.FilesNew, run.FilesChanged, run.FilesDeleted, run.FilesFailed, run.FinishedAt, run.ErrorMsg, run.ID)
	return err
}

func (db *DB) ListScanRuns(ctx context.Context, libraryID string, limit int) ([]models.ScanRun, error) {
	var runs []models.ScanRun
	err := db.SelectContext(ctx, &runs, `
		SELECT * FROM scan_runs WHERE library_id = ? ORDER BY started_at DESC LIMIT ?
	`, libraryID, limit)
	return runs, err
}

// ActionLog operations

func (db *DB) CreateActionLog(ctx context.Context, log *models.ActionLog) error {
	log.ID = uuid.NewString()
	log.CreatedAt = time.Now()

	_, err := db.ExecContext(ctx, `
		INSERT INTO action_logs (id, type, target_type, target_id, actor, before_json, after_json, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, log.ID, log.Type, log.TargetType, log.TargetID, log.Actor, log.BeforeJSON, log.AfterJSON, log.CreatedAt)

	return err
}

func (db *DB) ListActionLogs(ctx context.Context, targetType, targetID string, limit int) ([]models.ActionLog, error) {
	var logs []models.ActionLog
	query := "SELECT * FROM action_logs WHERE 1=1"
	args := []interface{}{}

	if targetType != "" {
		query += " AND target_type = ?"
		args = append(args, targetType)
	}
	if targetID != "" {
		query += " AND target_id = ?"
		args = append(args, targetID)
	}

	query += " ORDER BY created_at DESC LIMIT ?"
	args = append(args, limit)

	err := db.SelectContext(ctx, &logs, query, args...)
	return logs, err
}

// Stats

type DashboardStats struct {
	TotalLibraries  int     `db:"total_libraries" json:"totalLibraries"`
	TotalTracks     int     `db:"total_tracks" json:"totalTracks"`
	TotalSize       int64   `db:"total_size" json:"totalSize"`
	TracksWithIssues int    `db:"tracks_with_issues" json:"tracksWithIssues"`
	ActiveJobs      int     `db:"active_jobs" json:"activeJobs"`
	RecentScans     int     `db:"recent_scans" json:"recentScans"`
}

func (db *DB) GetDashboardStats(ctx context.Context) (*DashboardStats, error) {
	stats := &DashboardStats{}

	err := db.GetContext(ctx, &stats.TotalLibraries, "SELECT COUNT(*) FROM libraries")
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	err = db.GetContext(ctx, &stats.TotalTracks, "SELECT COUNT(*) FROM tracks")
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	err = db.GetContext(ctx, &stats.TotalSize, "SELECT COALESCE(SUM(size), 0) FROM media_files")
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	err = db.GetContext(ctx, &stats.TracksWithIssues, `
		SELECT COUNT(DISTINCT t.id) FROM tracks t
		JOIN analysis_results ar ON ar.track_id = t.id
		WHERE ar.lossless_status != 'pass'
	`)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	err = db.GetContext(ctx, &stats.ActiveJobs, "SELECT COUNT(*) FROM jobs WHERE status IN ('queued', 'running')")
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	err = db.GetContext(ctx, &stats.RecentScans, `
		SELECT COUNT(*) FROM scan_runs WHERE started_at > datetime('now', '-24 hours')
	`)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	return stats, nil
}

// ConversionProfile operations

func (db *DB) ListConversionProfiles(ctx context.Context) ([]models.ConversionProfile, error) {
	var profiles []models.ConversionProfile
	err := db.SelectContext(ctx, &profiles, "SELECT * FROM conversion_profiles ORDER BY name")
	return profiles, err
}

func (db *DB) GetConversionProfile(ctx context.Context, id string) (*models.ConversionProfile, error) {
	var profile models.ConversionProfile
	err := db.GetContext(ctx, &profile, "SELECT * FROM conversion_profiles WHERE id = ?", id)
	if err != nil {
		return nil, err
	}
	return &profile, nil
}

// Album represents a grouped album with version information
type Album struct {
	Name         string `db:"album_name" json:"name"`
	Artist       string `db:"album_artist" json:"artist"`
	Year         int    `db:"year" json:"year"`
	TrackCount   int    `db:"track_count" json:"trackCount"`
	VersionCount int    `db:"version_count" json:"versionCount"`
	Codecs       string `db:"codecs" json:"codecs"`
	TotalSize    int64  `db:"total_size" json:"totalSize"`
	HasIssues    bool   `db:"has_issues" json:"hasIssues"`
	ArtworkPath  string `db:"artwork_path" json:"artworkPath,omitempty"`
	AvgDR        int    `db:"avg_dr" json:"avgDR,omitempty"`        // Dynamic Range score
	IsLossless   bool   `db:"is_lossless" json:"isLossless"`
	IsSuspect    bool   `db:"is_suspect" json:"isSuspect"`          // Possible lossy transcode
	MaxBitDepth  int    `db:"max_bit_depth" json:"maxBitDepth"`
	MaxSampleRate int   `db:"max_sample_rate" json:"maxSampleRate"`
}

// AlbumVersion represents a specific version of an album (different path/quality)
type AlbumVersion struct {
	Path        string  `db:"base_path" json:"path"`
	Codec       string  `db:"codec" json:"codec"`
	SampleRate  int     `db:"sample_rate" json:"sampleRate"`
	BitDepth    int     `db:"bit_depth" json:"bitDepth"`
	TrackCount  int     `db:"track_count" json:"trackCount"`
	TotalSize   int64   `db:"total_size" json:"totalSize"`
	Quality     string  `json:"quality"`
}

func (db *DB) ListAlbums(ctx context.Context, limit, offset int) ([]Album, int, error) {
	// Get total count
	var total int
	err := db.GetContext(ctx, &total, `
		SELECT COUNT(DISTINCT COALESCE(t.album, ''))
		FROM tracks t
		WHERE t.album IS NOT NULL AND t.album != ''
	`)
	if err != nil {
		return nil, 0, err
	}

	// Get albums grouped with quality info
	var albums []Album
	err = db.SelectContext(ctx, &albums, `
		SELECT
			COALESCE(t.album, 'Unknown Album') as album_name,
			COALESCE(t.album_artist, t.artist, 'Unknown Artist') as album_artist,
			COALESCE(MAX(t.year), 0) as year,
			COUNT(DISTINCT t.id) as track_count,
			COUNT(DISTINCT SUBSTR(m.path, 1, LENGTH(m.path) - LENGTH(m.filename) - 1)) as version_count,
			GROUP_CONCAT(DISTINCT t.codec) as codecs,
			SUM(m.size) as total_size,
			CASE WHEN COUNT(ar.id) > 0 AND SUM(CASE WHEN ar.lossless_status != 'pass' THEN 1 ELSE 0 END) > 0 THEN 1 ELSE 0 END as has_issues,
			COALESCE((SELECT a.path FROM artifacts a WHERE a.track_id = (SELECT id FROM tracks WHERE album = t.album LIMIT 1) AND a.type = 'artwork' LIMIT 1), '') as artwork_path,
			CAST(COALESCE(AVG(ar.loudness_range + ar.crest_factor/2), 0) AS INTEGER) as avg_dr,
			CASE WHEN MAX(t.codec) IN ('flac', 'alac', 'wav', 'aiff') THEN 1 ELSE 0 END as is_lossless,
			CASE WHEN SUM(CASE WHEN ar.lossless_status = 'warn' OR ar.lossless_status = 'fail' THEN 1 ELSE 0 END) > 0 THEN 1 ELSE 0 END as is_suspect,
			MAX(t.bit_depth) as max_bit_depth,
			MAX(t.sample_rate) as max_sample_rate
		FROM tracks t
		JOIN media_files m ON t.media_file_id = m.id
		LEFT JOIN analysis_results ar ON ar.track_id = t.id
		WHERE t.album IS NOT NULL AND t.album != ''
		GROUP BY t.album, COALESCE(t.album_artist, t.artist)
		ORDER BY t.album
		LIMIT ? OFFSET ?
	`, limit, offset)

	return albums, total, err
}

// GetAlbumArtwork returns the artwork path for a specific album
func (db *DB) GetAlbumArtwork(ctx context.Context, albumName string) (string, error) {
	var path string
	err := db.GetContext(ctx, &path, `
		SELECT a.path
		FROM artifacts a
		JOIN tracks t ON a.track_id = t.id
		WHERE t.album = ? AND a.type = 'artwork'
		LIMIT 1
	`, albumName)
	if err != nil {
		return "", err
	}
	return path, nil
}

func (db *DB) GetAlbumVersions(ctx context.Context, albumName, artistName string) ([]AlbumVersion, error) {
	var versions []AlbumVersion
	err := db.SelectContext(ctx, &versions, `
		SELECT
			SUBSTR(m.path, 1, LENGTH(m.path) - LENGTH(m.filename) - 1) as base_path,
			t.codec,
			t.sample_rate,
			t.bit_depth,
			COUNT(t.id) as track_count,
			SUM(m.size) as total_size
		FROM tracks t
		JOIN media_files m ON t.media_file_id = m.id
		WHERE t.album = ? AND COALESCE(t.album_artist, t.artist) = ?
		GROUP BY SUBSTR(m.path, 1, LENGTH(m.path) - LENGTH(m.filename) - 1), t.codec, t.sample_rate, t.bit_depth
		ORDER BY t.bit_depth DESC, t.sample_rate DESC
	`, albumName, artistName)

	// Add quality label
	for i := range versions {
		versions[i].Quality = formatQuality(versions[i].Codec, versions[i].SampleRate, versions[i].BitDepth)
	}

	return versions, err
}

func formatQuality(codec string, sampleRate, bitDepth int) string {
	codecLabel := strings.ToUpper(codec)
	if sampleRate >= 88200 && bitDepth >= 24 {
		return fmt.Sprintf("%s Hi-Res (%d/%d)", codecLabel, bitDepth, sampleRate/1000)
	}
	if bitDepth >= 24 {
		return fmt.Sprintf("%s %d-bit", codecLabel, bitDepth)
	}
	if sampleRate == 44100 && bitDepth == 16 {
		return fmt.Sprintf("%s CD Quality", codecLabel)
	}
	return fmt.Sprintf("%s %dkHz/%d-bit", codecLabel, sampleRate/1000, bitDepth)
}

// AlbumDetail contains full album information with all tracks
type AlbumDetail struct {
	Name          string       `json:"name"`
	Artist        string       `json:"artist"`
	Year          int          `json:"year"`
	TrackCount    int          `json:"trackCount"`
	TotalDuration float64      `json:"totalDuration"`
	TotalSize     int64        `json:"totalSize"`
	ArtworkPath   string       `json:"artworkPath,omitempty"`
	Tracks        []AlbumTrack `json:"tracks"`
	Consistency   AlbumConsistency `json:"consistency"`
}

// AlbumTrack represents a track within an album with analysis data for consistency view
type AlbumTrack struct {
	ID               string  `db:"id" json:"id"`
	TrackNumber      int     `db:"track_number" json:"trackNumber"`
	DiscNumber       int     `db:"disc_number" json:"discNumber"`
	Title            string  `db:"title" json:"title"`
	Duration         float64 `db:"duration" json:"duration"`
	Codec            string  `db:"codec" json:"codec"`
	SampleRate       int     `db:"sample_rate" json:"sampleRate"`
	BitDepth         int     `db:"bit_depth" json:"bitDepth"`
	Bitrate          int     `db:"bitrate" json:"bitrate"`
	FileSize         int64   `db:"file_size" json:"fileSize"`
	Path             string  `db:"path" json:"path"`
	// Analysis data
	LosslessStatus   string  `db:"lossless_status" json:"losslessStatus"`
	LosslessScore    float64 `db:"lossless_score" json:"losslessScore"`
	IntegrityOK      bool    `db:"integrity_ok" json:"integrityOK"`
	ClippedSamples   int     `db:"clipped_samples" json:"clippedSamples"`
	PeakLevel        float64 `db:"peak_level" json:"peakLevel"`
	IntegratedLoudness float64 `db:"integrated_loudness" json:"integratedLoudness"`
	LoudnessRange    float64 `db:"loudness_range" json:"loudnessRange"`
	CrestFactor      float64 `db:"crest_factor" json:"crestFactor"`
	DRScore          int     `json:"drScore"`
	// Outlier flags
	IsCodecOutlier     bool `json:"isCodecOutlier"`
	IsSampleRateOutlier bool `json:"isSampleRateOutlier"`
	IsBitDepthOutlier  bool `json:"isBitDepthOutlier"`
	IsDROutlier        bool `json:"isDROutlier"`
	IsLoudnessOutlier  bool `json:"isLoudnessOutlier"`
	IsSuspect          bool `json:"isSuspect"`
}

// AlbumConsistency contains consistency analysis for an album
type AlbumConsistency struct {
	IsConsistent     bool    `json:"isConsistent"`
	DominantCodec    string  `json:"dominantCodec"`
	DominantSampleRate int   `json:"dominantSampleRate"`
	DominantBitDepth int     `json:"dominantBitDepth"`
	AvgDR            int     `json:"avgDR"`
	AvgLoudness      float64 `json:"avgLoudness"`
	CodecVariety     int     `json:"codecVariety"`
	SampleRateVariety int    `json:"sampleRateVariety"`
	BitDepthVariety  int     `json:"bitDepthVariety"`
	SuspectCount     int     `json:"suspectCount"`
	IssueCount       int     `json:"issueCount"`
}

// GetAlbumDetail retrieves full album details with all tracks and consistency analysis
func (db *DB) GetAlbumDetail(ctx context.Context, albumName, artistName string) (*AlbumDetail, error) {
	var tracks []AlbumTrack

	err := db.SelectContext(ctx, &tracks, `
		SELECT
			t.id,
			COALESCE(t.track_number, 0) as track_number,
			COALESCE(t.disc_number, 1) as disc_number,
			COALESCE(t.title, m.filename) as title,
			t.duration,
			t.codec,
			t.sample_rate,
			t.bit_depth,
			COALESCE(t.bitrate, 0) as bitrate,
			m.size as file_size,
			m.path,
			COALESCE(ar.lossless_status, 'pending') as lossless_status,
			COALESCE(ar.lossless_score, 0) as lossless_score,
			COALESCE(ar.integrity_ok, 1) as integrity_ok,
			COALESCE(ar.clipped_samples, 0) as clipped_samples,
			COALESCE(ar.peak_level, 0) as peak_level,
			COALESCE(ar.integrated_loudness, 0) as integrated_loudness,
			COALESCE(ar.loudness_range, 0) as loudness_range,
			COALESCE(ar.crest_factor, 0) as crest_factor
		FROM tracks t
		JOIN media_files m ON t.media_file_id = m.id
		LEFT JOIN analysis_results ar ON ar.track_id = t.id
		WHERE t.album = ? AND COALESCE(t.album_artist, t.artist, '') = ?
		ORDER BY t.disc_number, t.track_number, t.title
	`, albumName, artistName)
	if err != nil {
		return nil, err
	}

	if len(tracks) == 0 {
		return nil, sql.ErrNoRows
	}

	// Calculate DR scores and collect stats for consistency analysis
	codecCount := make(map[string]int)
	sampleRateCount := make(map[int]int)
	bitDepthCount := make(map[int]int)
	var totalDR, totalLoudness float64
	var drCount, loudnessCount int
	var totalDuration float64
	var totalSize int64
	var year int
	var suspectCount, issueCount int

	for i := range tracks {
		// Calculate DR score
		dr := int(tracks[i].LoudnessRange + tracks[i].CrestFactor/2)
		if dr < 1 {
			dr = 1
		}
		if dr > 20 {
			dr = 20
		}
		tracks[i].DRScore = dr

		// Collect for averages
		if tracks[i].LoudnessRange > 0 {
			totalDR += float64(dr)
			drCount++
		}
		if tracks[i].IntegratedLoudness != 0 {
			totalLoudness += tracks[i].IntegratedLoudness
			loudnessCount++
		}

		// Count for dominant detection
		codecCount[tracks[i].Codec]++
		sampleRateCount[tracks[i].SampleRate]++
		bitDepthCount[tracks[i].BitDepth]++

		totalDuration += tracks[i].Duration
		totalSize += tracks[i].FileSize

		// Track issues
		if tracks[i].LosslessStatus == "warn" || tracks[i].LosslessStatus == "fail" {
			tracks[i].IsSuspect = true
			suspectCount++
		}
		if !tracks[i].IntegrityOK || tracks[i].ClippedSamples > 100 {
			issueCount++
		}
	}

	// Find dominant values (most common)
	dominantCodec := findDominantString(codecCount)
	dominantSampleRate := findDominantInt(sampleRateCount)
	dominantBitDepth := findDominantInt(bitDepthCount)

	avgDR := 0
	if drCount > 0 {
		avgDR = int(totalDR / float64(drCount))
	}
	avgLoudness := 0.0
	if loudnessCount > 0 {
		avgLoudness = totalLoudness / float64(loudnessCount)
	}

	// Mark outliers
	for i := range tracks {
		if tracks[i].Codec != dominantCodec {
			tracks[i].IsCodecOutlier = true
		}
		if tracks[i].SampleRate != dominantSampleRate {
			tracks[i].IsSampleRateOutlier = true
		}
		if tracks[i].BitDepth != dominantBitDepth {
			tracks[i].IsBitDepthOutlier = true
		}
		// DR outlier if differs by more than 4 from average
		if avgDR > 0 && (tracks[i].DRScore < avgDR-4 || tracks[i].DRScore > avgDR+4) {
			tracks[i].IsDROutlier = true
		}
		// Loudness outlier if differs by more than 3 LUFS from average
		if avgLoudness != 0 && tracks[i].IntegratedLoudness != 0 {
			diff := tracks[i].IntegratedLoudness - avgLoudness
			if diff < -3 || diff > 3 {
				tracks[i].IsLoudnessOutlier = true
			}
		}
	}

	// Get artwork path
	artworkPath, _ := db.GetAlbumArtwork(ctx, albumName)

	// Get year from first track with a year
	yearQuery := `SELECT COALESCE(MAX(year), 0) FROM tracks WHERE album = ?`
	db.GetContext(ctx, &year, yearQuery, albumName)

	consistency := AlbumConsistency{
		IsConsistent:      len(codecCount) == 1 && len(sampleRateCount) == 1 && len(bitDepthCount) == 1 && suspectCount == 0,
		DominantCodec:     dominantCodec,
		DominantSampleRate: dominantSampleRate,
		DominantBitDepth:  dominantBitDepth,
		AvgDR:             avgDR,
		AvgLoudness:       avgLoudness,
		CodecVariety:      len(codecCount),
		SampleRateVariety: len(sampleRateCount),
		BitDepthVariety:   len(bitDepthCount),
		SuspectCount:      suspectCount,
		IssueCount:        issueCount,
	}

	return &AlbumDetail{
		Name:          albumName,
		Artist:        artistName,
		Year:          year,
		TrackCount:    len(tracks),
		TotalDuration: totalDuration,
		TotalSize:     totalSize,
		ArtworkPath:   artworkPath,
		Tracks:        tracks,
		Consistency:   consistency,
	}, nil
}

func findDominantString(counts map[string]int) string {
	maxCount := 0
	dominant := ""
	for val, count := range counts {
		if count > maxCount {
			maxCount = count
			dominant = val
		}
	}
	return dominant
}

func findDominantInt(counts map[int]int) int {
	maxCount := 0
	dominant := 0
	for val, count := range counts {
		if count > maxCount {
			maxCount = count
			dominant = val
		}
	}
	return dominant
}
