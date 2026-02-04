package audioscan

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Version constants
const (
	ManifestVersion = 1
	RawDataVersion  = 1
)

// AnalysisManifest is the top-level manifest for all analysis artifacts (v1)
type AnalysisManifest struct {
	Version     int                      `json:"version"`
	TrackID     string                   `json:"trackId"`
	GeneratedAt string                   `json:"generatedAt"` // RFC3339
	ProbeCache  ProbeCache               `json:"probeCache"`
	Modules     map[string]*ModuleResult `json:"modules"`
}

// ProbeCache contains cached ffprobe metadata for this track
type ProbeCache struct {
	Source       string  `json:"source"` // "ffprobe-cache"
	SampleRateHz int     `json:"sampleRateHz"`
	BitDepth     *int    `json:"bitDepth"` // nullable for lossy codecs
	Channels     int     `json:"channels"`
	Codec        string  `json:"codec"`
	Container    string  `json:"container"`
	DurationSec  float64 `json:"durationSec"`
}

// ModuleResult contains the result of one analysis module
type ModuleResult struct {
	Status      string         `json:"status"` // "ok", "error", "skipped"
	Summary     map[string]any `json:"summary,omitempty"`
	Raw         *ArtifactRef   `json:"raw,omitempty"`
	RenderHints *RenderHints   `json:"renderHints,omitempty"`
	Error       *ModuleError   `json:"error,omitempty"`
}

// ArtifactRef references a file artifact
type ArtifactRef struct {
	Path        string `json:"path"`
	SHA256      string `json:"sha256"`
	ContentType string `json:"contentType"`
}

// RenderHints provides rendering hints for the UI (computed from probe cache)
type RenderHints struct {
	// Spectrum-specific
	NyquistHz      int   `json:"nyquistHz,omitempty"`
	GuideLinesHz   []int `json:"guideLinesHz,omitempty"`
	FreqScaleLog   bool  `json:"freqScaleLog,omitempty"`
	MinFreqHz      int   `json:"minFreqHz,omitempty"`
	MaxFreqHz      int   `json:"maxFreqHz,omitempty"`

	// dB range hints
	MinDb float32 `json:"minDb,omitempty"`
	MaxDb float32 `json:"maxDb,omitempty"`

	// Time-series hints
	DurationSec float64 `json:"durationSec,omitempty"`

	// LUFS-specific
	MinLUFS float32 `json:"minLUFS,omitempty"`
	MaxLUFS float32 `json:"maxLUFS,omitempty"`

	// Correlation-specific
	MinCorr float32 `json:"minCorr,omitempty"`
	MaxCorr float32 `json:"maxCorr,omitempty"`

	// Units for axes
	XUnit string `json:"xUnit,omitempty"` // "Hz", "sec"
	YUnit string `json:"yUnit,omitempty"` // "dB", "LUFS", "correlation"
	Y2Unit string `json:"y2Unit,omitempty"` // optional second axis
}

// ModuleError contains error details for failed modules
type ModuleError struct {
	Message string `json:"message"`
	Detail  string `json:"detail,omitempty"`
}

// NewManifest creates a new analysis manifest
func NewManifest(trackID string, probeCache ProbeCache) *AnalysisManifest {
	return &AnalysisManifest{
		Version:     ManifestVersion,
		TrackID:     trackID,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		ProbeCache:  probeCache,
		Modules:     make(map[string]*ModuleResult),
	}
}

// SetModuleOK marks a module as successfully completed
func (m *AnalysisManifest) SetModuleOK(name string, summary map[string]any, raw *ArtifactRef, hints *RenderHints) {
	m.Modules[name] = &ModuleResult{
		Status:      "ok",
		Summary:     summary,
		Raw:         raw,
		RenderHints: hints,
	}
}

// SetModuleError marks a module as failed
func (m *AnalysisManifest) SetModuleError(name string, message, detail string) {
	m.Modules[name] = &ModuleResult{
		Status: "error",
		Error: &ModuleError{
			Message: message,
			Detail:  detail,
		},
	}
}

// SetModuleSkipped marks a module as skipped
func (m *AnalysisManifest) SetModuleSkipped(name string, reason string) {
	m.Modules[name] = &ModuleResult{
		Status: "skipped",
		Error: &ModuleError{
			Message: reason,
		},
	}
}

// Save writes the manifest to disk
func (m *AnalysisManifest) Save(dir string) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}

	path := filepath.Join(dir, "analysis_manifest_v1.json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}

	return nil
}

// LoadManifest reads a manifest from disk
func LoadManifest(dir string) (*AnalysisManifest, error) {
	path := filepath.Join(dir, "analysis_manifest_v1.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}

	var m AnalysisManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("unmarshal manifest: %w", err)
	}

	return &m, nil
}

// ComputeSHA256 computes SHA256 hash of a file
func ComputeSHA256(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}

// ArtifactDir returns the artifact directory path for a track
func ArtifactDir(basePath, trackID string) string {
	// Use first 2 chars of trackID as subdirectory for distribution
	prefix := trackID[:2]
	return filepath.Join(basePath, "tracks", prefix, trackID)
}

// EnsureArtifactDir creates the artifact directory if it doesn't exist
func EnsureArtifactDir(basePath, trackID string) (string, error) {
	dir := ArtifactDir(basePath, trackID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("create artifact dir: %w", err)
	}
	return dir, nil
}
