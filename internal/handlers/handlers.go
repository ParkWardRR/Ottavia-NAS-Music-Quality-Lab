package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"

	"github.com/ottavia-music/ottavia/internal/analyzer"
	"github.com/ottavia-music/ottavia/internal/artwork"
	"github.com/ottavia-music/ottavia/internal/database"
	"github.com/ottavia-music/ottavia/internal/metadata"
	"github.com/ottavia-music/ottavia/internal/models"
	"github.com/ottavia-music/ottavia/internal/scanner"
)

type Handler struct {
	db             *database.DB
	scanner        *scanner.Scanner
	analyzer       *analyzer.Analyzer
	metadataWriter *metadata.Writer
	artworkManager *artwork.Manager
}

func New(db *database.DB, scanner *scanner.Scanner, analyzer *analyzer.Analyzer, metadataWriter *metadata.Writer, artworkManager *artwork.Manager) *Handler {
	return &Handler{
		db:             db,
		scanner:        scanner,
		analyzer:       analyzer,
		metadataWriter: metadataWriter,
		artworkManager: artworkManager,
	}
}

func (h *Handler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

func (h *Handler) respondError(w http.ResponseWriter, status int, message string) {
	h.respondJSON(w, status, map[string]string{"error": message})
}

// Libraries

func (h *Handler) ListLibraries(w http.ResponseWriter, r *http.Request) {
	libs, err := h.db.ListLibraries(r.Context())
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.respondJSON(w, http.StatusOK, libs)
}

func (h *Handler) GetLibrary(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	lib, err := h.db.GetLibrary(r.Context(), id)
	if err != nil {
		h.respondError(w, http.StatusNotFound, "Library not found")
		return
	}
	h.respondJSON(w, http.StatusOK, lib)
}

type CreateLibraryRequest struct {
	Name         string `json:"name"`
	RootPath     string `json:"rootPath"`
	ScanInterval string `json:"scanInterval"`
	ReadOnly     bool   `json:"readOnly"`
	OutputPath   string `json:"outputPath,omitempty"`
}

func (h *Handler) CreateLibrary(w http.ResponseWriter, r *http.Request) {
	var req CreateLibraryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Name == "" || req.RootPath == "" {
		h.respondError(w, http.StatusBadRequest, "Name and root path are required")
		return
	}

	lib := &models.Library{
		Name:         req.Name,
		RootPath:     req.RootPath,
		ScanInterval: req.ScanInterval,
		ReadOnly:     req.ReadOnly,
	}

	if req.ScanInterval == "" {
		lib.ScanInterval = "15m"
	}

	if err := h.db.CreateLibrary(r.Context(), lib); err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondJSON(w, http.StatusCreated, lib)
}

func (h *Handler) UpdateLibrary(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	lib, err := h.db.GetLibrary(r.Context(), id)
	if err != nil {
		h.respondError(w, http.StatusNotFound, "Library not found")
		return
	}

	var req CreateLibraryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Name != "" {
		lib.Name = req.Name
	}
	if req.RootPath != "" {
		lib.RootPath = req.RootPath
	}
	if req.ScanInterval != "" {
		lib.ScanInterval = req.ScanInterval
	}
	lib.ReadOnly = req.ReadOnly

	if err := h.db.UpdateLibrary(r.Context(), lib); err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, lib)
}

func (h *Handler) DeleteLibrary(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := h.db.DeleteLibrary(r.Context(), id); err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ScanLibrary(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	go func() {
		ctx := context.Background()
		result, err := h.scanner.ScanLibrary(ctx, id)
		if err != nil {
			log.Error().Err(err).Str("library_id", id).Msg("Scan failed")
			return
		}
		log.Info().
			Str("library_id", id).
			Int("new", result.Run.FilesNew).
			Int("changed", result.Run.FilesChanged).
			Msg("Scan completed")
	}()

	h.respondJSON(w, http.StatusAccepted, map[string]string{
		"message": "Scan started",
		"status":  "running",
	})
}

// Tracks

func (h *Handler) ListTracks(w http.ResponseWriter, r *http.Request) {
	libraryID := r.URL.Query().Get("library_id")
	filter := r.URL.Query().Get("filter")
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 50
	offset := 0

	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}
	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	tracks, total, err := h.db.ListTracks(r.Context(), libraryID, filter, limit, offset)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"tracks": tracks,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

func (h *Handler) GetTrack(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	track, err := h.db.GetTrack(r.Context(), id)
	if err != nil {
		h.respondError(w, http.StatusNotFound, "Track not found")
		return
	}

	analysis, _ := h.db.GetAnalysisResult(r.Context(), id)
	artifacts, _ := h.db.ListArtifacts(r.Context(), id)

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"track":    track,
		"analysis": analysis,
		"artifacts": artifacts,
	})
}

func (h *Handler) GetTrackArtifacts(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	artifacts, err := h.db.ListArtifacts(r.Context(), id)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, artifacts)
}

type UpdateTagsRequest struct {
	Title       *string `json:"title,omitempty"`
	Artist      *string `json:"artist,omitempty"`
	Album       *string `json:"album,omitempty"`
	AlbumArtist *string `json:"albumArtist,omitempty"`
	TrackNumber *int    `json:"trackNumber,omitempty"`
	DiscNumber  *int    `json:"discNumber,omitempty"`
	Year        *int    `json:"year,omitempty"`
	Genre       *string `json:"genre,omitempty"`
}

// PreviewTrackTags performs a dry-run of tag changes showing what would be modified
func (h *Handler) PreviewTrackTags(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req UpdateTagsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	changes := &metadata.TagChanges{
		Title:       req.Title,
		Artist:      req.Artist,
		Album:       req.Album,
		AlbumArtist: req.AlbumArtist,
		TrackNumber: req.TrackNumber,
		DiscNumber:  req.DiscNumber,
		Year:        req.Year,
		Genre:       req.Genre,
	}

	preview, err := h.metadataWriter.PreviewChanges(r.Context(), id, changes)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, preview)
}

// UpdateTrackTags applies tag changes to a track with safe write operations
func (h *Handler) UpdateTrackTags(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req UpdateTagsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	changes := &metadata.TagChanges{
		Title:       req.Title,
		Artist:      req.Artist,
		Album:       req.Album,
		AlbumArtist: req.AlbumArtist,
		TrackNumber: req.TrackNumber,
		DiscNumber:  req.DiscNumber,
		Year:        req.Year,
		Genre:       req.Genre,
	}

	// Get actor from request (could be from auth header in future)
	actor := r.Header.Get("X-Actor")
	if actor == "" {
		actor = "system"
	}

	result, err := h.metadataWriter.ApplyChanges(r.Context(), id, changes, actor)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if !result.Success {
		h.respondJSON(w, http.StatusBadRequest, result)
		return
	}

	h.respondJSON(w, http.StatusOK, result)
}

// Bulk metadata operations

type BulkOperationRequest struct {
	TrackIDs  []string    `json:"trackIds"`
	Operation string      `json:"operation"` // "normalize_album_artist", "fix_track_numbers", "set_field"
	Value     interface{} `json:"value,omitempty"`
	Field     string      `json:"field,omitempty"`
}

// PreviewBulkOperation previews a bulk metadata operation
func (h *Handler) PreviewBulkOperation(w http.ResponseWriter, r *http.Request) {
	var req BulkOperationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if len(req.TrackIDs) == 0 {
		h.respondError(w, http.StatusBadRequest, "No track IDs provided")
		return
	}

	if req.Operation == "" {
		h.respondError(w, http.StatusBadRequest, "Operation is required")
		return
	}

	op := &metadata.BulkOperation{
		TrackIDs:  req.TrackIDs,
		Operation: req.Operation,
		Value:     req.Value,
		Field:     req.Field,
	}

	preview, err := h.metadataWriter.PreviewBulkOperation(r.Context(), op)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, preview)
}

// ApplyBulkOperation applies a bulk metadata operation
func (h *Handler) ApplyBulkOperation(w http.ResponseWriter, r *http.Request) {
	var req BulkOperationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if len(req.TrackIDs) == 0 {
		h.respondError(w, http.StatusBadRequest, "No track IDs provided")
		return
	}

	if req.Operation == "" {
		h.respondError(w, http.StatusBadRequest, "Operation is required")
		return
	}

	actor := r.Header.Get("X-Actor")
	if actor == "" {
		actor = "system"
	}

	op := &metadata.BulkOperation{
		TrackIDs:  req.TrackIDs,
		Operation: req.Operation,
		Value:     req.Value,
		Field:     req.Field,
	}

	result, err := h.metadataWriter.ApplyBulkOperation(r.Context(), op, actor)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, result)
}

type NormalizeAlbumArtistRequest struct {
	AlbumName   string `json:"albumName"`
	Artist      string `json:"artist"`
	AlbumArtist string `json:"albumArtist"`
}

// NormalizeAlbumArtist normalizes the album artist for all tracks in an album
func (h *Handler) NormalizeAlbumArtist(w http.ResponseWriter, r *http.Request) {
	var req NormalizeAlbumArtistRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.AlbumName == "" {
		h.respondError(w, http.StatusBadRequest, "Album name is required")
		return
	}

	actor := r.Header.Get("X-Actor")
	if actor == "" {
		actor = "system"
	}

	result, err := h.metadataWriter.NormalizeAlbumArtist(r.Context(), req.AlbumName, req.Artist, req.AlbumArtist, actor)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, result)
}

type FixTrackNumberingRequest struct {
	AlbumName string `json:"albumName"`
	Artist    string `json:"artist"`
}

// FixTrackNumbering renumbers tracks sequentially for an album
func (h *Handler) FixTrackNumbering(w http.ResponseWriter, r *http.Request) {
	var req FixTrackNumberingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.AlbumName == "" {
		h.respondError(w, http.StatusBadRequest, "Album name is required")
		return
	}

	actor := r.Header.Get("X-Actor")
	if actor == "" {
		actor = "system"
	}

	result, err := h.metadataWriter.FixTrackNumbering(r.Context(), req.AlbumName, req.Artist, actor)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, result)
}

// Jobs

func (h *Handler) ListJobs(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	limitStr := r.URL.Query().Get("limit")

	limit := 50
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	jobs, err := h.db.ListJobs(r.Context(), status, limit)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, jobs)
}

// Settings

func (h *Handler) GetSettings(w http.ResponseWriter, r *http.Request) {
	category := r.URL.Query().Get("category")

	settings, err := h.db.ListSettings(r.Context(), category)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	result := make(map[string]interface{})
	for _, s := range settings {
		switch s.Type {
		case "bool":
			result[s.Key] = s.Value == "true"
		case "int":
			if v, err := strconv.Atoi(s.Value); err == nil {
				result[s.Key] = v
			}
		default:
			result[s.Key] = s.Value
		}
	}

	h.respondJSON(w, http.StatusOK, result)
}

func (h *Handler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	for key, value := range req {
		existing, _ := h.db.GetSetting(r.Context(), key)

		setting := &models.Setting{
			Key:      key,
			Category: "general",
			Type:     "string",
		}

		if existing != nil {
			setting.Category = existing.Category
			setting.Type = existing.Type
		}

		switch v := value.(type) {
		case bool:
			setting.Value = strconv.FormatBool(v)
			setting.Type = "bool"
		case float64:
			setting.Value = strconv.FormatInt(int64(v), 10)
			setting.Type = "int"
		case string:
			setting.Value = v
		default:
			data, _ := json.Marshal(v)
			setting.Value = string(data)
			setting.Type = "json"
		}

		if err := h.db.SetSetting(r.Context(), setting); err != nil {
			h.respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	h.respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Dashboard stats

func (h *Handler) GetDashboardStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.db.GetDashboardStats(r.Context())
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.respondJSON(w, http.StatusOK, stats)
}

// Conversion profiles

func (h *Handler) ListConversionProfiles(w http.ResponseWriter, r *http.Request) {
	profiles, err := h.db.ListConversionProfiles(r.Context())
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.respondJSON(w, http.StatusOK, profiles)
}

// Scan runs

func (h *Handler) ListScanRuns(w http.ResponseWriter, r *http.Request) {
	libraryID := chi.URLParam(r, "id")
	limitStr := r.URL.Query().Get("limit")

	limit := 20
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	runs, err := h.db.ListScanRuns(r.Context(), libraryID, limit)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, runs)
}

// Action logs

func (h *Handler) ListActionLogs(w http.ResponseWriter, r *http.Request) {
	targetType := r.URL.Query().Get("target_type")
	targetID := r.URL.Query().Get("target_id")
	limitStr := r.URL.Query().Get("limit")

	limit := 50
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	logs, err := h.db.ListActionLogs(r.Context(), targetType, targetID, limit)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, logs)
}

// Artwork management

func (h *Handler) ListMissingArtwork(w http.ResponseWriter, r *http.Request) {
	libraryID := r.URL.Query().Get("library_id")

	missing, err := h.artworkManager.ListMissingArtwork(r.Context(), libraryID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, missing)
}

type ExtractArtworkRequest struct {
	TrackIDs []string `json:"trackIds"`
}

func (h *Handler) ExtractArtwork(w http.ResponseWriter, r *http.Request) {
	var req ExtractArtworkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if len(req.TrackIDs) == 0 {
		h.respondError(w, http.StatusBadRequest, "No track IDs provided")
		return
	}

	results := []interface{}{}
	for _, trackID := range req.TrackIDs {
		result, err := h.artworkManager.ExtractArtwork(r.Context(), trackID)
		if err != nil {
			log.Error().Err(err).Str("trackId", trackID).Msg("Failed to extract artwork")
			continue
		}
		results = append(results, result)
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"results": results,
	})
}

func (h *Handler) UploadArtwork(w http.ResponseWriter, r *http.Request) {
	trackID := chi.URLParam(r, "id")

	// Parse multipart form
	err := r.ParseMultipartForm(10 << 20) // 10 MB max
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Failed to parse form data")
		return
	}

	file, header, err := r.FormFile("artwork")
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "No artwork file provided")
		return
	}
	defer file.Close()

	// Read file data
	imageData, err := io.ReadAll(file)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to read artwork file")
		return
	}

	// Determine MIME type
	mimeType := header.Header.Get("Content-Type")
	if mimeType == "" {
		// Guess from file extension
		if strings.HasSuffix(strings.ToLower(header.Filename), ".png") {
			mimeType = "image/png"
		} else {
			mimeType = "image/jpeg"
		}
	}

	artworkInfo, err := h.artworkManager.UploadArtwork(r.Context(), trackID, imageData, mimeType)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, artworkInfo)
}

type ApplyArtworkRequest struct {
	SourceTrackID  string   `json:"sourceTrackId"`
	TargetTrackIDs []string `json:"targetTrackIds"`
}

func (h *Handler) ApplyArtwork(w http.ResponseWriter, r *http.Request) {
	var req ApplyArtworkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.SourceTrackID == "" {
		h.respondError(w, http.StatusBadRequest, "Source track ID is required")
		return
	}

	if len(req.TargetTrackIDs) == 0 {
		h.respondError(w, http.StatusBadRequest, "No target track IDs provided")
		return
	}

	results, err := h.artworkManager.ApplyArtworkToTracks(r.Context(), req.SourceTrackID, req.TargetTrackIDs)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"results": results,
	})
}

func (h *Handler) GetArtworkSuggestions(w http.ResponseWriter, r *http.Request) {
	trackID := chi.URLParam(r, "id")

	suggestions, err := h.artworkManager.GetSuggestions(r.Context(), trackID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, suggestions)
}

// Audio Scan handlers

func (h *Handler) GetAudioScanManifest(w http.ResponseWriter, r *http.Request) {
	trackID := chi.URLParam(r, "id")

	track, err := h.db.GetTrack(r.Context(), trackID)
	if err != nil {
		h.respondError(w, http.StatusNotFound, "Track not found")
		return
	}

	// Build the manifest path
	prefix := trackID
	if len(trackID) >= 2 {
		prefix = trackID[:2]
	}
	manifestPath := fmt.Sprintf("tracks/%s/%s/analysis_manifest_v1.json", prefix, trackID)

	// Return the manifest info (UI can fetch the actual file from /artifacts/)
	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"trackId":      trackID,
		"manifestPath": manifestPath,
		"artifactsUrl": fmt.Sprintf("/artifacts/tracks/%s/%s/", prefix, trackID),
		"track":        track,
	})
}

type RunAudioScanRequest struct {
	MaxDurationSec float64 `json:"maxDurationSec,omitempty"`
}

func (h *Handler) RunAudioScan(w http.ResponseWriter, r *http.Request) {
	trackID := chi.URLParam(r, "id")

	// Check if track exists
	_, err := h.db.GetTrack(r.Context(), trackID)
	if err != nil {
		h.respondError(w, http.StatusNotFound, "Track not found")
		return
	}

	// Parse optional request body
	var req RunAudioScanRequest
	json.NewDecoder(r.Body).Decode(&req) // Ignore errors for optional body

	// Queue the audio scan job
	job := &models.Job{
		Type:        "audioscan",
		TargetType:  "track",
		TargetID:    trackID,
		Status:      models.StatusQueued,
		MaxAttempts: 3,
	}

	if err := h.db.CreateJob(r.Context(), job); err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to queue audio scan job")
		return
	}

	h.respondJSON(w, http.StatusAccepted, map[string]interface{}{
		"status":  "queued",
		"jobId":   job.ID,
		"trackId": trackID,
		"message": "Audio scan job queued",
	})
}

// RunBulkAudioScan queues audio scan jobs for all tracks (optionally filtered by library)
func (h *Handler) RunBulkAudioScan(w http.ResponseWriter, r *http.Request) {
	libraryID := r.URL.Query().Get("library_id")
	skipExisting := r.URL.Query().Get("skip_existing") == "true"

	// Get all tracks
	tracks, total, err := h.db.ListTracks(r.Context(), libraryID, "", 10000, 0)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	queued := 0
	skipped := 0

	for _, track := range tracks {
		// Optionally skip tracks that already have analysis
		if skipExisting {
			// Check if manifest exists by trying to load it
			// For now, we'll skip this check and just queue all
		}

		job := &models.Job{
			Type:        "audioscan",
			TargetType:  "track",
			TargetID:    track.ID,
			Status:      models.StatusQueued,
			MaxAttempts: 3,
		}

		if err := h.db.CreateJob(r.Context(), job); err != nil {
			log.Warn().Err(err).Str("trackId", track.ID).Msg("Failed to queue audio scan job")
			skipped++
			continue
		}
		queued++
	}

	h.respondJSON(w, http.StatusAccepted, map[string]interface{}{
		"status":  "queued",
		"total":   total,
		"queued":  queued,
		"skipped": skipped,
		"message": fmt.Sprintf("Queued %d audio scan jobs", queued),
	})
}

// Health check

func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	h.respondJSON(w, http.StatusOK, map[string]string{
		"status":  "healthy",
		"version": "1.0.0",
	})
}

// Job logs - for verbose output during scans

// JobLoggerInterface allows dependency injection of the job logger
type JobLoggerInterface interface {
	GetLogSince(jobID string, sinceIndex int) ([]interface{}, int, string)
	GetRecentJobs(limit int) interface{}
	GetLog(jobID string) interface{}
}

var globalJobLogger JobLoggerInterface

// SetJobLogger sets the global job logger for handlers
func SetJobLogger(logger JobLoggerInterface) {
	globalJobLogger = logger
}

func (h *Handler) GetJobLogs(w http.ResponseWriter, r *http.Request) {
	jobID := chi.URLParam(r, "id")
	sinceStr := r.URL.Query().Get("since")

	if globalJobLogger == nil {
		h.respondError(w, http.StatusServiceUnavailable, "Job logger not available")
		return
	}

	var sinceIndex int
	if sinceStr != "" {
		if parsed, err := strconv.Atoi(sinceStr); err == nil {
			sinceIndex = parsed
		}
	}

	entries, nextIndex, status := globalJobLogger.GetLogSince(jobID, sinceIndex)

	if entries == nil {
		h.respondError(w, http.StatusNotFound, "Job not found")
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"entries":   entries,
		"nextIndex": nextIndex,
		"status":    status,
	})
}

func (h *Handler) ListJobLogs(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 20
	if limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	if globalJobLogger == nil {
		h.respondJSON(w, http.StatusOK, map[string]interface{}{
			"jobs": []interface{}{},
		})
		return
	}

	jobs := globalJobLogger.GetRecentJobs(limit)

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"jobs": jobs,
	})
}
