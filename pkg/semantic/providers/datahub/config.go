package datahub

import (
	"errors"
	"os"
	"time"
)

// Config configures the DataHub provider.
type Config struct {
	// Endpoint is the DataHub GraphQL endpoint.
	// Example: "https://datahub.example.com/api/graphql"
	Endpoint string

	// Token is the authentication token for DataHub.
	Token string

	// Platform is the data platform name used in DataHub URNs.
	// Default: "trino"
	Platform string

	// Environment is the DataHub environment.
	// Default: "PROD"
	Environment string

	// Timeout is the HTTP request timeout.
	// Default: 30s
	Timeout time.Duration
}

// Common errors.
var (
	ErrNoEndpoint = errors.New("datahub: endpoint is required")
	ErrNoToken    = errors.New("datahub: token is required")
)

// FromEnv loads configuration from environment variables.
//
// Environment variables:
//   - DATAHUB_ENDPOINT: DataHub GraphQL endpoint URL
//   - DATAHUB_TOKEN: Authentication token
//   - DATAHUB_PLATFORM: Data platform name (default: "trino")
//   - DATAHUB_ENVIRONMENT: Environment name (default: "PROD")
//   - DATAHUB_TIMEOUT: Request timeout (default: "30s")
func FromEnv() Config {
	cfg := Config{
		Endpoint:    os.Getenv("DATAHUB_ENDPOINT"),
		Token:       os.Getenv("DATAHUB_TOKEN"),
		Platform:    getEnvOrDefault("DATAHUB_PLATFORM", "trino"),
		Environment: getEnvOrDefault("DATAHUB_ENVIRONMENT", "PROD"),
		Timeout:     30 * time.Second,
	}

	if timeout := os.Getenv("DATAHUB_TIMEOUT"); timeout != "" {
		if d, err := time.ParseDuration(timeout); err == nil {
			cfg.Timeout = d
		}
	}

	return cfg
}

// Validate checks if the configuration is valid.
func (c Config) Validate() error {
	if c.Endpoint == "" {
		return ErrNoEndpoint
	}
	if c.Token == "" {
		return ErrNoToken
	}
	return nil
}

// WithDefaults returns a copy of the config with defaults applied.
func (c Config) WithDefaults() Config {
	if c.Platform == "" {
		c.Platform = "trino"
	}
	if c.Environment == "" {
		c.Environment = "PROD"
	}
	if c.Timeout <= 0 {
		c.Timeout = 30 * time.Second
	}
	return c
}

func getEnvOrDefault(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}
