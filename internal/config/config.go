package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Scanner  ScannerConfig  `yaml:"scanner"`
	Storage  StorageConfig  `yaml:"storage"`
	FFmpeg   FFmpegConfig   `yaml:"ffmpeg"`
}

type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type DatabaseConfig struct {
	Driver string `yaml:"driver"`
	DSN    string `yaml:"dsn"`
}

type ScannerConfig struct {
	DefaultInterval  string `yaml:"default_interval"`
	WorkerCount      int    `yaml:"worker_count"`
	BatchSize        int    `yaml:"batch_size"`
	MaxRetries       int    `yaml:"max_retries"`
	RetryBackoffBase int    `yaml:"retry_backoff_base"`
}

type StorageConfig struct {
	ArtifactsPath string `yaml:"artifacts_path"`
	TempPath      string `yaml:"temp_path"`
}

type FFmpegConfig struct {
	FFprobePath string `yaml:"ffprobe_path"`
	FFmpegPath  string `yaml:"ffmpeg_path"`
}

func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host: "0.0.0.0",
			Port: 8080,
		},
		Database: DatabaseConfig{
			Driver: "sqlite3",
			DSN:    "./ottavia.db",
		},
		Scanner: ScannerConfig{
			DefaultInterval:  "15m",
			WorkerCount:      4,
			BatchSize:        100,
			MaxRetries:       3,
			RetryBackoffBase: 60,
		},
		Storage: StorageConfig{
			ArtifactsPath: "./artifacts/data",
			TempPath:      "./artifacts/temp",
		},
		FFmpeg: FFmpegConfig{
			FFprobePath: "ffprobe",
			FFmpegPath:  "ffmpeg",
		},
	}
}

func Load(path string) (*Config, error) {
	cfg := DefaultConfig()

	if path == "" {
		return cfg, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) Save(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
