package static

import (
	"os"
	"time"
)

// Config configures the static file provider.
type Config struct {
	// FilePath is the path to the YAML or JSON file containing semantic metadata.
	FilePath string

	// WatchInterval enables periodic reloading of the file.
	// Set to 0 to disable watching. Default: 0 (disabled).
	WatchInterval time.Duration
}

// FromEnv loads configuration from environment variables.
//
// Environment variables:
//   - SEMANTIC_STATIC_FILE: Path to the semantic metadata file
//   - SEMANTIC_STATIC_WATCH_INTERVAL: Reload interval (e.g., "30s", "1m")
func FromEnv() Config {
	cfg := Config{
		FilePath: os.Getenv("SEMANTIC_STATIC_FILE"),
	}

	if interval := os.Getenv("SEMANTIC_STATIC_WATCH_INTERVAL"); interval != "" {
		if d, err := time.ParseDuration(interval); err == nil {
			cfg.WatchInterval = d
		}
	}

	return cfg
}

// Validate checks if the configuration is valid.
func (c Config) Validate() error {
	if c.FilePath == "" {
		return ErrNoFilePath
	}
	return nil
}
