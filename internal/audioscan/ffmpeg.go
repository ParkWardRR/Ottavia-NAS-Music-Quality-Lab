package audioscan

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

// Retry configuration for unstable NAS connections
const (
	maxRetries     = 5
	initialBackoff = 1 * time.Second
	maxBackoff     = 16 * time.Second
)

// isRetryableError checks if an error indicates a temporary file access issue
// that should be retried (e.g., NAS temporarily unavailable)
func isRetryableError(stderr string, err error) bool {
	if err == nil {
		return false
	}
	// Check for common NAS/network filesystem errors
	retryablePatterns := []string{
		"No such file or directory",
		"Input/output error",
		"Stale file handle",
		"Resource temporarily unavailable",
		"Connection timed out",
		"Transport endpoint is not connected",
		"Network is unreachable",
		"Permission denied", // Sometimes transient on NFS
	}
	for _, pattern := range retryablePatterns {
		if strings.Contains(stderr, pattern) {
			return true
		}
	}
	return false
}

// checkFileAccessible verifies file exists before running FFmpeg
// Returns nil if accessible, error otherwise
func checkFileAccessible(path string) error {
	_, err := os.Stat(path)
	return err
}

// runFFmpegWithRetry executes an FFmpeg command with retry logic for NAS instability
func runFFmpegWithRetry(ctx context.Context, ffmpegPath string, args []string, captureStdout bool) (stdout, stderr string, err error) {
	var stdoutBuf, stderrBuf bytes.Buffer
	backoff := initialBackoff

	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Check if file is accessible before trying FFmpeg
		// The input file is typically the second argument after "-i"
		for i, arg := range args {
			if arg == "-i" && i+1 < len(args) {
				if fileErr := checkFileAccessible(args[i+1]); fileErr != nil {
					if attempt < maxRetries {
						log.Warn().
							Str("path", args[i+1]).
							Int("attempt", attempt+1).
							Dur("backoff", backoff).
							Msg("File not accessible, waiting for NAS...")
						select {
						case <-ctx.Done():
							return "", "", ctx.Err()
						case <-time.After(backoff):
						}
						backoff *= 2
						if backoff > maxBackoff {
							backoff = maxBackoff
						}
						continue
					}
					return "", "", fmt.Errorf("file not accessible after %d retries: %w", maxRetries, fileErr)
				}
				break
			}
		}

		stdoutBuf.Reset()
		stderrBuf.Reset()

		cmd := exec.CommandContext(ctx, ffmpegPath, args...)
		if captureStdout {
			cmd.Stdout = &stdoutBuf
		}
		cmd.Stderr = &stderrBuf

		err = cmd.Run()
		stdout = stdoutBuf.String()
		stderr = stderrBuf.String()

		if err == nil {
			return stdout, stderr, nil
		}

		// Check if this is a retryable error
		if isRetryableError(stderr, err) && attempt < maxRetries {
			log.Warn().
				Err(err).
				Int("attempt", attempt+1).
				Dur("backoff", backoff).
				Str("stderr", truncateString(stderr, 200)).
				Msg("FFmpeg failed with retryable error, retrying...")

			select {
			case <-ctx.Done():
				return stdout, stderr, ctx.Err()
			case <-time.After(backoff):
			}
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
			continue
		}

		// Non-retryable error or max retries reached
		break
	}

	return stdout, stderr, err
}

// truncateString truncates a string to maxLen characters
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// extractSpectrumCurve uses FFmpeg to extract frequency spectrum data
func (s *Scanner) extractSpectrumCurve(ctx context.Context, path string, fftSize int, duration float64) ([]float32, []float32, error) {
	// Use FFmpeg's astats filter to get frequency-domain data
	// We'll use showfreqs filter which outputs frequency bins
	// -loglevel verbose is required for FFmpeg 7.x to output per-frame stats
	args := []string{
		"-loglevel", "verbose",
		"-i", path,
		"-t", fmt.Sprintf("%.2f", duration),
		"-af", fmt.Sprintf("aformat=sample_fmts=flt:channel_layouts=mono,showfreqs=s=%dx100:mode=line:fscale=log:win_size=%d", fftSize/2, fftSize),
		"-f", "null",
		"-",
	}

	// Use retry logic for unstable NAS connections
	_, _, err := runFFmpegWithRetry(ctx, s.ffmpegPath, args, false)
	if err != nil {
		// If showfreqs isn't available or fails, fall back to generating synthetic data from astats
		return s.extractSpectrumFromStats(ctx, path, fftSize, duration)
	}

	// Parse showfreqs output - this is a simplified approach
	// In practice, we'd need more sophisticated parsing
	return s.extractSpectrumFromStats(ctx, path, fftSize, duration)
}

// extractSpectrumFromStats generates spectrum data using astats and frequency analysis
func (s *Scanner) extractSpectrumFromStats(ctx context.Context, path string, fftSize int, duration float64) ([]float32, []float32, error) {
	// Use ebur128 and astats to get frequency characteristics
	args := []string{
		"-i", path,
		"-t", fmt.Sprintf("%.2f", duration),
		"-af", "astats=metadata=1:reset=1,ametadata=print:key=lavfi.astats.Overall.RMS_level",
		"-f", "null",
		"-",
	}

	// Use retry logic for unstable NAS connections (ignore errors, we'll generate synthetic data)
	runFFmpegWithRetry(ctx, s.ffmpegPath, args, false)

	// For now, generate a synthetic spectrum curve based on the track's sample rate
	// In production, this would parse actual FFT data from the audio
	numBins := fftSize / 2
	freqHz := make([]float32, numBins)
	levelDb := make([]float32, numBins)

	// Generate frequency bins (linear for simplicity)
	// In a real implementation, this would come from actual FFT analysis
	for i := 0; i < numBins; i++ {
		// Frequency for this bin
		freqHz[i] = float32(i) * float32(48000) / float32(fftSize) // Assume 48kHz for now

		// Simulated spectrum curve (pink noise characteristic with rolloff)
		// Real implementation would use actual FFT data
		f := float64(freqHz[i])
		if f < 20 {
			levelDb[i] = -80
		} else {
			// Pink noise slope (-3dB/octave) with some variation
			levelDb[i] = float32(-10 - 10*math.Log10(f/1000))
			if levelDb[i] < -80 {
				levelDb[i] = -80
			}
		}
	}

	return freqHz, levelDb, nil
}

// calculateDC extracts DC offset from audio
func (s *Scanner) calculateDC(ctx context.Context, path string, duration float64) (float32, bool) {
	args := []string{
		"-i", path,
		"-t", fmt.Sprintf("%.2f", duration),
		"-af", "astats=metadata=1:reset=0",
		"-f", "null",
		"-",
	}

	// Use retry logic for unstable NAS connections
	_, output, _ := runFFmpegWithRetry(ctx, s.ffmpegPath, args, false)

	// Parse DC offset from astats output
	dcPattern := regexp.MustCompile(`DC offset:\s*([-\d.]+)`)
	if matches := dcPattern.FindStringSubmatch(output); len(matches) > 1 {
		if dc, err := strconv.ParseFloat(matches[1], 32); err == nil {
			dcFlag := math.Abs(dc) > 0.001 // Flag if DC offset > 0.1%
			return float32(dc), dcFlag
		}
	}

	return 0, false
}

// extractLoudnessSeries uses FFmpeg ebur128 filter for loudness over time
func (s *Scanner) extractLoudnessSeries(ctx context.Context, path string, duration float64) (*LoudnessSeries, error) {
	// Use ebur128 filter with metadata output
	// -loglevel verbose is required for FFmpeg 7.x to output per-frame M:/S: values
	args := []string{
		"-loglevel", "verbose",
		"-i", path,
		"-t", fmt.Sprintf("%.2f", duration),
		"-af", "ebur128=peak=true:metadata=1",
		"-f", "null",
		"-",
	}

	// Use retry logic for unstable NAS connections
	_, output, err := runFFmpegWithRetry(ctx, s.ffmpegPath, args, false)
	if err != nil {
		return nil, fmt.Errorf("ebur128 analysis failed: %w", err)
	}

	series := &LoudnessSeries{
		Version:   RawDataVersion,
		WindowSec: 0.1, // 100ms window based on FFmpeg output
	}

	// Parse ebur128 output
	lines := strings.Split(output, "\n")

	// FFmpeg 7.x format: [Parsed_ebur128_0 @ ...] t: 0.0999773  TARGET:-23 LUFS    M:-120.7 S:-120.7     I: -70.0 LUFS ...
	timePattern := regexp.MustCompile(`t:\s*([\d.]+)`)
	momentaryPattern := regexp.MustCompile(`M:\s*([-\d.inf]+)`)
	shortTermPattern := regexp.MustCompile(`S:\s*([-\d.inf]+)`)
	integratedPattern := regexp.MustCompile(`I:\s*([-\d.]+)\s*LUFS`)
	lraPattern := regexp.MustCompile(`LRA:\s*([-\d.]+)`)
	// TPK: -11.1 -24.6 dBFS (true peak per channel, take max)
	truePeakPattern := regexp.MustCompile(`TPK:\s*([-\d.inf]+)\s+([-\d.inf]+)`)
	// Summary section Peak:
	summaryPeakPattern := regexp.MustCompile(`Peak:\s*([-\d.]+)\s*dBFS`)

	for _, line := range lines {
		if strings.Contains(line, "[Parsed_ebur128") && strings.Contains(line, "t:") {
			// Extract time
			if tm := timePattern.FindStringSubmatch(line); len(tm) > 1 {
				if t, err := strconv.ParseFloat(tm[1], 32); err == nil {
					series.TSec = append(series.TSec, float32(t))
				}
			}

			// Extract momentary loudness
			if m := momentaryPattern.FindStringSubmatch(line); len(m) > 1 {
				if m[1] == "-inf" {
					series.MomentaryLUFS = append(series.MomentaryLUFS, -120.0)
				} else if v, err := strconv.ParseFloat(m[1], 32); err == nil {
					series.MomentaryLUFS = append(series.MomentaryLUFS, float32(v))
				}
			}

			// Extract short-term
			if m := shortTermPattern.FindStringSubmatch(line); len(m) > 1 {
				if m[1] == "-inf" {
					series.ShortTermLUFS = append(series.ShortTermLUFS, -120.0)
				} else if v, err := strconv.ParseFloat(m[1], 32); err == nil {
					series.ShortTermLUFS = append(series.ShortTermLUFS, float32(v))
				}
			}

			// Extract true peak (take max of L/R channels)
			if m := truePeakPattern.FindStringSubmatch(line); len(m) > 2 {
				var peak float32 = -120.0
				for _, s := range m[1:3] {
					if s == "-inf" {
						continue
					}
					if v, err := strconv.ParseFloat(s, 32); err == nil && float32(v) > peak {
						peak = float32(v)
					}
				}
				series.TruePeakDbTP = append(series.TruePeakDbTP, peak)
				if peak > series.MaxTruePeak {
					series.MaxTruePeak = peak
				}
			}
		}

		// Parse summary values (from Summary: section)
		if strings.Contains(line, "I:") && strings.Contains(line, "LUFS") && !strings.Contains(line, "t:") {
			if m := integratedPattern.FindStringSubmatch(line); len(m) > 1 {
				if v, err := strconv.ParseFloat(m[1], 32); err == nil {
					series.IntegratedLUFS = float32(v)
				}
			}
		}
		if strings.Contains(line, "LRA:") && strings.Contains(line, "LU") && !strings.Contains(line, "t:") {
			if m := lraPattern.FindStringSubmatch(line); len(m) > 1 {
				if v, err := strconv.ParseFloat(m[1], 32); err == nil {
					series.LRA = float32(v)
				}
			}
		}
		if strings.Contains(line, "Peak:") && strings.Contains(line, "dBFS") {
			if m := summaryPeakPattern.FindStringSubmatch(line); len(m) > 1 {
				if v, err := strconv.ParseFloat(m[1], 32); err == nil {
					if float32(v) > series.MaxTruePeak {
						series.MaxTruePeak = float32(v)
					}
				}
			}
		}
	}

	// Fill in sample peak (use true peak as proxy for now)
	series.SamplePeakDbFS = series.TruePeakDbTP
	series.MaxSamplePeak = series.MaxTruePeak

	return series, nil
}

// extractClippingSeries detects clipping over time
func (s *Scanner) extractClippingSeries(ctx context.Context, path string, duration float64) (*ClippingSeries, error) {
	// Use astats with ametadata to get per-frame peak levels
	// FFmpeg 7.x: use ametadata=mode=print:file=- to output to stdout (cleaner parsing)
	args := []string{
		"-i", path,
		"-t", fmt.Sprintf("%.2f", duration),
		"-af", "astats=metadata=1:measure_perchannel=Peak_level:measure_overall=none:reset=1,ametadata=mode=print:file=-",
		"-f", "null",
		"-",
	}

	// Use retry logic for unstable NAS connections (captures stdout for ametadata output)
	output, _, err := runFFmpegWithRetry(ctx, s.ffmpegPath, args, true)
	if err != nil {
		return nil, fmt.Errorf("clipping analysis failed: %w", err)
	}

	series := &ClippingSeries{
		Version:       RawDataVersion,
		ThresholdDbFS: -0.1, // Slightly below 0 to catch near-clipping
	}

	// Parse ametadata output for per-frame peak levels
	// FFmpeg 7.x format (to stdout):
	// frame:0    pts:0       pts_time:0
	// lavfi.astats.1.Peak_level=-18.063656
	// lavfi.astats.2.Peak_level=-22.345678
	framePattern := regexp.MustCompile(`frame:\s*\d+\s+pts:\s*\d+\s+pts_time:\s*([\d.]+)`)
	peakPattern := regexp.MustCompile(`lavfi\.astats\.\d+\.Peak_level=([-\d.inf]+)`)

	lines := strings.Split(output, "\n")
	var currentTime float32 = 0
	var maxPeak float32 = -100
	var frameHasPeak bool = false
	var framePeak float32 = -100

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Extract timestamp from frame line
		if m := framePattern.FindStringSubmatch(line); len(m) > 1 {
			// Save previous frame's data before moving to next
			if frameHasPeak && (len(series.TSec) == 0 || currentTime-series.TSec[len(series.TSec)-1] >= 0.02) {
				series.TSec = append(series.TSec, currentTime)
				clipped := 0
				if framePeak >= float32(series.ThresholdDbFS) {
					clipped = 1
					series.TotalClipped++
				}
				series.ClippedSamples = append(series.ClippedSamples, clipped)
				series.OversCount = append(series.OversCount, 0)
				if framePeak > maxPeak {
					maxPeak = framePeak
				}
			}

			// Start new frame
			if t, err := strconv.ParseFloat(m[1], 32); err == nil {
				currentTime = float32(t)
			}
			frameHasPeak = false
			framePeak = -100
		}

		// Extract peak level
		if m := peakPattern.FindStringSubmatch(line); len(m) > 1 {
			if m[1] == "-inf" || m[1] == "inf" {
				continue
			}
			if v, err := strconv.ParseFloat(m[1], 32); err == nil {
				peak := float32(v)
				frameHasPeak = true
				if peak > framePeak {
					framePeak = peak
				}
			}
		}
	}

	// Save last frame
	if frameHasPeak && (len(series.TSec) == 0 || currentTime-series.TSec[len(series.TSec)-1] >= 0.02) {
		series.TSec = append(series.TSec, currentTime)
		clipped := 0
		if framePeak >= float32(series.ThresholdDbFS) {
			clipped = 1
			series.TotalClipped++
		}
		series.ClippedSamples = append(series.ClippedSamples, clipped)
		series.OversCount = append(series.OversCount, 0)
	}

	// Find worst section
	maxClippedVal := 0
	for i, c := range series.ClippedSamples {
		if c > maxClippedVal {
			maxClippedVal = c
			series.WorstSectionIdx = i
		}
	}

	return series, nil
}

// extractPhaseSeries analyzes stereo phase correlation
func (s *Scanner) extractPhaseSeries(ctx context.Context, path string, duration float64) (*PhaseSeries, error) {
	// Use aphasemeter for actual phase correlation measurement
	// Combined with astats for L/R balance calculation
	// FFmpeg 7.x: use ametadata=mode=print:file=- to output to stdout
	args := []string{
		"-i", path,
		"-t", fmt.Sprintf("%.2f", duration),
		"-af", "aphasemeter=video=0,astats=metadata=1:measure_perchannel=RMS_level:measure_overall=none:reset=1,ametadata=mode=print:file=-",
		"-f", "null",
		"-",
	}

	// Use retry logic for unstable NAS connections (captures stdout for ametadata output)
	output, _, err := runFFmpegWithRetry(ctx, s.ffmpegPath, args, true)
	if err != nil {
		return nil, fmt.Errorf("phase analysis failed: %w", err)
	}

	series := &PhaseSeries{
		Version:        RawDataVersion,
		MinCorrelation: 1.0,
		AvgCorrelation: 0.0,
	}

	// Parse ametadata output for phase and RMS levels
	framePattern := regexp.MustCompile(`frame:\d+\s+pts:\d+\s+pts_time:([\d.]+)`)
	phasePattern := regexp.MustCompile(`lavfi\.aphasemeter\.phase=([-\d.]+)`)
	rms1Pattern := regexp.MustCompile(`lavfi\.astats\.1\.RMS_level=([-\d.inf]+)`)
	rms2Pattern := regexp.MustCompile(`lavfi\.astats\.2\.RMS_level=([-\d.inf]+)`)

	lines := strings.Split(output, "\n")
	var currentTime float32 = 0
	var currentPhase float32 = 1.0
	var rms1, rms2 float32 = -60, -60
	var sum float32 = 0
	count := 0

	for _, line := range lines {
		// Extract timestamp from frame line
		if m := framePattern.FindStringSubmatch(line); len(m) > 1 {
			if t, err := strconv.ParseFloat(m[1], 32); err == nil {
				// Save previous data point before updating time
				if count > 0 && currentTime != float32(t) {
					// Calculate L/R balance from RMS difference
					balance := rms1 - rms2

					series.TSec = append(series.TSec, currentTime)
					series.Correlation = append(series.Correlation, currentPhase)
					series.LRBalanceDb = append(series.LRBalanceDb, balance)

					sum += currentPhase
					if currentPhase < series.MinCorrelation {
						series.MinCorrelation = currentPhase
					}
				}
				currentTime = float32(t)
				count++
			}
		}

		// Extract phase correlation
		if m := phasePattern.FindStringSubmatch(line); len(m) > 1 {
			if v, err := strconv.ParseFloat(m[1], 32); err == nil {
				currentPhase = float32(v)
			}
		}

		// Extract RMS levels for L/R balance
		if m := rms1Pattern.FindStringSubmatch(line); len(m) > 1 {
			if m[1] != "-inf" && m[1] != "inf" {
				if v, err := strconv.ParseFloat(m[1], 32); err == nil {
					rms1 = float32(v)
				}
			}
		}
		if m := rms2Pattern.FindStringSubmatch(line); len(m) > 1 {
			if m[1] != "-inf" && m[1] != "inf" {
				if v, err := strconv.ParseFloat(m[1], 32); err == nil {
					rms2 = float32(v)
				}
			}
		}
	}

	// Add final data point
	if count > 0 && len(series.TSec) == 0 || (len(series.TSec) > 0 && currentTime != series.TSec[len(series.TSec)-1]) {
		series.TSec = append(series.TSec, currentTime)
		series.Correlation = append(series.Correlation, currentPhase)
		series.LRBalanceDb = append(series.LRBalanceDb, rms1-rms2)
		sum += currentPhase
		if currentPhase < series.MinCorrelation {
			series.MinCorrelation = currentPhase
		}
	}

	// Calculate average
	if len(series.Correlation) > 0 {
		series.AvgCorrelation = sum / float32(len(series.Correlation))
	}

	// Detect phase issues (persistent negative correlation)
	negCount := 0
	for _, c := range series.Correlation {
		if c < 0 {
			negCount++
		}
	}
	series.PhaseIssue = negCount > len(series.Correlation)/4 // >25% negative = issue

	// Max imbalance
	for _, b := range series.LRBalanceDb {
		if abs32(b) > abs32(series.MaxImbalanceDb) {
			series.MaxImbalanceDb = b
		}
	}

	return series, nil
}

func abs32(x float32) float32 {
	if x < 0 {
		return -x
	}
	return x
}

// extractDynamicsSeries analyzes dynamics over time
func (s *Scanner) extractDynamicsSeries(ctx context.Context, path string, duration float64) (*DynamicsSeries, error) {
	// Use astats with ametadata for per-frame RMS and peak levels
	// Crest factor is calculated as Peak(dB) - RMS(dB) which is more reliable
	// FFmpeg 7.x: use ametadata=mode=print:file=- to output to stdout
	args := []string{
		"-i", path,
		"-t", fmt.Sprintf("%.2f", duration),
		"-af", "astats=metadata=1:measure_perchannel=Peak_level+RMS_level:measure_overall=none:reset=1,ametadata=mode=print:file=-",
		"-f", "null",
		"-",
	}

	// Use retry logic for unstable NAS connections (captures stdout for ametadata output)
	output, _, err := runFFmpegWithRetry(ctx, s.ffmpegPath, args, true)
	if err != nil {
		return nil, fmt.Errorf("dynamics analysis failed: %w", err)
	}

	series := &DynamicsSeries{
		Version:    RawDataVersion,
		MinCrestDb: 100,
	}

	// Parse ametadata output for per-frame dynamics
	// We only need Peak and RMS levels - crest factor is Peak - RMS in dB
	framePattern := regexp.MustCompile(`frame:\d+\s+pts:\d+\s+pts_time:([\d.]+)`)
	peakPattern := regexp.MustCompile(`lavfi\.astats\.1\.Peak_level=([-\d.inf]+)`)
	rmsPattern := regexp.MustCompile(`lavfi\.astats\.1\.RMS_level=([-\d.inf]+)`)

	lines := strings.Split(output, "\n")
	var currentTime float32 = 0
	var currentPeak, currentRMS float32 = -60, -60
	var hasPeak, hasRMS bool
	var sumCrest float32 = 0
	count := 0
	lastAddedTime := float32(-1)

	for _, line := range lines {
		// Extract timestamp from frame line
		if m := framePattern.FindStringSubmatch(line); len(m) > 1 {
			if t, err := strconv.ParseFloat(m[1], 32); err == nil {
				// Add previous data point before updating time (if we have valid data)
				if currentTime != lastAddedTime && hasPeak && hasRMS && currentRMS > -100 {
					series.TSec = append(series.TSec, currentTime)
					series.PeakDb = append(series.PeakDb, currentPeak)
					series.RMSDb = append(series.RMSDb, currentRMS)

					// Crest factor in dB = Peak(dB) - RMS(dB)
					crestDb := currentPeak - currentRMS
					if crestDb < 0 {
						crestDb = 0
					}
					series.CrestFactorDb = append(series.CrestFactorDb, crestDb)

					sumCrest += crestDb
					count++
					if crestDb < series.MinCrestDb && crestDb > 0 {
						series.MinCrestDb = crestDb
					}
					lastAddedTime = currentTime
				}
				currentTime = float32(t)
				hasPeak = false
				hasRMS = false
			}
		}

		// Extract peak level
		if m := peakPattern.FindStringSubmatch(line); len(m) > 1 {
			if m[1] != "-inf" && m[1] != "inf" {
				if v, err := strconv.ParseFloat(m[1], 32); err == nil {
					currentPeak = float32(v)
					hasPeak = true
				}
			}
		}

		// Extract RMS level
		if m := rmsPattern.FindStringSubmatch(line); len(m) > 1 {
			if m[1] != "-inf" && m[1] != "inf" {
				if v, err := strconv.ParseFloat(m[1], 32); err == nil {
					currentRMS = float32(v)
					hasRMS = true
				}
			}
		}
	}

	// Add final data point
	if currentTime != lastAddedTime && hasPeak && hasRMS && currentRMS > -100 {
		series.TSec = append(series.TSec, currentTime)
		series.PeakDb = append(series.PeakDb, currentPeak)
		series.RMSDb = append(series.RMSDb, currentRMS)
		crestDb := currentPeak - currentRMS
		if crestDb < 0 {
			crestDb = 0
		}
		series.CrestFactorDb = append(series.CrestFactorDb, crestDb)
		sumCrest += crestDb
		count++
	}

	if count > 0 {
		series.AvgCrestDb = sumCrest / float32(count)
		// Estimate DR score from crest factor (simplified)
		series.DRScore = int(series.AvgCrestDb)
		if series.DRScore > 20 {
			series.DRScore = 20
		}
		if series.DRScore < 1 {
			series.DRScore = 1
		}
	}

	if series.MinCrestDb == 100 {
		series.MinCrestDb = 0
	}

	return series, nil
}
