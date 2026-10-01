package client

import (
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/trinodb/trino-go-client/trino"
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
		_ = os.Unsetenv(v)
	}
}

func TestFromEnv_Defaults(t *testing.T) {
	clearTrinoEnvVars(t)

	cfg := FromEnv()
	if cfg.Host != "localhost" {
		t.Errorf("expected default Host, got %q", cfg.Host)
	}
	if cfg.SSL {
		t.Error("expected SSL false for localhost")
	}
}

func TestFromEnv_RemoteHostEnablesSSL(t *testing.T) {
	clearTrinoEnvVars(t)
	t.Setenv("TRINO_HOST", "trino.example.com")

	cfg := FromEnv()
	if cfg.Host != "trino.example.com" {
		t.Errorf("expected Host 'trino.example.com', got %q", cfg.Host)
	}
	if !cfg.SSL {
		t.Error("expected SSL true for remote host")
	}
	if cfg.Port != 443 {
		t.Errorf("expected Port 443 for SSL, got %d", cfg.Port)
	}
}

func TestFromEnv_LocalhostKeepsSSLDisabled(t *testing.T) {
	clearTrinoEnvVars(t)
	t.Setenv("TRINO_HOST", "localhost")

	cfg := FromEnv()
	if cfg.SSL {
		t.Error("expected SSL false for localhost")
	}
}

func TestFromEnv_LoopbackKeepsSSLDisabled(t *testing.T) {
	clearTrinoEnvVars(t)
	t.Setenv("TRINO_HOST", "127.0.0.1")

	cfg := FromEnv()
	if cfg.SSL {
		t.Error("expected SSL false for 127.0.0.1")
	}
}

func TestFromEnv_CustomPort(t *testing.T) {
	clearTrinoEnvVars(t)
	t.Setenv("TRINO_PORT", "9090")

	cfg := FromEnv()
	if cfg.Port != 9090 {
		t.Errorf("expected Port 9090, got %d", cfg.Port)
	}
}

func TestFromEnv_InvalidPortKeepsDefault(t *testing.T) {
	clearTrinoEnvVars(t)
	t.Setenv("TRINO_PORT", "invalid")

	cfg := FromEnv()
	if cfg.Port != 8080 {
		t.Errorf("expected default Port 8080, got %d", cfg.Port)
	}
}

func TestFromEnv_UserAndPassword(t *testing.T) {
	clearTrinoEnvVars(t)
	t.Setenv("TRINO_USER", "testuser")
	t.Setenv("TRINO_PASSWORD", "testpass")

	cfg := FromEnv()
	if cfg.User != "testuser" {
		t.Errorf("expected User 'testuser', got %q", cfg.User)
	}
	if cfg.Password != "testpass" {
		t.Errorf("expected Password 'testpass', got %q", cfg.Password)
	}
}

func TestFromEnv_CatalogAndSchema(t *testing.T) {
	clearTrinoEnvVars(t)
	t.Setenv("TRINO_CATALOG", "hive")
	t.Setenv("TRINO_SCHEMA", "sales")

	cfg := FromEnv()
	if cfg.Catalog != "hive" {
		t.Errorf("expected Catalog 'hive', got %q", cfg.Catalog)
	}
	if cfg.Schema != "sales" {
		t.Errorf("expected Schema 'sales', got %q", cfg.Schema)
	}
}

func TestFromEnv_SSLSettingsTrue(t *testing.T) {
	clearTrinoEnvVars(t)
	t.Setenv("TRINO_SSL", "true")
	t.Setenv("TRINO_SSL_VERIFY", "true")

	cfg := FromEnv()
	if !cfg.SSL {
		t.Error("expected SSL true")
	}
	if !cfg.SSLVerify {
		t.Error("expected SSLVerify true")
	}
}

func TestFromEnv_SSLSettingsWithOne(t *testing.T) {
	clearTrinoEnvVars(t)
	t.Setenv("TRINO_SSL", "1")
	t.Setenv("TRINO_SSL_VERIFY", "1")

	cfg := FromEnv()
	if !cfg.SSL {
		t.Error("expected SSL true with '1'")
	}
	if !cfg.SSLVerify {
		t.Error("expected SSLVerify true with '1'")
	}
}

func TestFromEnv_SSLSettingsFalse(t *testing.T) {
	clearTrinoEnvVars(t)
	t.Setenv("TRINO_SSL", "false")
	t.Setenv("TRINO_SSL_VERIFY", "false")

	cfg := FromEnv()
	if cfg.SSL {
		t.Error("expected SSL false")
	}
	if cfg.SSLVerify {
		t.Error("expected SSLVerify false")
	}
}

func TestFromEnv_TimeoutInSeconds(t *testing.T) {
	clearTrinoEnvVars(t)
	t.Setenv("TRINO_TIMEOUT", "60")

	cfg := FromEnv()
	if cfg.Timeout != 60*time.Second {
		t.Errorf("expected Timeout 60s, got %v", cfg.Timeout)
	}
}

func TestFromEnv_InvalidTimeoutKeepsDefault(t *testing.T) {
	clearTrinoEnvVars(t)
	t.Setenv("TRINO_TIMEOUT", "invalid")

	cfg := FromEnv()
	if cfg.Timeout != 120*time.Second {
		t.Errorf("expected default Timeout 120s, got %v", cfg.Timeout)
	}
}

func TestFromEnv_CustomSource(t *testing.T) {
	clearTrinoEnvVars(t)
	t.Setenv("TRINO_SOURCE", "my-app")

	cfg := FromEnv()
	if cfg.Source != "my-app" {
		t.Errorf("expected Source 'my-app', got %q", cfg.Source)
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
			expected: "https://admin@trino.example.com:443?catalog=hive&schema=default&source=test",
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
			expected: "https://admin@trino.example.com:443?catalog=hive&schema=default&source=test&sslVerify=false",
		},
		{
			name: "HTTPS with password",
			config: Config{
				Host:      "trino.example.com",
				Port:      443,
				User:      "admin",
				Password:  "secret123",
				SSL:       true,
				SSLVerify: true,
				Source:    "test",
				Catalog:   "hive",
				Schema:    "default",
			},
			expected: "https://admin:secret123@trino.example.com:443?catalog=hive&schema=default&source=test",
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
			expected: "http://admin@localhost:8080?catalog=memory&source=test",
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

// TestConfig_DSN_DriverParse parses DSN the way the Trino driver does, so the
// session catalog and schema the driver sends cannot drift from Config again.
func TestConfig_DSN_DriverParse(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		wantCatalog string
		wantSchema  string
	}{
		{
			name:        "catalog and schema",
			config:      Config{Host: "trino.example.com", Port: 443, User: "admin", SSL: true, Source: "test", Catalog: "hive", Schema: "sales"},
			wantCatalog: "hive",
			wantSchema:  "sales",
		},
		{
			name:        "catalog only",
			config:      Config{Host: "localhost", Port: 8080, User: "admin", Source: "test", Catalog: "memory"},
			wantCatalog: "memory",
		},
		{
			name:   "schema without catalog is dropped",
			config: Config{Host: "localhost", Port: 8080, User: "admin", Source: "test", Schema: "default"},
		},
		{
			name:   "neither",
			config: Config{Host: "localhost", Port: 8080, User: "admin", Source: "test"},
		},
		{
			name:   "bracketed IPv6 host",
			config: Config{Host: "[::1]", Port: 8080, User: "admin", Source: "test"},
		},
		{
			name:   "bare IPv6 host",
			config: Config{Host: "::1", Port: 8080, User: "admin", Source: "test"},
		},
		{
			name: "values needing escapes",
			config: Config{
				Host: "localhost", Port: 8080, User: "admin", Password: "p@ss/w&rd?#",
				Source: "my app", Catalog: "a&b", Schema: "c=d",
			},
			wantCatalog: "a&b",
			wantSchema:  "c=d",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := trino.ParseDSN(tt.config.DSN())
			if err != nil {
				t.Fatalf("ParseDSN(%q): %v", tt.config.DSN(), err)
			}
			if parsed.Catalog != tt.wantCatalog {
				t.Errorf("catalog = %q, want %q", parsed.Catalog, tt.wantCatalog)
			}
			if parsed.Schema != tt.wantSchema {
				t.Errorf("schema = %q, want %q", parsed.Schema, tt.wantSchema)
			}
			if parsed.Source != tt.config.Source {
				t.Errorf("source = %q, want %q", parsed.Source, tt.config.Source)
			}
			server, err := url.Parse(parsed.ServerURI)
			if err != nil {
				t.Fatalf("server URI %q: %v", parsed.ServerURI, err)
			}
			wantHost := net.JoinHostPort(strings.Trim(tt.config.Host, "[]"), strconv.Itoa(tt.config.Port))
			if server.Host != wantHost {
				t.Errorf("host = %q, want %q", server.Host, wantHost)
			}
			password, _ := server.User.Password()
			if server.User.Username() != tt.config.User || password != tt.config.Password {
				t.Errorf("credentials = %q/%q, want %q/%q", server.User.Username(), password, tt.config.User, tt.config.Password)
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
