package artwork

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ottavia-music/ottavia/internal/database"
	"github.com/ottavia-music/ottavia/internal/models"
	"github.com/rs/zerolog/log"
)

// Manager handles artwork extraction, upload, and application
type Manager struct {
	db           *database.DB
	ffmpegPath   string
	artifactPath string
}

// New creates a new artwork manager
func New(db *database.DB, ffmpegPath, artifactPath string) *Manager {
	return &Manager{
		db:           db,
		ffmpegPath:   ffmpegPath,
		artifactPath: artifactPath,
	}
}

// MissingArtworkSummary represents a group of tracks missing artwork
type MissingArtworkSummary struct {
	Album       string   `json:"album"`
	AlbumArtist string   `json:"albumArtist"`
	TrackCount  int      `json:"trackCount"`
	TrackIDs    []string `json:"trackIds"`
	Year        *int     `json:"year,omitempty"`
}

// ArtworkInfo represents artwork metadata
type ArtworkInfo struct {
	ID        string `json:"id"`
	TrackID   string `json:"trackId"`
	Path      string `json:"path"`
	MimeType  string `json:"mimeType"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	Size      int64  `json:"size"`
	Hash      string `json:"hash"`
	CreatedAt string `json:"createdAt"`
}

// ApplySuggestion represents a suggestion to apply artwork to tracks
type ApplySuggestion struct {
	Album         string   `json:"album"`
	AlbumArtist   string   `json:"albumArtist"`
	TrackCount    int      `json:"trackCount"`
	TrackIDs      []string `json:"trackIds"`
	MatchType     string   `json:"matchType"` // exact, fuzzy, artist
	Confidence    float64  `json:"confidence"`
	SourceTrackID string   `json:"sourceTrackId"`
	ArtworkID     string   `json:"artworkId"`
}

// ExtractResult represents the result of extracting artwork from a track
type ExtractResult struct {
	TrackID   string       `json:"trackId"`
	Success   bool         `json:"success"`
	Artwork   *ArtworkInfo `json:"artwork,omitempty"`
	Error     string       `json:"error,omitempty"`
	Skipped   bool         `json:"skipped,omitempty"`
	SkipReason string      `json:"skipReason,omitempty"`
}

// ListMissingArtwork returns tracks grouped by album that are missing artwork
func (m *Manager) ListMissingArtwork(ctx context.Context, libraryID string) ([]MissingArtworkSummary, error) {
	query := `
		SELECT
			t.album,
			t.album_artist,
			t.year,
			COUNT(*) as track_count,
			GROUP_CONCAT(t.id) as track_ids
		FROM tracks t
		JOIN media_files mf ON t.media_file_id = mf.id
		WHERE t.has_artwork = 0
		AND t.album IS NOT NULL
	`

	args := []interface{}{}
	if libraryID != "" {
		query += " AND mf.library_id = ?"
		args = append(args, libraryID)
	}

	query += " GROUP BY t.album, t.album_artist ORDER BY t.album"

	rows, err := m.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query missing artwork: %w", err)
	}
	defer rows.Close()

	var results []MissingArtworkSummary
	for rows.Next() {
		var summary MissingArtworkSummary
		var trackIDsStr string
		var yearNull *int32

		err := rows.Scan(&summary.Album, &summary.AlbumArtist, &yearNull, &summary.TrackCount, &trackIDsStr)
		if err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}

		if yearNull != nil {
			year := int(*yearNull)
			summary.Year = &year
		}
		summary.TrackIDs = strings.Split(trackIDsStr, ",")
		results = append(results, summary)
	}

	return results, nil
}

// ExtractArtwork extracts embedded artwork from an audio file
func (m *Manager) ExtractArtwork(ctx context.Context, trackID string) (*ExtractResult, error) {
	track, err := m.db.GetTrack(ctx, trackID)
	if err != nil {
		return &ExtractResult{
			TrackID: trackID,
			Success: false,
			Error:   fmt.Sprintf("track not found: %v", err),
		}, nil
	}

	// Check if already has artwork artifact
	existingArtwork, err := m.getTrackArtwork(ctx, trackID)
	if err == nil && existingArtwork != nil {
		return &ExtractResult{
			TrackID:    trackID,
			Success:    true,
			Skipped:    true,
			SkipReason: "artwork already extracted",
			Artwork:    existingArtwork,
		}, nil
	}

	// Check if track has embedded artwork
	if !track.HasArtwork {
		return &ExtractResult{
			TrackID:    trackID,
			Success:    false,
			Skipped:    true,
			SkipReason: "no embedded artwork in file",
		}, nil
	}

	// Extract artwork using ffmpeg
	artworkID := uuid.New().String()
	ext := "jpg" // Default to jpg
	tempFile := filepath.Join(os.TempDir(), fmt.Sprintf("%s.%s", artworkID, ext))

	// Use ffmpeg to extract cover art
	cmd := exec.CommandContext(ctx, m.ffmpegPath,
		"-i", track.Path,
		"-an", // no audio
		"-vcodec", "copy",
		tempFile,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return &ExtractResult{
			TrackID: trackID,
			Success: false,
			Error:   fmt.Sprintf("ffmpeg extraction failed: %v - %s", err, string(output)),
		}, nil
	}

	defer os.Remove(tempFile)

	// Get image dimensions and format
	imgFile, err := os.Open(tempFile)
	if err != nil {
		return &ExtractResult{
			TrackID: trackID,
			Success: false,
			Error:   fmt.Sprintf("failed to open extracted image: %v", err),
		}, nil
	}
	defer imgFile.Close()

	imgConfig, format, err := image.DecodeConfig(imgFile)
	if err != nil {
		return &ExtractResult{
			TrackID: trackID,
			Success: false,
			Error:   fmt.Sprintf("failed to decode image: %v", err),
		}, nil
	}

	// Determine actual extension from format
	if format == "png" {
		ext = "png"
	} else if format == "jpeg" {
		ext = "jpg"
	}

	// Calculate file hash
	imgFile.Seek(0, 0)
	hasher := sha256.New()
	fileData, _ := os.ReadFile(tempFile)
	hasher.Write(fileData)
	hash := hex.EncodeToString(hasher.Sum(nil))

	// Get file size
	fileInfo, _ := os.Stat(tempFile)
	fileSize := fileInfo.Size()

	// Move to artifacts directory
	artworkFileName := fmt.Sprintf("artwork_%s.%s", artworkID, ext)
	destPath := filepath.Join(m.artifactPath, artworkFileName)

	if err := os.Rename(tempFile, destPath); err != nil {
		// If rename fails (cross-device), try copy
		if err := copyFile(tempFile, destPath); err != nil {
			return &ExtractResult{
				TrackID: trackID,
				Success: false,
				Error:   fmt.Sprintf("failed to move artwork: %v", err),
			}, nil
		}
	}

	// Store in database
	artifact := &models.Artifact{
		ID:       artworkID,
		TrackID:  trackID,
		Type:     "artwork",
		Path:     artworkFileName,
		MimeType: fmt.Sprintf("image/%s", format),
		Width:    sql.NullInt32{Int32: int32(imgConfig.Width), Valid: true},
		Height:   sql.NullInt32{Int32: int32(imgConfig.Height), Valid: true},
	}

	if err := m.db.CreateArtifact(ctx, artifact); err != nil {
		os.Remove(destPath)
		return &ExtractResult{
			TrackID: trackID,
			Success: false,
			Error:   fmt.Sprintf("failed to save artifact: %v", err),
		}, nil
	}

	artworkInfo := &ArtworkInfo{
		ID:        artworkID,
		TrackID:   trackID,
		Path:      artworkFileName,
		MimeType:  artifact.MimeType,
		Width:     imgConfig.Width,
		Height:    imgConfig.Height,
		Size:      fileSize,
		Hash:      hash,
		CreatedAt: time.Now().Format(time.RFC3339),
	}

	return &ExtractResult{
		TrackID: trackID,
		Success: true,
		Artwork: artworkInfo,
	}, nil
}

// UploadArtwork uploads artwork and associates it with a track
func (m *Manager) UploadArtwork(ctx context.Context, trackID string, imageData []byte, mimeType string) (*ArtworkInfo, error) {
	// Decode image to get dimensions
	img, format, err := image.Decode(strings.NewReader(string(imageData)))
	if err != nil {
		return nil, fmt.Errorf("invalid image data: %w", err)
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Generate ID and filename
	artworkID := uuid.New().String()
	ext := "jpg"
	if format == "png" {
		ext = "png"
	}

	artworkFileName := fmt.Sprintf("artwork_%s.%s", artworkID, ext)
	destPath := filepath.Join(m.artifactPath, artworkFileName)

	// Write file
	if err := os.WriteFile(destPath, imageData, 0644); err != nil {
		return nil, fmt.Errorf("failed to write artwork file: %w", err)
	}

	// Calculate hash
	hasher := sha256.New()
	hasher.Write(imageData)
	hash := hex.EncodeToString(hasher.Sum(nil))

	// Store in database
	artifact := &models.Artifact{
		ID:       artworkID,
		TrackID:  trackID,
		Type:     "artwork",
		Path:     artworkFileName,
		MimeType: mimeType,
		Width:    sql.NullInt32{Int32: int32(width), Valid: true},
		Height:   sql.NullInt32{Int32: int32(height), Valid: true},
	}

	if err := m.db.CreateArtifact(ctx, artifact); err != nil {
		os.Remove(destPath)
		return nil, fmt.Errorf("failed to save artifact: %w", err)
	}

	// Update track has_artwork flag
	if err := m.db.UpdateTrackArtworkStatus(ctx, trackID, true, int32(width), int32(height)); err != nil {
		log.Warn().Err(err).Str("trackId", trackID).Msg("Failed to update track artwork status")
	}

	return &ArtworkInfo{
		ID:        artworkID,
		TrackID:   trackID,
		Path:      artworkFileName,
		MimeType:  mimeType,
		Width:     width,
		Height:    height,
		Size:      int64(len(imageData)),
		Hash:      hash,
		CreatedAt: time.Now().Format(time.RFC3339),
	}, nil
}

// ApplyArtworkToTracks applies artwork from one track to multiple other tracks
func (m *Manager) ApplyArtworkToTracks(ctx context.Context, sourceTrackID string, targetTrackIDs []string) ([]ExtractResult, error) {
	// Get source artwork
	sourceArtwork, err := m.getTrackArtwork(ctx, sourceTrackID)
	if err != nil {
		return nil, fmt.Errorf("source track has no artwork: %w", err)
	}

	// Read source artwork file
	sourcePath := filepath.Join(m.artifactPath, sourceArtwork.Path)
	imageData, err := os.ReadFile(sourcePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read source artwork: %w", err)
	}

	results := []ExtractResult{}

	for _, targetID := range targetTrackIDs {
		// Check if target already has artwork
		existingArtwork, _ := m.getTrackArtwork(ctx, targetID)
		if existingArtwork != nil {
			results = append(results, ExtractResult{
				TrackID:    targetID,
				Success:    true,
				Skipped:    true,
				SkipReason: "artwork already exists",
				Artwork:    existingArtwork,
			})
			continue
		}

		// Upload artwork for this track
		artworkInfo, err := m.UploadArtwork(ctx, targetID, imageData, sourceArtwork.MimeType)
		if err != nil {
			results = append(results, ExtractResult{
				TrackID: targetID,
				Success: false,
				Error:   err.Error(),
			})
			continue
		}

		results = append(results, ExtractResult{
			TrackID: targetID,
			Success: true,
			Artwork: artworkInfo,
		})
	}

	return results, nil
}

// GetSuggestions returns intelligent suggestions for applying artwork to similar tracks
func (m *Manager) GetSuggestions(ctx context.Context, trackID string) ([]ApplySuggestion, error) {
	track, err := m.db.GetTrack(ctx, trackID)
	if err != nil {
		return nil, fmt.Errorf("track not found: %w", err)
	}

	// Check if track has artwork
	artwork, err := m.getTrackArtwork(ctx, trackID)
	if err != nil {
		return nil, fmt.Errorf("track has no artwork: %w", err)
	}

	var suggestions []ApplySuggestion

	// Exact match: same album and album artist
	if track.Album.Valid && track.Album.String != "" {
		albumArtist := ""
		if track.AlbumArtist.Valid {
			albumArtist = track.AlbumArtist.String
		}
		exactMatches, err := m.findTracksWithoutArtwork(ctx, track.Album.String, albumArtist, true)
		if err == nil && len(exactMatches) > 0 {
			suggestions = append(suggestions, ApplySuggestion{
				Album:         track.Album.String,
				AlbumArtist:   albumArtist,
				TrackCount:    len(exactMatches),
				TrackIDs:      exactMatches,
				MatchType:     "exact",
				Confidence:    1.0,
				SourceTrackID: trackID,
				ArtworkID:     artwork.ID,
			})
		}
	}

	// Fuzzy match: same album, different/missing artist
	if track.Album.Valid && track.Album.String != "" {
		fuzzyMatches, err := m.findTracksWithoutArtwork(ctx, track.Album.String, "", false)
		if err == nil && len(fuzzyMatches) > 0 {
			// Filter out exact matches
			filtered := []string{}
			for _, id := range fuzzyMatches {
				found := false
				if len(suggestions) > 0 {
					for _, eid := range suggestions[0].TrackIDs {
						if eid == id {
							found = true
							break
						}
					}
				}
				if !found {
					filtered = append(filtered, id)
				}
			}

			if len(filtered) > 0 {
				suggestions = append(suggestions, ApplySuggestion{
					Album:         track.Album.String,
					AlbumArtist:   "",
					TrackCount:    len(filtered),
					TrackIDs:      filtered,
					MatchType:     "fuzzy",
					Confidence:    0.8,
					SourceTrackID: trackID,
					ArtworkID:     artwork.ID,
				})
			}
		}
	}

	// Artist match: same album artist, different album
	if track.AlbumArtist.Valid && track.AlbumArtist.String != "" {
		artistMatches, err := m.findTracksByArtistWithoutArtwork(ctx, track.AlbumArtist.String)
		if err == nil && len(artistMatches) > 0 {
			// Group by album
			albumGroups := make(map[string][]string)
			for _, id := range artistMatches {
				t, err := m.db.GetTrack(ctx, id)
				if err == nil && t.Album.Valid && t.Album.String != track.Album.String {
					albumGroups[t.Album.String] = append(albumGroups[t.Album.String], id)
				}
			}

			for album, trackIDs := range albumGroups {
				suggestions = append(suggestions, ApplySuggestion{
					Album:         album,
					AlbumArtist:   track.AlbumArtist.String,
					TrackCount:    len(trackIDs),
					TrackIDs:      trackIDs,
					MatchType:     "artist",
					Confidence:    0.5,
					SourceTrackID: trackID,
					ArtworkID:     artwork.ID,
				})
			}
		}
	}

	return suggestions, nil
}

// Helper functions

func (m *Manager) getTrackArtwork(ctx context.Context, trackID string) (*ArtworkInfo, error) {
	query := `
		SELECT id, track_id, path, mime_type, width, height, created_at
		FROM artifacts
		WHERE track_id = ? AND type = 'artwork'
		ORDER BY created_at DESC
		LIMIT 1
	`

	var info ArtworkInfo
	var createdAt time.Time
	var width, height sql.NullInt32

	err := m.db.QueryRowContext(ctx, query, trackID).Scan(
		&info.ID,
		&info.TrackID,
		&info.Path,
		&info.MimeType,
		&width,
		&height,
		&createdAt,
	)

	if err != nil {
		return nil, err
	}

	if width.Valid {
		info.Width = int(width.Int32)
	}
	if height.Valid {
		info.Height = int(height.Int32)
	}

	info.CreatedAt = createdAt.Format(time.RFC3339)

	// Get file size
	filePath := filepath.Join(m.artifactPath, info.Path)
	if fileInfo, err := os.Stat(filePath); err == nil {
		info.Size = fileInfo.Size()
	}

	return &info, nil
}

func (m *Manager) findTracksWithoutArtwork(ctx context.Context, album, albumArtist string, exactMatch bool) ([]string, error) {
	query := `
		SELECT id FROM tracks
		WHERE has_artwork = 0
		AND album = ?
	`

	args := []interface{}{album}

	if exactMatch && albumArtist != "" {
		query += " AND album_artist = ?"
		args = append(args, albumArtist)
	}

	rows, err := m.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trackIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		trackIDs = append(trackIDs, id)
	}

	return trackIDs, nil
}

func (m *Manager) findTracksByArtistWithoutArtwork(ctx context.Context, albumArtist string) ([]string, error) {
	query := `
		SELECT id FROM tracks
		WHERE has_artwork = 0
		AND album_artist = ?
	`

	rows, err := m.db.QueryContext(ctx, query, albumArtist)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trackIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		trackIDs = append(trackIDs, id)
	}

	return trackIDs, nil
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}
