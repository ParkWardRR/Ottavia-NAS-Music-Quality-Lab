package audioscan

import (
	"bytes"
	"fmt"
	"os"

	"github.com/klauspost/compress/zstd"
	"github.com/vmihailenco/msgpack/v5"
)

// AudioScanCurve represents the spectrum analysis raw data (v1)
type AudioScanCurve struct {
	Version      int     `msgpack:"version"`
	SampleRateHz int     `msgpack:"sampleRateHz"`
	NyquistHz    int     `msgpack:"nyquistHz"`
	Analyzed     struct {
		StartSec    float64 `msgpack:"startSec"`
		DurationSec float64 `msgpack:"durationSec"`
		ChannelMode string  `msgpack:"channelMode"` // "mono" or "stereo-downmix"
		DecodeFormat string `msgpack:"decodeFormat"` // "f32le"
	} `msgpack:"analyzed"`
	FFT struct {
		FFTSize          int     `msgpack:"fftSize"`
		HopSize          int     `msgpack:"hopSize"`
		Window           string  `msgpack:"window"` // "hann"
		Frames           int     `msgpack:"frames"`
		SmoothingOctaves float64 `msgpack:"smoothingOctaves,omitempty"`
	} `msgpack:"fft"`
	Curve struct {
		FreqHz  []float32 `msgpack:"freqHz"`  // ascending frequency bins
		LevelDb []float32 `msgpack:"levelDb"` // corresponding dB levels
	} `msgpack:"curve"`
	Metrics struct {
		BandwidthHz int     `msgpack:"bandwidthHz,omitempty"`
		DCMean      float32 `msgpack:"dcMean"`
		DCFlag      bool    `msgpack:"dcFlag"`
	} `msgpack:"metrics"`
	Guides struct {
		VerticalLinesHz []int `msgpack:"verticalLinesHz"` // computed from probe cache
	} `msgpack:"guides"`
}

// LoudnessSeries represents loudness over time raw data (v1)
type LoudnessSeries struct {
	Version       int       `msgpack:"version"`
	WindowSec     float64   `msgpack:"windowSec"` // e.g., 0.4s momentary
	TSec          []float32 `msgpack:"tSec"`
	MomentaryLUFS []float32 `msgpack:"momentaryLUFS"`
	ShortTermLUFS []float32 `msgpack:"shortTermLUFS"`
	TruePeakDbTP  []float32 `msgpack:"truePeakDbTP"`
	SamplePeakDbFS []float32 `msgpack:"samplePeakDbFS"`
	// Summary scalars
	IntegratedLUFS float32 `msgpack:"integratedLUFS"`
	LRA            float32 `msgpack:"lra"` // Loudness Range
	MaxTruePeak    float32 `msgpack:"maxTruePeak"`
	MaxSamplePeak  float32 `msgpack:"maxSamplePeak"`
}

// ClippingSeries represents clipping detection over time raw data (v1)
type ClippingSeries struct {
	Version         int       `msgpack:"version"`
	TSec            []float32 `msgpack:"tSec"`
	ClippedSamples  []int     `msgpack:"clippedSamples"` // per bucket
	OversCount      []int     `msgpack:"oversCount"`     // true-peak overs
	ThresholdDbFS   float32   `msgpack:"thresholdDbFS"`  // e.g., 0.0
	// Summary scalars
	TotalClipped    int `msgpack:"totalClipped"`
	TotalOvers      int `msgpack:"totalOvers"`
	WorstSectionIdx int `msgpack:"worstSectionIdx"`
}

// PhaseSeries represents phase correlation over time raw data (v1)
type PhaseSeries struct {
	Version       int       `msgpack:"version"`
	TSec          []float32 `msgpack:"tSec"`
	Correlation   []float32 `msgpack:"correlation"` // -1 to +1
	LRBalanceDb   []float32 `msgpack:"lrBalanceDb"`
	// Summary scalars
	MinCorrelation float32 `msgpack:"minCorrelation"`
	AvgCorrelation float32 `msgpack:"avgCorrelation"`
	MaxImbalanceDb float32 `msgpack:"maxImbalanceDb"`
}

// DynamicsSeries represents dynamics/DR over time raw data (v1)
type DynamicsSeries struct {
	Version       int       `msgpack:"version"`
	TSec          []float32 `msgpack:"tSec"`
	CrestFactorDb []float32 `msgpack:"crestFactorDb"`
	RMSDb         []float32 `msgpack:"rmsDb"`
	PeakDb        []float32 `msgpack:"peakDb"`
	// Summary scalars
	DRScore       int     `msgpack:"drScore"`
	AvgCrestDb    float32 `msgpack:"avgCrestDb"`
	MinCrestDb    float32 `msgpack:"minCrestDb"`
}

// SaveMsgpackZstd serializes data to MessagePack and compresses with Zstd
func SaveMsgpackZstd(path string, data interface{}) error {
	// Serialize to msgpack
	var buf bytes.Buffer
	enc := msgpack.NewEncoder(&buf)
	enc.SetCustomStructTag("msgpack")
	if err := enc.Encode(data); err != nil {
		return fmt.Errorf("msgpack encode: %w", err)
	}

	// Compress with zstd
	encoder, err := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedDefault))
	if err != nil {
		return fmt.Errorf("create zstd encoder: %w", err)
	}
	defer encoder.Close()

	compressed := encoder.EncodeAll(buf.Bytes(), nil)

	// Write to file
	if err := os.WriteFile(path, compressed, 0644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}

// LoadMsgpackZstd decompresses and deserializes data
func LoadMsgpackZstd(path string, data interface{}) error {
	// Read compressed file
	compressed, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	// Decompress with zstd
	decoder, err := zstd.NewReader(nil)
	if err != nil {
		return fmt.Errorf("create zstd decoder: %w", err)
	}
	defer decoder.Close()

	decompressed, err := decoder.DecodeAll(compressed, nil)
	if err != nil {
		return fmt.Errorf("zstd decode: %w", err)
	}

	// Deserialize from msgpack
	dec := msgpack.NewDecoder(bytes.NewReader(decompressed))
	dec.SetCustomStructTag("msgpack")
	if err := dec.Decode(data); err != nil {
		return fmt.Errorf("msgpack decode: %w", err)
	}

	return nil
}
