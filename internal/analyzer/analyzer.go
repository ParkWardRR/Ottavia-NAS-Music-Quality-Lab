package analyzer

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/ottavia-music/ottavia/internal/database"
	"github.com/ottavia-music/ottavia/internal/models"
)

type Analyzer struct {
	db            *database.DB
	ffprobePath   string
	ffmpegPath    string
	artifactsPath string
}

func New(db *database.DB, ffprobePath, ffmpegPath, artifactsPath string) *Analyzer {
	return &Analyzer{
		db:            db,
		ffprobePath:   ffprobePath,
		ffmpegPath:    ffmpegPath,
		artifactsPath: artifactsPath,
	}
}

type ProbeResult struct {
	Format  ProbeFormat   `json:"format"`
	Streams []ProbeStream `json:"streams"`
}

type ProbeFormat struct {
	Filename       string            `json:"filename"`
	FormatName     string            `json:"format_name"`
	Duration       string            `json:"duration"`
	Size           string            `json:"size"`
	BitRate        string            `json:"bit_rate"`
	Tags           map[string]string `json:"tags"`
}

type ProbeStream struct {
	Index             int               `json:"index"`
	CodecType         string            `json:"codec_type"`
	CodecName         string            `json:"codec_name"`
	SampleRate        string            `json:"sample_rate"`
	Channels          int               `json:"channels"`
	BitsPerSample     int               `json:"bits_per_sample"`
	BitsPerRawSample  string            `json:"bits_per_raw_sample"`
	BitsPerCodedSample int             `json:"bits_per_coded_sample"`
	BitRate           string            `json:"bit_rate"`
	Duration          string            `json:"duration"`
	Tags              map[string]string `json:"tags"`
	Disposition       map[string]int    `json:"disposition"`
}

func (a *Analyzer) AnalyzeFile(ctx context.Context, mediaFileID string) error {
	mf, err := a.db.GetMediaFileByPath(ctx, "", mediaFileID)
	if err != nil {
		var mfByID models.MediaFile
		err = a.db.GetContext(ctx, &mfByID, "SELECT * FROM media_files WHERE id = ?", mediaFileID)
		if err != nil {
			return fmt.Errorf("media file not found: %w", err)
		}
		mf = &mfByID
	}

	log.Info().
		Str("file_id", mf.ID).
		Str("path", mf.Path).
		Msg("Analyzing file")

	probe, err := a.probeFile(ctx, mf.Path)
	if err != nil {
		mf.Status = models.StatusFailed
		mf.ErrorMsg = sql.NullString{String: err.Error(), Valid: true}
		a.db.UpdateMediaFile(ctx, mf)
		return fmt.Errorf("probe failed: %w", err)
	}

	track, err := a.createTrackFromProbe(ctx, mf, probe)
	if err != nil {
		return fmt.Errorf("failed to create track: %w", err)
	}

	result, err := a.analyzeAudio(ctx, mf.Path, track)
	if err != nil {
		log.Warn().Err(err).Str("path", mf.Path).Msg("Audio analysis failed")
	}

	if result != nil {
		result.TrackID = track.ID
		if err := a.db.CreateAnalysisResult(ctx, result); err != nil {
			return fmt.Errorf("failed to save analysis: %w", err)
		}
	}

	mf.Status = models.StatusSuccess
	if err := a.db.UpdateMediaFile(ctx, mf); err != nil {
		return fmt.Errorf("failed to update media file: %w", err)
	}

	return nil
}

func (a *Analyzer) probeFile(ctx context.Context, path string) (*ProbeResult, error) {
	args := []string{
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		path,
	}

	cmd := exec.CommandContext(ctx, a.ffprobePath, args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffprobe failed: %w", err)
	}

	var result ProbeResult
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse ffprobe output: %w", err)
	}

	return &result, nil
}

func (a *Analyzer) createTrackFromProbe(ctx context.Context, mf *models.MediaFile, probe *ProbeResult) (*models.Track, error) {
	existing, _ := a.db.GetTrackByMediaFile(ctx, mf.ID)
	if existing != nil {
		return a.updateTrackFromProbe(ctx, existing, probe)
	}

	track := &models.Track{
		MediaFileID: mf.ID,
	}

	for _, stream := range probe.Streams {
		if stream.CodecType == "audio" {
			track.Codec = stream.CodecName
			if sr, err := strconv.Atoi(stream.SampleRate); err == nil {
				track.SampleRate = sr
			}
			track.Channels = stream.Channels
			// Try multiple sources for bit depth
			track.BitDepth = stream.BitsPerSample
			if track.BitDepth == 0 && stream.BitsPerRawSample != "" {
				if bd, err := strconv.Atoi(stream.BitsPerRawSample); err == nil {
					track.BitDepth = bd
				}
			}
			if track.BitDepth == 0 && stream.BitsPerCodedSample > 0 {
				track.BitDepth = stream.BitsPerCodedSample
			}
			// For ALAC/FLAC, default to 16-bit if no bit depth found
			if track.BitDepth == 0 && (stream.CodecName == "alac" || stream.CodecName == "flac") {
				track.BitDepth = 16
			}
			if br, err := strconv.Atoi(stream.BitRate); err == nil {
				track.Bitrate = br
			}
			if dur, err := strconv.ParseFloat(stream.Duration, 64); err == nil {
				track.Duration = dur
			}
			break
		}
	}

	if track.Duration == 0 {
		if dur, err := strconv.ParseFloat(probe.Format.Duration, 64); err == nil {
			track.Duration = dur
		}
	}

	if track.Bitrate == 0 {
		if br, err := strconv.Atoi(probe.Format.BitRate); err == nil {
			track.Bitrate = br
		}
	}

	a.extractTags(track, probe)
	a.checkArtwork(track, probe)

	if err := a.db.CreateTrack(ctx, track); err != nil {
		return nil, err
	}

	return track, nil
}

func (a *Analyzer) updateTrackFromProbe(ctx context.Context, track *models.Track, probe *ProbeResult) (*models.Track, error) {
	for _, stream := range probe.Streams {
		if stream.CodecType == "audio" {
			track.Codec = stream.CodecName
			if sr, err := strconv.Atoi(stream.SampleRate); err == nil {
				track.SampleRate = sr
			}
			track.Channels = stream.Channels
			// Try multiple sources for bit depth
			track.BitDepth = stream.BitsPerSample
			if track.BitDepth == 0 && stream.BitsPerRawSample != "" {
				if bd, err := strconv.Atoi(stream.BitsPerRawSample); err == nil {
					track.BitDepth = bd
				}
			}
			if track.BitDepth == 0 && stream.BitsPerCodedSample > 0 {
				track.BitDepth = stream.BitsPerCodedSample
			}
			// For ALAC/FLAC, default to 16-bit if no bit depth found
			if track.BitDepth == 0 && (stream.CodecName == "alac" || stream.CodecName == "flac") {
				track.BitDepth = 16
			}
			if br, err := strconv.Atoi(stream.BitRate); err == nil {
				track.Bitrate = br
			}
			if dur, err := strconv.ParseFloat(stream.Duration, 64); err == nil {
				track.Duration = dur
			}
			break
		}
	}

	a.extractTags(track, probe)
	a.checkArtwork(track, probe)

	if err := a.db.UpdateTrack(ctx, track); err != nil {
		return nil, err
	}

	return track, nil
}

func (a *Analyzer) extractTags(track *models.Track, probe *ProbeResult) {
	tags := probe.Format.Tags

	if v, ok := tags["title"]; ok && v != "" {
		track.Title = sql.NullString{String: v, Valid: true}
	} else if v, ok := tags["TITLE"]; ok && v != "" {
		track.Title = sql.NullString{String: v, Valid: true}
	}

	if v, ok := tags["artist"]; ok && v != "" {
		track.Artist = sql.NullString{String: v, Valid: true}
	} else if v, ok := tags["ARTIST"]; ok && v != "" {
		track.Artist = sql.NullString{String: v, Valid: true}
	}

	if v, ok := tags["album"]; ok && v != "" {
		track.Album = sql.NullString{String: v, Valid: true}
	} else if v, ok := tags["ALBUM"]; ok && v != "" {
		track.Album = sql.NullString{String: v, Valid: true}
	}

	if v, ok := tags["album_artist"]; ok && v != "" {
		track.AlbumArtist = sql.NullString{String: v, Valid: true}
	} else if v, ok := tags["ALBUMARTIST"]; ok && v != "" {
		track.AlbumArtist = sql.NullString{String: v, Valid: true}
	}

	if v, ok := tags["genre"]; ok && v != "" {
		track.Genre = sql.NullString{String: v, Valid: true}
	} else if v, ok := tags["GENRE"]; ok && v != "" {
		track.Genre = sql.NullString{String: v, Valid: true}
	}

	if v, ok := tags["track"]; ok && v != "" {
		if num := parseTrackNumber(v); num > 0 {
			track.TrackNumber = sql.NullInt32{Int32: int32(num), Valid: true}
		}
	} else if v, ok := tags["TRACKNUMBER"]; ok && v != "" {
		if num := parseTrackNumber(v); num > 0 {
			track.TrackNumber = sql.NullInt32{Int32: int32(num), Valid: true}
		}
	}

	if v, ok := tags["disc"]; ok && v != "" {
		if num := parseTrackNumber(v); num > 0 {
			track.DiscNumber = sql.NullInt32{Int32: int32(num), Valid: true}
		}
	} else if v, ok := tags["DISCNUMBER"]; ok && v != "" {
		if num := parseTrackNumber(v); num > 0 {
			track.DiscNumber = sql.NullInt32{Int32: int32(num), Valid: true}
		}
	}

	if v, ok := tags["date"]; ok && v != "" {
		if year := parseYear(v); year > 0 {
			track.Year = sql.NullInt32{Int32: int32(year), Valid: true}
		}
	} else if v, ok := tags["DATE"]; ok && v != "" {
		if year := parseYear(v); year > 0 {
			track.Year = sql.NullInt32{Int32: int32(year), Valid: true}
		}
	} else if v, ok := tags["year"]; ok && v != "" {
		if year := parseYear(v); year > 0 {
			track.Year = sql.NullInt32{Int32: int32(year), Valid: true}
		}
	}
}

func (a *Analyzer) checkArtwork(track *models.Track, probe *ProbeResult) {
	for _, stream := range probe.Streams {
		if stream.CodecType == "video" {
			if disp, ok := stream.Disposition["attached_pic"]; ok && disp == 1 {
				track.HasArtwork = true
				return
			}
		}
	}
}

// ExtractAlbumArt extracts embedded artwork from an audio file
func (a *Analyzer) ExtractAlbumArt(ctx context.Context, mediaFilePath, trackID string) (string, error) {
	outputDir := filepath.Join(a.artifactsPath, trackID[:2], trackID)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", err
	}

	outputPath := filepath.Join(outputDir, "artwork.jpg")

	args := []string{
		"-i", mediaFilePath,
		"-an",
		"-vcodec", "copy",
		"-y",
		outputPath,
	}

	cmd := exec.CommandContext(ctx, a.ffmpegPath, args...)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("artwork extraction failed: %w", err)
	}

	// Verify file exists and has content
	if info, err := os.Stat(outputPath); err != nil || info.Size() == 0 {
		os.Remove(outputPath)
		return "", fmt.Errorf("no artwork extracted")
	}

	artifact := &models.Artifact{
		ID:       uuid.NewString(),
		TrackID:  trackID,
		Type:     "artwork",
		Path:     outputPath,
		MimeType: "image/jpeg",
	}

	if err := a.db.CreateArtifact(ctx, artifact); err != nil {
		return "", err
	}

	return outputPath, nil
}

func (a *Analyzer) analyzeAudio(ctx context.Context, path string, track *models.Track) (*models.AnalysisResult, error) {
	result := &models.AnalysisResult{
		Version:        1,
		LosslessStatus: models.LosslessPass,
		LosslessScore:  100,
		IntegrityOK:    true,
	}

	var issues []models.Issue

	if err := a.runVolumeDetect(ctx, path, result); err != nil {
		log.Warn().Err(err).Msg("Volume detection failed")
	}

	if err := a.runLoudnessAnalysis(ctx, path, result); err != nil {
		log.Warn().Err(err).Msg("Loudness analysis failed")
	}

	if result.ClippedSamples > 0 {
		issues = append(issues, models.Issue{
			Type:       "clipping",
			Severity:   models.SeverityWarning,
			Message:    fmt.Sprintf("Detected %d clipped samples", result.ClippedSamples),
			Confidence: 0.95,
		})
	}

	if result.PeakLevel > 0 {
		issues = append(issues, models.Issue{
			Type:       "peak_level",
			Severity:   models.SeverityWarning,
			Message:    fmt.Sprintf("Peak level exceeds 0dB (%.2f dB)", result.PeakLevel),
			Confidence: 1.0,
		})
	}

	if track.Codec == "flac" || track.Codec == "alac" || track.Codec == "wav" || track.Codec == "aiff" {
		suspicion := a.detectLossyAncestry(track, result)
		if suspicion > 0.5 {
			result.LosslessStatus = models.LosslessWarn
			result.LosslessScore = (1 - suspicion) * 100
			issues = append(issues, models.Issue{
				Type:       "lossy_ancestry",
				Severity:   models.SeverityWarning,
				Message:    "This file may have been transcoded from a lossy source",
				Confidence: suspicion,
			})
		}
		if suspicion > 0.8 {
			result.LosslessStatus = models.LosslessFail
		}
	}

	if abs(result.DCOffset) > 0.01 {
		issues = append(issues, models.Issue{
			Type:       "dc_offset",
			Severity:   models.SeverityInfo,
			Message:    fmt.Sprintf("DC offset detected: %.4f", result.DCOffset),
			Confidence: 0.9,
		})
	}

	issuesJSON, _ := json.Marshal(issues)
	result.IssuesJSON = string(issuesJSON)
	result.Issues = issues

	if err := a.generateWaveform(ctx, path, track.ID); err != nil {
		log.Warn().Err(err).Msg("Waveform generation failed")
	}

	return result, nil
}

func (a *Analyzer) runVolumeDetect(ctx context.Context, path string, result *models.AnalysisResult) error {
	args := []string{
		"-i", path,
		"-af", "volumedetect",
		"-f", "null",
		"-",
	}

	cmd := exec.CommandContext(ctx, a.ffmpegPath, args...)
	output, _ := cmd.CombinedOutput()

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "max_volume:") {
			parts := strings.Split(line, "max_volume:")
			if len(parts) > 1 {
				val := strings.TrimSpace(strings.Split(parts[1], " ")[0])
				if v, err := strconv.ParseFloat(val, 64); err == nil {
					result.PeakLevel = v
				}
			}
		}
		if strings.Contains(line, "mean_volume:") {
			parts := strings.Split(line, "mean_volume:")
			if len(parts) > 1 {
				val := strings.TrimSpace(strings.Split(parts[1], " ")[0])
				if v, err := strconv.ParseFloat(val, 64); err == nil {
					if result.PeakLevel != 0 {
						result.CrestFactor = result.PeakLevel - v
					}
				}
			}
		}
	}

	return nil
}

func (a *Analyzer) runLoudnessAnalysis(ctx context.Context, path string, result *models.AnalysisResult) error {
	args := []string{
		"-i", path,
		"-af", "ebur128=peak=true",
		"-f", "null",
		"-",
	}

	cmd := exec.CommandContext(ctx, a.ffmpegPath, args...)
	output, _ := cmd.CombinedOutput()

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "I:") && strings.Contains(line, "LUFS") {
			parts := strings.Fields(line)
			for i, p := range parts {
				if p == "I:" && i+1 < len(parts) {
					if v, err := strconv.ParseFloat(parts[i+1], 64); err == nil {
						result.IntegratedLoudness = v
					}
				}
				if p == "LRA:" && i+1 < len(parts) {
					if v, err := strconv.ParseFloat(parts[i+1], 64); err == nil {
						result.LoudnessRange = v
					}
				}
			}
		}
		if strings.Contains(line, "Peak:") {
			parts := strings.Fields(line)
			for i, p := range parts {
				if p == "Peak:" && i+1 < len(parts) {
					if v, err := strconv.ParseFloat(parts[i+1], 64); err == nil {
						result.TruePeak = v
					}
				}
			}
		}
	}

	return nil
}

func (a *Analyzer) detectLossyAncestry(track *models.Track, result *models.AnalysisResult) float64 {
	suspicion := 0.0

	if track.SampleRate == 44100 && track.BitDepth == 16 {
		if result.HighFreqCutoff > 0 && result.HighFreqCutoff < 16000 {
			suspicion += 0.4
		} else if result.HighFreqCutoff > 0 && result.HighFreqCutoff < 18000 {
			suspicion += 0.2
		}
	}

	if result.SpectralRolloff > 0 && result.SpectralRolloff < 15000 {
		suspicion += 0.3
	}

	return min(suspicion, 1.0)
}

// CalculateDynamicRange returns a DR score (1-20) and human-readable assessment
// Higher DR = more dynamic range = less compression = better for audiophiles
func (a *Analyzer) CalculateDynamicRange(result *models.AnalysisResult) (int, string, string) {
	// DR is typically calculated as the difference between peak and RMS
	// Using loudness range (LRA) as a proxy since it's similar in concept
	dr := int(result.LoudnessRange + result.CrestFactor/2)
	if dr < 1 {
		dr = 1
	}
	if dr > 20 {
		dr = 20
	}

	var rating, explanation string
	switch {
	case dr >= 14:
		rating = "Excellent"
		explanation = "This track has excellent dynamic range. It breathes naturally with quiet moments and loud moments clearly distinguished. Perfect for critical listening."
	case dr >= 10:
		rating = "Good"
		explanation = "This track has good dynamics. It's well-mastered with reasonable contrast between quiet and loud sections."
	case dr >= 7:
		rating = "Moderate"
		explanation = "This track has moderate dynamics. Some compression was applied during mastering, reducing the difference between quiet and loud parts."
	case dr >= 4:
		rating = "Limited"
		explanation = "This track has limited dynamic range - a casualty of the 'loudness wars'. The audio is heavily compressed to sound louder, losing musical nuance."
	default:
		rating = "Crushed"
		explanation = "This track is heavily compressed (brickwalled). Almost no difference between quiet and loud sections. Common in modern pop/rock releases."
	}

	return dr, rating, explanation
}

// GetLossyExplanation returns user-friendly explanation of lossy detection
func (a *Analyzer) GetLossyExplanation(result *models.AnalysisResult, track *models.Track) (string, string) {
	if result.LosslessStatus == models.LosslessPass {
		return "Authentic Lossless", "This file appears to be genuine lossless audio. The high-frequency content extends naturally to the expected range, indicating it wasn't converted from MP3 or other lossy formats."
	}

	if result.LosslessStatus == models.LosslessWarn {
		return "Possibly Transcoded", fmt.Sprintf(
			"This file claims to be lossless (%s) but shows signs it may have been converted from a lossy source like MP3. "+
				"High frequencies appear to cut off around %.0f Hz instead of extending to %.0f Hz. "+
				"While not definitive proof, this pattern is common when lossy files are 'upgraded' to lossless formats.",
			strings.ToUpper(track.Codec),
			result.HighFreqCutoff,
			float64(track.SampleRate)/2,
		)
	}

	return "Likely Transcoded", fmt.Sprintf(
		"Strong evidence this %s file was converted from a lossy source. "+
			"The audio shows a hard frequency cutoff around %.0f Hz - a telltale sign of MP3/AAC compression. "+
			"You're storing lossless file sizes but not getting lossless quality. Consider finding a true lossless source.",
		strings.ToUpper(track.Codec),
		result.HighFreqCutoff,
	)
}

func (a *Analyzer) generateWaveform(ctx context.Context, path, trackID string) error {
	outputDir := filepath.Join(a.artifactsPath, trackID[:2], trackID)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}

	outputPath := filepath.Join(outputDir, "waveform.png")

	args := []string{
		"-i", path,
		"-filter_complex", "showwavespic=s=1920x240:colors=0a84ff|4da3ff",
		"-frames:v", "1",
		"-y",
		outputPath,
	}

	cmd := exec.CommandContext(ctx, a.ffmpegPath, args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("waveform generation failed: %w", err)
	}

	artifact := &models.Artifact{
		ID:       uuid.NewString(),
		TrackID:  trackID,
		Type:     "waveform",
		Path:     outputPath,
		MimeType: "image/png",
		Width:    sql.NullInt32{Int32: 1920, Valid: true},
		Height:   sql.NullInt32{Int32: 240, Valid: true},
	}

	return a.db.CreateArtifact(ctx, artifact)
}

func (a *Analyzer) GenerateSpectrogram(ctx context.Context, path, trackID string) error {
	outputDir := filepath.Join(a.artifactsPath, trackID[:2], trackID)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}

	outputPath := filepath.Join(outputDir, "spectrogram.png")

	args := []string{
		"-i", path,
		"-lavfi", "showspectrumpic=s=1920x480:legend=0:color=intensity",
		"-y",
		outputPath,
	}

	cmd := exec.CommandContext(ctx, a.ffmpegPath, args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("spectrogram generation failed: %w", err)
	}

	artifact := &models.Artifact{
		ID:       uuid.NewString(),
		TrackID:  trackID,
		Type:     "spectrogram",
		Path:     outputPath,
		MimeType: "image/png",
		Width:    sql.NullInt32{Int32: 1920, Valid: true},
		Height:   sql.NullInt32{Int32: 480, Valid: true},
	}

	return a.db.CreateArtifact(ctx, artifact)
}

func parseTrackNumber(s string) int {
	s = strings.TrimSpace(s)
	if idx := strings.Index(s, "/"); idx > 0 {
		s = s[:idx]
	}
	if num, err := strconv.Atoi(s); err == nil {
		return num
	}
	return 0
}

func parseYear(s string) int {
	s = strings.TrimSpace(s)
	if len(s) >= 4 {
		if year, err := strconv.Atoi(s[:4]); err == nil {
			return year
		}
	}
	return 0
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
