package client

import (
	"os"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Host != "localhost" {
		t.Errorf("expected Host 'localhost', got %q", cfg.Host)
	}
	if cfg.Port != 8080 {
		t.Errorf("expected Port 8080, got %d", cfg.Port)
	}
	if cfg.SSL {
		t.Error("expected SSL to be false by default")
	}
	if !cfg.SSLVerify {
		t.Error("expected SSLVerify to be true by default")
	}
	if cfg.Timeout != 120*time.Second {
		t.Errorf("expected Timeout 120s, got %v", cfg.Timeout)
	}
	if cfg.Source != "mcp-trino" {
		t.Errorf("expected Source 'mcp-trino', got %q", cfg.Source)
	}
	if cfg.Catalog != "memory" {
		t.Errorf("expected Catalog 'memory', got %q", cfg.Catalog)
	}
	if cfg.Schema != "default" {
		t.Errorf("expected Schema 'default', got %q", cfg.Schema)
	}
}

func TestFromEnv(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		validate func(t *testing.T, cfg Config)
	}{
		{
			name:    "defaults with no env vars",
			envVars: map[string]string{},
			validate: func(t *testing.T, cfg Config) {
				if cfg.Host != "localhost" {
					t.Errorf("expected default Host, got %q", cfg.Host)
				}
				if cfg.SSL {
					t.Error("expected SSL false for localhost")
				}
			},
		},
		{
			name: "remote host enables SSL",
			envVars: map[string]string{
				"TRINO_HOST": "trino.example.com",
			},
			validate: func(t *testing.T, cfg Config) {
				if cfg.Host != "trino.example.com" {
					t.Errorf("expected Host 'trino.example.com', got %q", cfg.Host)
				}
				if !cfg.SSL {
					t.Error("expected SSL true for remote host")
				}
				if cfg.Port != 443 {
					t.Errorf("expected Port 443 for SSL, got %d", cfg.Port)
				}
			},
		},
		{
			name: "localhost keeps SSL disabled",
			envVars: map[string]string{
				"TRINO_HOST": "localhost",
			},
			validate: func(t *testing.T, cfg Config) {
				if cfg.SSL {
					t.Error("expected SSL false for localhost")
				}
			},
		},
		{
			name: "127.0.0.1 keeps SSL disabled",
			envVars: map[string]string{
				"TRINO_HOST": "127.0.0.1",
			},
			validate: func(t *testing.T, cfg Config) {
				if cfg.SSL {
					t.Error("expected SSL false for 127.0.0.1")
				}
			},
		},
		{
			name: "custom port",
			envVars: map[string]string{
				"TRINO_PORT": "9090",
			},
			validate: func(t *testing.T, cfg Config) {
				if cfg.Port != 9090 {
					t.Errorf("expected Port 9090, got %d", cfg.Port)
				}
			},
		},
		{
			name: "invalid port keeps default",
			envVars: map[string]string{
				"TRINO_PORT": "invalid",
			},
			validate: func(t *testing.T, cfg Config) {
				if cfg.Port != 8080 {
					t.Errorf("expected default Port 8080, got %d", cfg.Port)
				}
			},
		},
		{
			name: "user and password",
			envVars: map[string]string{
				"TRINO_USER":     "testuser",
				"TRINO_PASSWORD": "testpass",
			},
			validate: func(t *testing.T, cfg Config) {
				if cfg.User != "testuser" {
					t.Errorf("expected User 'testuser', got %q", cfg.User)
				}
				if cfg.Password != "testpass" {
					t.Errorf("expected Password 'testpass', got %q", cfg.Password)
				}
			},
		},
		{
			name: "catalog and schema",
			envVars: map[string]string{
				"TRINO_CATALOG": "hive",
				"TRINO_SCHEMA":  "sales",
			},
			validate: func(t *testing.T, cfg Config) {
				if cfg.Catalog != "hive" {
					t.Errorf("expected Catalog 'hive', got %q", cfg.Catalog)
				}
				if cfg.Schema != "sales" {
					t.Errorf("expected Schema 'sales', got %q", cfg.Schema)
				}
			},
		},
		{
			name: "SSL settings true",
			envVars: map[string]string{
				"TRINO_SSL":        "true",
				"TRINO_SSL_VERIFY": "true",
			},
			validate: func(t *testing.T, cfg Config) {
				if !cfg.SSL {
					t.Error("expected SSL true")
				}
				if !cfg.SSLVerify {
					t.Error("expected SSLVerify true")
				}
			},
		},
		{
			name: "SSL settings with 1",
			envVars: map[string]string{
				"TRINO_SSL":        "1",
				"TRINO_SSL_VERIFY": "1",
			},
			validate: func(t *testing.T, cfg Config) {
				if !cfg.SSL {
					t.Error("expected SSL true with '1'")
				}
				if !cfg.SSLVerify {
					t.Error("expected SSLVerify true with '1'")
				}
			},
		},
		{
			name: "SSL settings false",
			envVars: map[string]string{
				"TRINO_SSL":        "false",
				"TRINO_SSL_VERIFY": "false",
			},
			validate: func(t *testing.T, cfg Config) {
				if cfg.SSL {
					t.Error("expected SSL false")
				}
				if cfg.SSLVerify {
					t.Error("expected SSLVerify false")
				}
			},
		},
		{
			name: "timeout in seconds",
			envVars: map[string]string{
				"TRINO_TIMEOUT": "60",
			},
			validate: func(t *testing.T, cfg Config) {
				if cfg.Timeout != 60*time.Second {
					t.Errorf("expected Timeout 60s, got %v", cfg.Timeout)
				}
			},
		},
		{
			name: "invalid timeout keeps default",
			envVars: map[string]string{
				"TRINO_TIMEOUT": "invalid",
			},
			validate: func(t *testing.T, cfg Config) {
				if cfg.Timeout != 120*time.Second {
					t.Errorf("expected default Timeout 120s, got %v", cfg.Timeout)
				}
			},
		},
		{
			name: "custom source",
			envVars: map[string]string{
				"TRINO_SOURCE": "my-app",
			},
			validate: func(t *testing.T, cfg Config) {
				if cfg.Source != "my-app" {
					t.Errorf("expected Source 'my-app', got %q", cfg.Source)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear all TRINO_ env vars
			clearTrinoEnvVars(t)

			// Set test env vars
			for k, v := range tt.envVars {
				t.Setenv(k, v)
			}

			cfg := FromEnv()
			tt.validate(t, cfg)
		})
	}
}

func clearTrinoEnvVars(t *testing.T) {
	t.Helper()
	envVars := []string{
		"TRINO_HOST",
		"TRINO_PORT",
		"TRINO_USER",
		"TRINO_PASSWORD",
		"TRINO_CATALOG",
		"TRINO_SCHEMA",
		"TRINO_SSL",
		"TRINO_SSL_VERIFY",
		"TRINO_TIMEOUT",
		"TRINO_SOURCE",
	}
	for _, v := range envVars {
		os.Unsetenv(v)
	}
}

func TestConfig_DSN(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		expected string
	}{
		{
			name: "basic HTTP",
			config: Config{
				Host:    "localhost",
				Port:    8080,
				User:    "admin",
				SSL:     false,
				Source:  "test",
				Catalog: "",
			},
			expected: "http://admin@localhost:8080?source=test",
		},
		{
			name: "HTTPS with catalog and schema",
			config: Config{
				Host:      "trino.example.com",
				Port:      443,
				User:      "admin",
				SSL:       true,
				SSLVerify: true,
				Source:    "test",
				Catalog:   "hive",
				Schema:    "default",
			},
			expected: "https://admin@trino.example.com:443/hive/default?source=test",
		},
		{
			name: "HTTPS without SSL verify",
			config: Config{
				Host:      "trino.example.com",
				Port:      443,
				User:      "admin",
				SSL:       true,
				SSLVerify: false,
				Source:    "test",
				Catalog:   "hive",
				Schema:    "default",
			},
			expected: "https://admin@trino.example.com:443/hive/default?source=test&sslVerify=false",
		},
		{
			name: "catalog only, no schema",
			config: Config{
				Host:    "localhost",
				Port:    8080,
				User:    "admin",
				SSL:     false,
				Source:  "test",
				Catalog: "memory",
				Schema:  "",
			},
			expected: "http://admin@localhost:8080/memory?source=test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dsn := tt.config.DSN()
			if dsn != tt.expected {
				t.Errorf("expected DSN %q, got %q", tt.expected, dsn)
			}
		})
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name      string
		config    Config
		wantError bool
		errMsg    string
	}{
		{
			name: "valid config",
			config: Config{
				Host: "localhost",
				Port: 8080,
				User: "admin",
			},
			wantError: false,
		},
		{
			name: "missing host",
			config: Config{
				Host: "",
				Port: 8080,
				User: "admin",
			},
			wantError: true,
			errMsg:    "host is required",
		},
		{
			name: "missing user",
			config: Config{
				Host: "localhost",
				Port: 8080,
				User: "",
			},
			wantError: true,
			errMsg:    "user is required",
		},
		{
			name: "port too low",
			config: Config{
				Host: "localhost",
				Port: 0,
				User: "admin",
			},
			wantError: true,
			errMsg:    "port must be between 1 and 65535",
		},
		{
			name: "port negative",
			config: Config{
				Host: "localhost",
				Port: -1,
				User: "admin",
			},
			wantError: true,
			errMsg:    "port must be between 1 and 65535",
		},
		{
			name: "port too high",
			config: Config{
				Host: "localhost",
				Port: 65536,
				User: "admin",
			},
			wantError: true,
			errMsg:    "port must be between 1 and 65535",
		},
		{
			name: "port at max boundary",
			config: Config{
				Host: "localhost",
				Port: 65535,
				User: "admin",
			},
			wantError: false,
		},
		{
			name: "port at min boundary",
			config: Config{
				Host: "localhost",
				Port: 1,
				User: "admin",
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantError {
				if err == nil {
					t.Error("expected error, got nil")
				} else if err.Error() != tt.errMsg {
					t.Errorf("expected error %q, got %q", tt.errMsg, err.Error())
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
