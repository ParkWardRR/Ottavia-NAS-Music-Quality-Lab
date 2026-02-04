package audioscan

import (
	"encoding/json"
	"math"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// SeriesResponse is the JSON response for series data endpoints
type SeriesResponse struct {
	Version     int                    `json:"version"`
	Module      string                 `json:"module"`
	Units       map[string]string      `json:"units"`
	RenderHints *RenderHints           `json:"renderHints"`
	Series      map[string][]float64   `json:"series"`
}

// ManifestResponse is the JSON response for the manifest endpoint
type ManifestResponse struct {
	TrackID     string                   `json:"trackId"`
	GeneratedAt string                   `json:"generatedAt"`
	ProbeCache  ProbeCache               `json:"probeCache"`
	Modules     map[string]*ModuleResult `json:"modules"`
}

// APIHandler handles audio scan API requests
type APIHandler struct {
	scanner *Scanner
}

// NewAPIHandler creates a new API handler
func NewAPIHandler(scanner *Scanner) *APIHandler {
	return &APIHandler{scanner: scanner}
}

// GetManifest returns the analysis manifest for a track
func (h *APIHandler) GetManifest(w http.ResponseWriter, r *http.Request) {
	trackID := chi.URLParam(r, "id")

	artifactDir := ArtifactDir(h.scanner.GetArtifactsPath(), trackID)
	manifest, err := LoadManifest(artifactDir)
	if err != nil {
		http.Error(w, "Analysis not found", http.StatusNotFound)
		return
	}

	resp := ManifestResponse{
		TrackID:     manifest.TrackID,
		GeneratedAt: manifest.GeneratedAt,
		ProbeCache:  manifest.ProbeCache,
		Modules:     manifest.Modules,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// GetSeries returns decimated series data for a module
func (h *APIHandler) GetSeries(w http.ResponseWriter, r *http.Request) {
	trackID := chi.URLParam(r, "id")
	module := r.URL.Query().Get("module")

	if module == "" {
		http.Error(w, "module parameter required", http.StatusBadRequest)
		return
	}

	// Parse maxPoints (default 1500, max 5000)
	maxPoints := 1500
	if mp := r.URL.Query().Get("maxPoints"); mp != "" {
		if v, err := strconv.Atoi(mp); err == nil && v > 0 {
			maxPoints = v
		}
	}
	if maxPoints > 5000 {
		maxPoints = 5000
	}

	// Parse optional time range
	var startSec, endSec float64 = 0, -1
	if s := r.URL.Query().Get("startSec"); s != "" {
		if v, err := strconv.ParseFloat(s, 64); err == nil {
			startSec = v
		}
	}
	if e := r.URL.Query().Get("endSec"); e != "" {
		if v, err := strconv.ParseFloat(e, 64); err == nil {
			endSec = v
		}
	}

	artifactDir := ArtifactDir(h.scanner.GetArtifactsPath(), trackID)

	// Load manifest for render hints
	manifest, err := LoadManifest(artifactDir)
	if err != nil {
		http.Error(w, "Analysis not found", http.StatusNotFound)
		return
	}

	moduleResult, ok := manifest.Modules[module]
	if !ok || moduleResult.Status != "ok" {
		http.Error(w, "Module not found or failed", http.StatusNotFound)
		return
	}

	// Load and decimate series based on module type
	var resp *SeriesResponse
	switch module {
	case "audioscan":
		resp, err = h.loadAudioScanSeries(artifactDir, maxPoints, moduleResult.RenderHints)
	case "loudness":
		resp, err = h.loadLoudnessSeries(artifactDir, maxPoints, startSec, endSec, moduleResult.RenderHints)
	case "clipping":
		resp, err = h.loadClippingSeries(artifactDir, maxPoints, startSec, endSec, moduleResult.RenderHints)
	case "phase":
		resp, err = h.loadPhaseSeries(artifactDir, maxPoints, startSec, endSec, moduleResult.RenderHints)
	case "dynamics":
		resp, err = h.loadDynamicsSeries(artifactDir, maxPoints, startSec, endSec, moduleResult.RenderHints)
	default:
		http.Error(w, "Unknown module", http.StatusBadRequest)
		return
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *APIHandler) loadAudioScanSeries(dir string, maxPoints int, hints *RenderHints) (*SeriesResponse, error) {
	var curve AudioScanCurve
	if err := LoadMsgpackZstd(dir+"/audioscan_curve_v1.msgpack.zst", &curve); err != nil {
		return nil, err
	}

	// Convert float32 to float64 and decimate
	freqHz := float32ToFloat64(curve.Curve.FreqHz)
	levelDb := float32ToFloat64(curve.Curve.LevelDb)

	// Decimate if needed
	if len(freqHz) > maxPoints {
		freqHz, levelDb = decimateLTTB(freqHz, levelDb, maxPoints)
	}

	return &SeriesResponse{
		Version: 1,
		Module:  "audioscan",
		Units: map[string]string{
			"x": "Hz",
			"y": "dB",
		},
		RenderHints: hints,
		Series: map[string][]float64{
			"x": freqHz,
			"y": levelDb,
		},
	}, nil
}

func (h *APIHandler) loadLoudnessSeries(dir string, maxPoints int, startSec, endSec float64, hints *RenderHints) (*SeriesResponse, error) {
	var series LoudnessSeries
	if err := LoadMsgpackZstd(dir+"/loudness_series_v1.msgpack.zst", &series); err != nil {
		return nil, err
	}

	// Convert and filter by time range
	tSec := float32ToFloat64(series.TSec)
	momentary := float32ToFloat64(series.MomentaryLUFS)
	shortTerm := float32ToFloat64(series.ShortTermLUFS)
	truePeak := float32ToFloat64(series.TruePeakDbTP)

	// Filter by time range if specified
	if endSec > 0 {
		tSec, momentary, shortTerm, truePeak = filterTimeRange(tSec, startSec, endSec, momentary, shortTerm, truePeak)
	}

	// Decimate preserving peaks (use min/max envelope for true peak)
	if len(tSec) > maxPoints {
		tSec, momentary = decimateLTTB(tSec, momentary, maxPoints)
		_, shortTerm = decimateLTTB(tSec, shortTerm, maxPoints)
		_, truePeak = decimateMinMaxEnvelope(tSec, truePeak, maxPoints)
	}

	return &SeriesResponse{
		Version: 1,
		Module:  "loudness",
		Units: map[string]string{
			"x":         "sec",
			"momentary": "LUFS",
			"shortTerm": "LUFS",
			"truePeak":  "dBTP",
		},
		RenderHints: hints,
		Series: map[string][]float64{
			"x":         tSec,
			"momentary": momentary,
			"shortTerm": shortTerm,
			"truePeak":  truePeak,
		},
	}, nil
}

func (h *APIHandler) loadClippingSeries(dir string, maxPoints int, startSec, endSec float64, hints *RenderHints) (*SeriesResponse, error) {
	var series ClippingSeries
	if err := LoadMsgpackZstd(dir+"/clipping_series_v1.msgpack.zst", &series); err != nil {
		return nil, err
	}

	tSec := float32ToFloat64(series.TSec)
	clipped := intToFloat64(series.ClippedSamples)
	overs := intToFloat64(series.OversCount)

	// Filter by time range
	if endSec > 0 {
		tSec, clipped, overs, _ = filterTimeRange(tSec, startSec, endSec, clipped, overs, nil)
	}

	// Use min/max envelope to preserve peaks
	if len(tSec) > maxPoints {
		tSec, clipped = decimateMinMaxEnvelope(tSec, clipped, maxPoints)
		_, overs = decimateMinMaxEnvelope(tSec, overs, maxPoints)
	}

	return &SeriesResponse{
		Version: 1,
		Module:  "clipping",
		Units: map[string]string{
			"x":       "sec",
			"clipped": "samples",
			"overs":   "samples",
		},
		RenderHints: hints,
		Series: map[string][]float64{
			"x":       tSec,
			"clipped": clipped,
			"overs":   overs,
		},
	}, nil
}

func (h *APIHandler) loadPhaseSeries(dir string, maxPoints int, startSec, endSec float64, hints *RenderHints) (*SeriesResponse, error) {
	var series PhaseSeries
	if err := LoadMsgpackZstd(dir+"/phase_series_v1.msgpack.zst", &series); err != nil {
		return nil, err
	}

	tSec := float32ToFloat64(series.TSec)
	correlation := float32ToFloat64(series.Correlation)
	lrBalance := float32ToFloat64(series.LRBalanceDb)

	// Filter by time range
	if endSec > 0 {
		tSec, correlation, lrBalance, _ = filterTimeRange(tSec, startSec, endSec, correlation, lrBalance, nil)
	}

	// Decimate
	if len(tSec) > maxPoints {
		tSec, correlation = decimateLTTB(tSec, correlation, maxPoints)
		_, lrBalance = decimateLTTB(tSec, lrBalance, maxPoints)
	}

	return &SeriesResponse{
		Version: 1,
		Module:  "phase",
		Units: map[string]string{
			"x":           "sec",
			"correlation": "",
			"lrBalance":   "dB",
		},
		RenderHints: hints,
		Series: map[string][]float64{
			"x":           tSec,
			"correlation": correlation,
			"lrBalance":   lrBalance,
		},
	}, nil
}

func (h *APIHandler) loadDynamicsSeries(dir string, maxPoints int, startSec, endSec float64, hints *RenderHints) (*SeriesResponse, error) {
	var series DynamicsSeries
	if err := LoadMsgpackZstd(dir+"/dynamics_series_v1.msgpack.zst", &series); err != nil {
		return nil, err
	}

	tSec := float32ToFloat64(series.TSec)
	crestFactor := float32ToFloat64(series.CrestFactorDb)
	rmsLevel := float32ToFloat64(series.RMSDb)
	peakLevel := float32ToFloat64(series.PeakDb)

	// Filter by time range
	if endSec > 0 {
		tSec, crestFactor, rmsLevel, peakLevel = filterTimeRange(tSec, startSec, endSec, crestFactor, rmsLevel, peakLevel)
	}

	// Decimate
	if len(tSec) > maxPoints {
		tSec, crestFactor = decimateLTTB(tSec, crestFactor, maxPoints)
		_, rmsLevel = decimateLTTB(tSec, rmsLevel, maxPoints)
		_, peakLevel = decimateMinMaxEnvelope(tSec, peakLevel, maxPoints)
	}

	return &SeriesResponse{
		Version: 1,
		Module:  "dynamics",
		Units: map[string]string{
			"x":           "sec",
			"crestFactor": "dB",
			"rmsLevel":    "dB",
			"peakLevel":   "dB",
		},
		RenderHints: hints,
		Series: map[string][]float64{
			"x":           tSec,
			"crestFactor": crestFactor,
			"rmsLevel":    rmsLevel,
			"peakLevel":   peakLevel,
		},
	}, nil
}

// Decimation functions

// decimateLTTB implements Largest-Triangle-Three-Buckets algorithm
// for downsampling while preserving visual appearance
func decimateLTTB(x, y []float64, targetPoints int) ([]float64, []float64) {
	n := len(x)
	if n <= targetPoints || targetPoints < 3 {
		return x, y
	}

	// Always keep first and last points
	outX := make([]float64, targetPoints)
	outY := make([]float64, targetPoints)
	outX[0] = x[0]
	outY[0] = y[0]
	outX[targetPoints-1] = x[n-1]
	outY[targetPoints-1] = y[n-1]

	// Bucket size
	bucketSize := float64(n-2) / float64(targetPoints-2)

	// Previous selected point
	prevX, prevY := x[0], y[0]

	for i := 1; i < targetPoints-1; i++ {
		// Current bucket range
		bucketStart := int(float64(i-1)*bucketSize) + 1
		bucketEnd := int(float64(i)*bucketSize) + 1
		if bucketEnd > n-1 {
			bucketEnd = n - 1
		}

		// Next bucket average (for triangle calculation)
		nextBucketStart := bucketEnd
		nextBucketEnd := int(float64(i+1)*bucketSize) + 1
		if nextBucketEnd > n-1 {
			nextBucketEnd = n - 1
		}

		avgX, avgY := 0.0, 0.0
		count := 0
		for j := nextBucketStart; j < nextBucketEnd; j++ {
			avgX += x[j]
			avgY += y[j]
			count++
		}
		if count > 0 {
			avgX /= float64(count)
			avgY /= float64(count)
		}

		// Find point in current bucket that maximizes triangle area
		maxArea := -1.0
		maxIdx := bucketStart
		for j := bucketStart; j < bucketEnd; j++ {
			// Triangle area using cross product
			area := math.Abs((prevX-avgX)*(y[j]-prevY) - (prevX-x[j])*(avgY-prevY))
			if area > maxArea {
				maxArea = area
				maxIdx = j
			}
		}

		outX[i] = x[maxIdx]
		outY[i] = y[maxIdx]
		prevX, prevY = x[maxIdx], y[maxIdx]
	}

	return outX, outY
}

// decimateMinMaxEnvelope preserves min and max values in each bucket
// Critical for clipping/peak data where we must not lose maximums
func decimateMinMaxEnvelope(x, y []float64, targetPoints int) ([]float64, []float64) {
	n := len(x)
	if n <= targetPoints {
		return x, y
	}

	// Each output point represents a bucket - we keep the max value
	bucketSize := float64(n) / float64(targetPoints)
	outX := make([]float64, targetPoints)
	outY := make([]float64, targetPoints)

	for i := 0; i < targetPoints; i++ {
		bucketStart := int(float64(i) * bucketSize)
		bucketEnd := int(float64(i+1) * bucketSize)
		if bucketEnd > n {
			bucketEnd = n
		}

		// Find max in bucket
		maxY := y[bucketStart]
		maxIdx := bucketStart
		for j := bucketStart; j < bucketEnd; j++ {
			if y[j] > maxY {
				maxY = y[j]
				maxIdx = j
			}
		}

		outX[i] = x[maxIdx]
		outY[i] = maxY
	}

	return outX, outY
}

// Helper functions

func float32ToFloat64(arr []float32) []float64 {
	result := make([]float64, len(arr))
	for i, v := range arr {
		result[i] = float64(v)
	}
	return result
}

func intToFloat64(arr []int) []float64 {
	result := make([]float64, len(arr))
	for i, v := range arr {
		result[i] = float64(v)
	}
	return result
}

func filterTimeRange(tSec []float64, startSec, endSec float64, series1, series2, series3 []float64) ([]float64, []float64, []float64, []float64) {
	var outT, out1, out2, out3 []float64

	for i, t := range tSec {
		if t >= startSec && t <= endSec {
			outT = append(outT, t)
			if i < len(series1) {
				out1 = append(out1, series1[i])
			}
			if series2 != nil && i < len(series2) {
				out2 = append(out2, series2[i])
			}
			if series3 != nil && i < len(series3) {
				out3 = append(out3, series3[i])
			}
		}
	}

	return outT, out1, out2, out3
}
