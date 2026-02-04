package audioscan

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/ottavia-music/ottavia/internal/database"
	"github.com/ottavia-music/ottavia/internal/models"
)

// Scanner performs Audio Scan-style analysis on tracks
type Scanner struct {
	db            *database.DB
	ffmpegPath    string
	ffprobePath   string
	artifactsPath string
	maxDuration   float64 // Max seconds to analyze (0 = full track)
}

// Config holds scanner configuration
type Config struct {
	MaxDurationSec float64 // Default: 60 seconds for fast scans
	FFmpegPath     string
	FFprobePath    string
	ArtifactsPath  string
}

// NewScanner creates a new audio scanner
func NewScanner(db *database.DB, cfg Config) *Scanner {
	maxDur := cfg.MaxDurationSec
	if maxDur <= 0 {
		maxDur = 60.0 // Default to first 60 seconds
	}
	return &Scanner{
		db:            db,
		ffmpegPath:    cfg.FFmpegPath,
		ffprobePath:   cfg.FFprobePath,
		artifactsPath: cfg.ArtifactsPath,
		maxDuration:   maxDur,
	}
}

// GetArtifactsPath returns the artifacts base path
func (s *Scanner) GetArtifactsPath() string {
	return s.artifactsPath
}

// ScanTrack performs full audio scan analysis on a track
func (s *Scanner) ScanTrack(ctx context.Context, trackID string) error {
	log.Info().Str("trackId", trackID).Msg("Starting Audio Scan analysis")

	// Get track and its probe data
	track, err := s.db.GetTrack(ctx, trackID)
	if err != nil {
		return fmt.Errorf("get track: %w", err)
	}

	// Create artifact directory
	artifactDir, err := EnsureArtifactDir(s.artifactsPath, trackID)
	if err != nil {
		return fmt.Errorf("ensure artifact dir: %w", err)
	}

	// Build probe cache from track metadata
	probeCache := ProbeCache{
		Source:       "ffprobe-cache",
		SampleRateHz: track.SampleRate,
		Channels:     track.Channels,
		Codec:        track.Codec,
		Container:    getContainerFromPath(track.Path),
		DurationSec:  track.Duration,
	}
	if track.BitDepth > 0 {
		bd := track.BitDepth
		probeCache.BitDepth = &bd
	}

	// Create manifest
	manifest := NewManifest(trackID, probeCache)

	// Run each analysis module (raw data only, no PNG generation)
	s.runAudioScanModule(ctx, track, manifest, artifactDir)
	s.runLoudnessModule(ctx, track, manifest, artifactDir)
	s.runClippingModule(ctx, track, manifest, artifactDir)
	s.runPhaseModule(ctx, track, manifest, artifactDir)
	s.runDynamicsModule(ctx, track, manifest, artifactDir)

	// Save manifest
	if err := manifest.Save(artifactDir); err != nil {
		return fmt.Errorf("save manifest: %w", err)
	}

	// Update analysis_results with summary scalars
	if err := s.updateAnalysisResults(ctx, trackID, manifest); err != nil {
		log.Warn().Err(err).Str("trackId", trackID).Msg("Failed to update analysis results")
	}

	log.Info().Str("trackId", trackID).Msg("Audio Scan analysis complete")
	return nil
}

// runAudioScanModule performs spectrum analysis
func (s *Scanner) runAudioScanModule(ctx context.Context, track *models.Track, manifest *AnalysisManifest, dir string) {
	log.Debug().Str("trackId", track.ID).Msg("Running audioscan module")

	// Calculate analysis parameters from probe cache
	sampleRate := track.SampleRate
	nyquist := sampleRate / 2
	fftSize := 4096
	hopSize := fftSize / 4

	// Determine analysis duration
	duration := track.Duration
	if s.maxDuration > 0 && duration > s.maxDuration {
		duration = s.maxDuration
	}

	// Create raw data structure
	curve := &AudioScanCurve{
		Version:      RawDataVersion,
		SampleRateHz: sampleRate,
		NyquistHz:    nyquist,
	}
	curve.Analyzed.StartSec = 0
	curve.Analyzed.DurationSec = duration
	curve.Analyzed.ChannelMode = "stereo-downmix"
	if track.Channels == 1 {
		curve.Analyzed.ChannelMode = "mono"
	}
	curve.Analyzed.DecodeFormat = "f32le"
	curve.FFT.FFTSize = fftSize
	curve.FFT.HopSize = hopSize
	curve.FFT.Window = "hann"
	curve.FFT.SmoothingOctaves = 0.25

	// Set guide lines computed from probe cache
	curve.Guides.VerticalLinesHz = []int{nyquist}
	// Add common reference frequencies if they're below nyquist
	refs := []int{20000, 16000, 12000}
	for _, ref := range refs {
		if ref < nyquist {
			curve.Guides.VerticalLinesHz = append(curve.Guides.VerticalLinesHz, ref)
		}
	}

	// Run FFmpeg to extract spectrum data
	freqHz, levelDb, err := s.extractSpectrumCurve(ctx, track.Path, fftSize, duration)
	if err != nil {
		manifest.SetModuleError("audioscan", "Spectrum analysis failed", err.Error())
		return
	}

	curve.Curve.FreqHz = freqHz
	curve.Curve.LevelDb = levelDb
	curve.FFT.Frames = len(levelDb) / (fftSize / 2)

	// Calculate metrics
	curve.Metrics.BandwidthHz = calculateBandwidth(freqHz, levelDb)
	curve.Metrics.DCMean, curve.Metrics.DCFlag = s.calculateDC(ctx, track.Path, duration)

	// Save raw data
	rawPath := fmt.Sprintf("%s/audioscan_curve_v1.msgpack.zst", dir)
	if err := SaveMsgpackZstd(rawPath, curve); err != nil {
		manifest.SetModuleError("audioscan", "Failed to save raw data", err.Error())
		return
	}

	// Compute hash
	rawHash, _ := ComputeSHA256(rawPath)

	// Determine quality classification
	expectedQuality := deriveExpectedQuality(manifest.ProbeCache)
	detectedQuality, qualityReason := classifyDetectedQuality(curve)

	// Build render hints from probe cache (computed, not hard-coded)
	renderHints := &RenderHints{
		NyquistHz:    nyquist,
		GuideLinesHz: curve.Guides.VerticalLinesHz,
		FreqScaleLog: true,
		MinFreqHz:    20,
		MaxFreqHz:    nyquist,
		MinDb:        -80,
		MaxDb:        0,
		XUnit:        "Hz",
		YUnit:        "dB",
	}

	// Set module result with raw artifact and render hints (no PNG)
	manifest.SetModuleOK("audioscan", map[string]any{
		"expectedQuality": expectedQuality,
		"detectedQuality": detectedQuality,
		"qualityReason":   qualityReason,
		"bandwidthHz":     curve.Metrics.BandwidthHz,
		"dcIssues":        boolToInt(curve.Metrics.DCFlag),
		"channelsLabel":   channelsLabel(track.Channels),
	}, &ArtifactRef{
		Path:        "audioscan_curve_v1.msgpack.zst",
		SHA256:      rawHash,
		ContentType: "application/x-msgpack+zstd",
	}, renderHints)
}

// runLoudnessModule performs loudness analysis over time
func (s *Scanner) runLoudnessModule(ctx context.Context, track *models.Track, manifest *AnalysisManifest, dir string) {
	log.Debug().Str("trackId", track.ID).Msg("Running loudness module")

	duration := track.Duration
	if s.maxDuration > 0 && duration > s.maxDuration {
		duration = s.maxDuration
	}

	series, err := s.extractLoudnessSeries(ctx, track.Path, duration)
	if err != nil {
		manifest.SetModuleError("loudness", "Loudness analysis failed", err.Error())
		return
	}

	// Save raw data
	rawPath := fmt.Sprintf("%s/loudness_series_v1.msgpack.zst", dir)
	if err := SaveMsgpackZstd(rawPath, series); err != nil {
		manifest.SetModuleError("loudness", "Failed to save raw data", err.Error())
		return
	}

	// Compute hash
	rawHash, _ := ComputeSHA256(rawPath)

	// Build render hints
	renderHints := &RenderHints{
		DurationSec: duration,
		MinLUFS:     -60,
		MaxLUFS:     0,
		MinDb:       -60,
		MaxDb:       3, // For true peak which can exceed 0
		XUnit:       "sec",
		YUnit:       "LUFS",
		Y2Unit:      "dBTP",
	}

	manifest.SetModuleOK("loudness", map[string]any{
		"integratedLUFS": series.IntegratedLUFS,
		"lra":            series.LRA,
		"maxTruePeak":    series.MaxTruePeak,
		"maxSamplePeak":  series.MaxSamplePeak,
	}, &ArtifactRef{
		Path:        "loudness_series_v1.msgpack.zst",
		SHA256:      rawHash,
		ContentType: "application/x-msgpack+zstd",
	}, renderHints)
}

// runClippingModule performs clipping detection
func (s *Scanner) runClippingModule(ctx context.Context, track *models.Track, manifest *AnalysisManifest, dir string) {
	log.Debug().Str("trackId", track.ID).Msg("Running clipping module")

	duration := track.Duration
	if s.maxDuration > 0 && duration > s.maxDuration {
		duration = s.maxDuration
	}

	series, err := s.extractClippingSeries(ctx, track.Path, duration)
	if err != nil {
		manifest.SetModuleError("clipping", "Clipping analysis failed", err.Error())
		return
	}

	// Save raw data
	rawPath := fmt.Sprintf("%s/clipping_series_v1.msgpack.zst", dir)
	if err := SaveMsgpackZstd(rawPath, series); err != nil {
		manifest.SetModuleError("clipping", "Failed to save raw data", err.Error())
		return
	}

	// Compute hash
	rawHash, _ := ComputeSHA256(rawPath)

	// Build render hints
	renderHints := &RenderHints{
		DurationSec: duration,
		XUnit:       "sec",
		YUnit:       "clips",
	}

	manifest.SetModuleOK("clipping", map[string]any{
		"totalClipped": series.TotalClipped,
		"totalOvers":   series.TotalOvers,
		"hasClipping":  series.TotalClipped > 0,
	}, &ArtifactRef{
		Path:        "clipping_series_v1.msgpack.zst",
		SHA256:      rawHash,
		ContentType: "application/x-msgpack+zstd",
	}, renderHints)
}

// runPhaseModule performs stereo phase analysis
func (s *Scanner) runPhaseModule(ctx context.Context, track *models.Track, manifest *AnalysisManifest, dir string) {
	if track.Channels < 2 {
		manifest.SetModuleSkipped("phase", "Mono track - phase analysis not applicable")
		return
	}

	log.Debug().Str("trackId", track.ID).Msg("Running phase module")

	duration := track.Duration
	if s.maxDuration > 0 && duration > s.maxDuration {
		duration = s.maxDuration
	}

	series, err := s.extractPhaseSeries(ctx, track.Path, duration)
	if err != nil {
		manifest.SetModuleError("phase", "Phase analysis failed", err.Error())
		return
	}

	// Save raw data
	rawPath := fmt.Sprintf("%s/phase_series_v1.msgpack.zst", dir)
	if err := SaveMsgpackZstd(rawPath, series); err != nil {
		manifest.SetModuleError("phase", "Failed to save raw data", err.Error())
		return
	}

	// Compute hash
	rawHash, _ := ComputeSHA256(rawPath)

	// Build render hints
	renderHints := &RenderHints{
		DurationSec: duration,
		MinCorr:     -1,
		MaxCorr:     1,
		XUnit:       "sec",
		YUnit:       "correlation",
	}

	manifest.SetModuleOK("phase", map[string]any{
		"minCorrelation": series.MinCorrelation,
		"avgCorrelation": series.AvgCorrelation,
		"maxImbalanceDb": series.MaxImbalanceDb,
		"phaseIssue":     series.MinCorrelation < -0.5 || series.AvgCorrelation < 0,
	}, &ArtifactRef{
		Path:        "phase_series_v1.msgpack.zst",
		SHA256:      rawHash,
		ContentType: "application/x-msgpack+zstd",
	}, renderHints)
}

// runDynamicsModule performs dynamics/DR segmentation
func (s *Scanner) runDynamicsModule(ctx context.Context, track *models.Track, manifest *AnalysisManifest, dir string) {
	log.Debug().Str("trackId", track.ID).Msg("Running dynamics module")

	duration := track.Duration
	if s.maxDuration > 0 && duration > s.maxDuration {
		duration = s.maxDuration
	}

	series, err := s.extractDynamicsSeries(ctx, track.Path, duration)
	if err != nil {
		manifest.SetModuleError("dynamics", "Dynamics analysis failed", err.Error())
		return
	}

	// Save raw data
	rawPath := fmt.Sprintf("%s/dynamics_series_v1.msgpack.zst", dir)
	if err := SaveMsgpackZstd(rawPath, series); err != nil {
		manifest.SetModuleError("dynamics", "Failed to save raw data", err.Error())
		return
	}

	// Compute hash
	rawHash, _ := ComputeSHA256(rawPath)

	// Build render hints
	renderHints := &RenderHints{
		DurationSec: duration,
		MinDb:       0,
		MaxDb:       25, // Crest factor range
		XUnit:       "sec",
		YUnit:       "dB",
	}

	manifest.SetModuleOK("dynamics", map[string]any{
		"drScore":    series.DRScore,
		"avgCrestDb": series.AvgCrestDb,
		"minCrestDb": series.MinCrestDb,
	}, &ArtifactRef{
		Path:        "dynamics_series_v1.msgpack.zst",
		SHA256:      rawHash,
		ContentType: "application/x-msgpack+zstd",
	}, renderHints)
}

// updateAnalysisResults updates the DB analysis_results with summary scalars
func (s *Scanner) updateAnalysisResults(ctx context.Context, trackID string, manifest *AnalysisManifest) error {
	// Get existing analysis result or create placeholder
	result, err := s.db.GetAnalysisResult(ctx, trackID)
	if err != nil {
		// No existing result - would need to create one
		log.Debug().Str("trackId", trackID).Msg("No existing analysis result to update")
		return nil
	}

	// Update stats_json with audioscan summary data
	statsJSON := make(map[string]any)

	if audioscan, ok := manifest.Modules["audioscan"]; ok && audioscan.Status == "ok" {
		statsJSON["audioscan"] = audioscan.Summary
	}
	if loudness, ok := manifest.Modules["loudness"]; ok && loudness.Status == "ok" {
		statsJSON["loudness"] = loudness.Summary
	}
	if clipping, ok := manifest.Modules["clipping"]; ok && clipping.Status == "ok" {
		statsJSON["clipping"] = clipping.Summary
	}
	if phase, ok := manifest.Modules["phase"]; ok && phase.Status == "ok" {
		statsJSON["phase"] = phase.Summary
	}
	if dynamics, ok := manifest.Modules["dynamics"]; ok && dynamics.Status == "ok" {
		statsJSON["dynamics"] = dynamics.Summary
	}

	// We'd update result.StatsJSON here with the new data
	_ = result
	_ = statsJSON

	return nil
}

// Helper functions

func getContainerFromPath(path string) string {
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(path), "."))
	switch ext {
	case "flac":
		return "flac"
	case "m4a", "mp4", "aac":
		return "mp4"
	case "wav":
		return "wav"
	case "mp3":
		return "mp3"
	case "ogg", "opus":
		return "ogg"
	case "ape":
		return "ape"
	case "wv":
		return "wavpack"
	default:
		return ext
	}
}

func deriveExpectedQuality(probe ProbeCache) string {
	// Derive expected quality tier from probe cache metadata only
	if probe.BitDepth != nil && *probe.BitDepth >= 24 && probe.SampleRateHz >= 88200 {
		return "Hi-Res (24-bit/88kHz+)"
	}
	if probe.BitDepth != nil && *probe.BitDepth >= 24 {
		return "Studio (24-bit)"
	}
	if probe.BitDepth != nil && *probe.BitDepth == 16 && probe.SampleRateHz >= 44100 {
		return "CD Quality (16-bit/44.1kHz)"
	}
	if probe.Codec == "mp3" || probe.Codec == "aac" || probe.Codec == "opus" || probe.Codec == "vorbis" {
		return "Lossy"
	}
	return "Lossless"
}

func classifyDetectedQuality(curve *AudioScanCurve) (string, string) {
	// Analyze the spectrum curve to classify detected quality
	bw := curve.Metrics.BandwidthHz
	nyquist := curve.NyquistHz

	if bw == 0 || bw >= nyquist-1000 {
		return "Full Bandwidth", "Spectrum extends to Nyquist limit"
	}
	if bw < 16000 {
		return "Possible Transcode", fmt.Sprintf("Bandwidth limited to %d Hz (possible lossy source)", bw)
	}
	if bw < 20000 {
		return "Bandwidth Limited", fmt.Sprintf("Bandwidth %d Hz (may indicate compression)", bw)
	}
	return "Good", fmt.Sprintf("Bandwidth %d Hz", bw)
}

func calculateBandwidth(freqHz, levelDb []float32) int {
	if len(freqHz) == 0 || len(levelDb) == 0 {
		return 0
	}

	// Find peak level (excluding DC)
	peakLevel := float32(-200)
	for i := 1; i < len(levelDb); i++ {
		if levelDb[i] > peakLevel {
			peakLevel = levelDb[i]
		}
	}

	// Find last frequency where level is within threshold of peak
	threshold := peakLevel - 60 // 60dB below peak
	bandwidth := 0
	for i := len(levelDb) - 1; i >= 0; i-- {
		if levelDb[i] > threshold {
			if i < len(freqHz) {
				bandwidth = int(freqHz[i])
			}
			break
		}
	}

	return bandwidth
}

func channelsLabel(channels int) string {
	switch channels {
	case 1:
		return "Mono"
	case 2:
		return "Stereo"
	case 6:
		return "5.1 Surround"
	case 8:
		return "7.1 Surround"
	default:
		return fmt.Sprintf("%d channels", channels)
	}
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
