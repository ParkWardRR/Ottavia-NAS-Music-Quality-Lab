package metadata

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/ottavia-music/ottavia/internal/database"
	"github.com/ottavia-music/ottavia/internal/models"
	"github.com/rs/zerolog/log"
)

// Writer handles safe metadata writing operations
type Writer struct {
	db         *database.DB
	ffmpegPath string
}

// New creates a new metadata writer
func New(db *database.DB, ffmpegPath string) *Writer {
	return &Writer{
		db:         db,
		ffmpegPath: ffmpegPath,
	}
}

// TagChanges represents the changes to be made to a track's metadata
type TagChanges struct {
	Title       *string `json:"title,omitempty"`
	Artist      *string `json:"artist,omitempty"`
	Album       *string `json:"album,omitempty"`
	AlbumArtist *string `json:"albumArtist,omitempty"`
	TrackNumber *int    `json:"trackNumber,omitempty"`
	DiscNumber  *int    `json:"discNumber,omitempty"`
	Year        *int    `json:"year,omitempty"`
	Genre       *string `json:"genre,omitempty"`
}

// TagDiff represents the before/after diff for a single tag
type TagDiff struct {
	Field  string      `json:"field"`
	Before interface{} `json:"before"`
	After  interface{} `json:"after"`
}

// WritePreview represents the result of a dry-run preview
type WritePreview struct {
	TrackID  string    `json:"trackId"`
	Path     string    `json:"path"`
	Diffs    []TagDiff `json:"diffs"`
	CanWrite bool      `json:"canWrite"`
	Error    string    `json:"error,omitempty"`
}

// WriteResult represents the result of a write operation
type WriteResult struct {
	TrackID     string    `json:"trackId"`
	Path        string    `json:"path"`
	Success     bool      `json:"success"`
	Diffs       []TagDiff `json:"diffs"`
	ActionLogID string    `json:"actionLogId,omitempty"`
	Error       string    `json:"error,omitempty"`
}

// PreviewChanges performs a dry run showing what would change without modifying files
func (w *Writer) PreviewChanges(ctx context.Context, trackID string, changes *TagChanges) (*WritePreview, error) {
	// Fetch the current track
	track, err := w.db.GetTrack(ctx, trackID)
	if err != nil {
		return &WritePreview{
			TrackID:  trackID,
			CanWrite: false,
			Error:    fmt.Sprintf("Track not found: %v", err),
		}, nil
	}

	preview := &WritePreview{
		TrackID:  trackID,
		Path:     track.Path,
		Diffs:    []TagDiff{},
		CanWrite: true,
	}

	// Check if file exists and is writable
	if _, err := os.Stat(track.Path); os.IsNotExist(err) {
		preview.CanWrite = false
		preview.Error = "File does not exist"
		return preview, nil
	}

	// Check if file is writable
	file, err := os.OpenFile(track.Path, os.O_WRONLY, 0)
	if err != nil {
		preview.CanWrite = false
		preview.Error = fmt.Sprintf("File is not writable: %v", err)
		return preview, nil
	}
	file.Close()

	// Calculate diffs
	preview.Diffs = w.calculateDiffs(track, changes)

	return preview, nil
}

// ApplyChanges applies metadata changes to a track with safety mechanisms
func (w *Writer) ApplyChanges(ctx context.Context, trackID string, changes *TagChanges, actor string) (*WriteResult, error) {
	// First, preview to get the diffs and check writability
	preview, err := w.PreviewChanges(ctx, trackID, changes)
	if err != nil {
		return nil, err
	}

	if !preview.CanWrite {
		return &WriteResult{
			TrackID: trackID,
			Path:    preview.Path,
			Success: false,
			Diffs:   preview.Diffs,
			Error:   preview.Error,
		}, nil
	}

	if len(preview.Diffs) == 0 {
		return &WriteResult{
			TrackID: trackID,
			Path:    preview.Path,
			Success: true,
			Diffs:   []TagDiff{},
		}, nil
	}

	// Fetch track for the operation
	track, err := w.db.GetTrack(ctx, trackID)
	if err != nil {
		return nil, err
	}

	// Create before state for action log
	beforeState := w.trackToMap(track)

	// Perform the atomic write operation
	if err := w.atomicWrite(ctx, track.Path, changes); err != nil {
		return &WriteResult{
			TrackID: trackID,
			Path:    track.Path,
			Success: false,
			Diffs:   preview.Diffs,
			Error:   fmt.Sprintf("Write failed: %v", err),
		}, nil
	}

	// Update the database record
	w.applyChangesToTrack(track, changes)
	if err := w.db.UpdateTrack(ctx, track); err != nil {
		log.Error().Err(err).Str("track_id", trackID).Msg("Failed to update track in database after successful file write")
		// Note: File was already modified, but DB update failed
		// This is logged but not returned as failure since file write succeeded
	}

	// Create after state for action log
	afterState := w.trackToMap(track)

	// Log the action
	beforeJSON, _ := json.Marshal(beforeState)
	afterJSON, _ := json.Marshal(afterState)

	actionLog := &models.ActionLog{
		Type:       "tag_edit",
		TargetType: "track",
		TargetID:   trackID,
		Actor:      actor,
		BeforeJSON: string(beforeJSON),
		AfterJSON:  string(afterJSON),
	}

	if err := w.db.CreateActionLog(ctx, actionLog); err != nil {
		log.Error().Err(err).Str("track_id", trackID).Msg("Failed to create action log")
	}

	return &WriteResult{
		TrackID:     trackID,
		Path:        track.Path,
		Success:     true,
		Diffs:       preview.Diffs,
		ActionLogID: actionLog.ID,
	}, nil
}

// atomicWrite performs the actual metadata write using ffmpeg with atomic file operations
func (w *Writer) atomicWrite(ctx context.Context, filePath string, changes *TagChanges) error {
	// Create a temporary file in the same directory (for same-filesystem rename)
	dir := filepath.Dir(filePath)
	ext := filepath.Ext(filePath)
	tempFile := filepath.Join(dir, fmt.Sprintf(".ottavia_tmp_%d%s", time.Now().UnixNano(), ext))

	// Build ffmpeg command to copy file with new metadata
	args := []string{
		"-i", filePath,
		"-c", "copy", // Copy streams without re-encoding
	}

	// Add metadata arguments
	if changes.Title != nil {
		args = append(args, "-metadata", fmt.Sprintf("title=%s", *changes.Title))
	}
	if changes.Artist != nil {
		args = append(args, "-metadata", fmt.Sprintf("artist=%s", *changes.Artist))
	}
	if changes.Album != nil {
		args = append(args, "-metadata", fmt.Sprintf("album=%s", *changes.Album))
	}
	if changes.AlbumArtist != nil {
		args = append(args, "-metadata", fmt.Sprintf("album_artist=%s", *changes.AlbumArtist))
	}
	if changes.TrackNumber != nil {
		args = append(args, "-metadata", fmt.Sprintf("track=%d", *changes.TrackNumber))
	}
	if changes.DiscNumber != nil {
		args = append(args, "-metadata", fmt.Sprintf("disc=%d", *changes.DiscNumber))
	}
	if changes.Year != nil {
		args = append(args, "-metadata", fmt.Sprintf("date=%d", *changes.Year))
	}
	if changes.Genre != nil {
		args = append(args, "-metadata", fmt.Sprintf("genre=%s", *changes.Genre))
	}

	// Output to temp file
	args = append(args, "-y", tempFile)

	log.Debug().Strs("args", args).Msg("Running ffmpeg for metadata write")

	// Execute ffmpeg
	cmd := exec.CommandContext(ctx, w.ffmpegPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Clean up temp file if it exists
		os.Remove(tempFile)
		return fmt.Errorf("ffmpeg failed: %v, output: %s", err, string(output))
	}

	// Verify temp file was created successfully
	if _, err := os.Stat(tempFile); os.IsNotExist(err) {
		return fmt.Errorf("temp file was not created")
	}

	// Create backup of original file
	backupFile := filePath + ".ottavia_backup"
	if err := os.Rename(filePath, backupFile); err != nil {
		os.Remove(tempFile)
		return fmt.Errorf("failed to create backup: %v", err)
	}

	// Atomic rename temp to original
	if err := os.Rename(tempFile, filePath); err != nil {
		// Attempt to restore backup
		os.Rename(backupFile, filePath)
		return fmt.Errorf("failed to rename temp file: %v", err)
	}

	// Remove backup on success
	os.Remove(backupFile)

	log.Info().Str("path", filePath).Msg("Successfully wrote metadata")
	return nil
}

// calculateDiffs computes the differences between current track state and proposed changes
func (w *Writer) calculateDiffs(track *models.Track, changes *TagChanges) []TagDiff {
	diffs := []TagDiff{}

	if changes.Title != nil {
		current := ""
		if track.Title.Valid {
			current = track.Title.String
		}
		if current != *changes.Title {
			diffs = append(diffs, TagDiff{Field: "title", Before: current, After: *changes.Title})
		}
	}

	if changes.Artist != nil {
		current := ""
		if track.Artist.Valid {
			current = track.Artist.String
		}
		if current != *changes.Artist {
			diffs = append(diffs, TagDiff{Field: "artist", Before: current, After: *changes.Artist})
		}
	}

	if changes.Album != nil {
		current := ""
		if track.Album.Valid {
			current = track.Album.String
		}
		if current != *changes.Album {
			diffs = append(diffs, TagDiff{Field: "album", Before: current, After: *changes.Album})
		}
	}

	if changes.AlbumArtist != nil {
		current := ""
		if track.AlbumArtist.Valid {
			current = track.AlbumArtist.String
		}
		if current != *changes.AlbumArtist {
			diffs = append(diffs, TagDiff{Field: "albumArtist", Before: current, After: *changes.AlbumArtist})
		}
	}

	if changes.TrackNumber != nil {
		var current int32 = 0
		if track.TrackNumber.Valid {
			current = track.TrackNumber.Int32
		}
		if int(current) != *changes.TrackNumber {
			diffs = append(diffs, TagDiff{Field: "trackNumber", Before: current, After: *changes.TrackNumber})
		}
	}

	if changes.DiscNumber != nil {
		var current int32 = 0
		if track.DiscNumber.Valid {
			current = track.DiscNumber.Int32
		}
		if int(current) != *changes.DiscNumber {
			diffs = append(diffs, TagDiff{Field: "discNumber", Before: current, After: *changes.DiscNumber})
		}
	}

	if changes.Year != nil {
		var current int32 = 0
		if track.Year.Valid {
			current = track.Year.Int32
		}
		if int(current) != *changes.Year {
			diffs = append(diffs, TagDiff{Field: "year", Before: current, After: *changes.Year})
		}
	}

	if changes.Genre != nil {
		current := ""
		if track.Genre.Valid {
			current = track.Genre.String
		}
		if current != *changes.Genre {
			diffs = append(diffs, TagDiff{Field: "genre", Before: current, After: *changes.Genre})
		}
	}

	return diffs
}

// applyChangesToTrack updates the track model with the changes
func (w *Writer) applyChangesToTrack(track *models.Track, changes *TagChanges) {
	if changes.Title != nil {
		track.Title.String = *changes.Title
		track.Title.Valid = true
	}
	if changes.Artist != nil {
		track.Artist.String = *changes.Artist
		track.Artist.Valid = true
	}
	if changes.Album != nil {
		track.Album.String = *changes.Album
		track.Album.Valid = true
	}
	if changes.AlbumArtist != nil {
		track.AlbumArtist.String = *changes.AlbumArtist
		track.AlbumArtist.Valid = true
	}
	if changes.TrackNumber != nil {
		track.TrackNumber.Int32 = int32(*changes.TrackNumber)
		track.TrackNumber.Valid = true
	}
	if changes.DiscNumber != nil {
		track.DiscNumber.Int32 = int32(*changes.DiscNumber)
		track.DiscNumber.Valid = true
	}
	if changes.Year != nil {
		track.Year.Int32 = int32(*changes.Year)
		track.Year.Valid = true
	}
	if changes.Genre != nil {
		track.Genre.String = *changes.Genre
		track.Genre.Valid = true
	}
}

// trackToMap converts track metadata fields to a map for action logging
func (w *Writer) trackToMap(track *models.Track) map[string]interface{} {
	m := make(map[string]interface{})

	if track.Title.Valid {
		m["title"] = track.Title.String
	}
	if track.Artist.Valid {
		m["artist"] = track.Artist.String
	}
	if track.Album.Valid {
		m["album"] = track.Album.String
	}
	if track.AlbumArtist.Valid {
		m["albumArtist"] = track.AlbumArtist.String
	}
	if track.TrackNumber.Valid {
		m["trackNumber"] = track.TrackNumber.Int32
	}
	if track.DiscNumber.Valid {
		m["discNumber"] = track.DiscNumber.Int32
	}
	if track.Year.Valid {
		m["year"] = track.Year.Int32
	}
	if track.Genre.Valid {
		m["genre"] = track.Genre.String
	}

	return m
}
