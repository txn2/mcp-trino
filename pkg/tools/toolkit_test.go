package tools

import (
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.DefaultLimit != 1000 {
		t.Errorf("expected DefaultLimit 1000, got %d", cfg.DefaultLimit)
	}
	if cfg.MaxLimit != 10000 {
		t.Errorf("expected MaxLimit 10000, got %d", cfg.MaxLimit)
	}
	if cfg.DefaultTimeout != 120*time.Second {
		t.Errorf("expected DefaultTimeout 120s, got %v", cfg.DefaultTimeout)
	}
	if cfg.MaxTimeout != 300*time.Second {
		t.Errorf("expected MaxTimeout 300s, got %v", cfg.MaxTimeout)
	}
}

func TestNewToolkit_DefaultsZeroValues(t *testing.T) {
	// NewToolkit should apply defaults for zero values
	cfg := Config{
		DefaultLimit:   0,
		MaxLimit:       0,
		DefaultTimeout: 0,
		MaxTimeout:     0,
	}

	toolkit := NewToolkit(nil, cfg)

	actualCfg := toolkit.Config()

	if actualCfg.DefaultLimit != 1000 {
		t.Errorf("expected DefaultLimit to default to 1000, got %d", actualCfg.DefaultLimit)
	}
	if actualCfg.MaxLimit != 10000 {
		t.Errorf("expected MaxLimit to default to 10000, got %d", actualCfg.MaxLimit)
	}
	if actualCfg.DefaultTimeout != 120*time.Second {
		t.Errorf("expected DefaultTimeout to default to 120s, got %v", actualCfg.DefaultTimeout)
	}
	if actualCfg.MaxTimeout != 300*time.Second {
		t.Errorf("expected MaxTimeout to default to 300s, got %v", actualCfg.MaxTimeout)
	}
}

func TestNewToolkit_PreservesCustomValues(t *testing.T) {
	cfg := Config{
		DefaultLimit:   500,
		MaxLimit:       5000,
		DefaultTimeout: 60 * time.Second,
		MaxTimeout:     180 * time.Second,
	}

	toolkit := NewToolkit(nil, cfg)

	actualCfg := toolkit.Config()

	if actualCfg.DefaultLimit != 500 {
		t.Errorf("expected DefaultLimit 500, got %d", actualCfg.DefaultLimit)
	}
	if actualCfg.MaxLimit != 5000 {
		t.Errorf("expected MaxLimit 5000, got %d", actualCfg.MaxLimit)
	}
	if actualCfg.DefaultTimeout != 60*time.Second {
		t.Errorf("expected DefaultTimeout 60s, got %v", actualCfg.DefaultTimeout)
	}
	if actualCfg.MaxTimeout != 180*time.Second {
		t.Errorf("expected MaxTimeout 180s, got %v", actualCfg.MaxTimeout)
	}
}

func TestNewToolkit_NegativeValues(t *testing.T) {
	// Negative values should be treated as zero and get defaults
	cfg := Config{
		DefaultLimit:   -1,
		MaxLimit:       -1,
		DefaultTimeout: -1,
		MaxTimeout:     -1,
	}

	toolkit := NewToolkit(nil, cfg)

	actualCfg := toolkit.Config()

	if actualCfg.DefaultLimit != 1000 {
		t.Errorf("expected DefaultLimit to default to 1000 for negative, got %d", actualCfg.DefaultLimit)
	}
	if actualCfg.MaxLimit != 10000 {
		t.Errorf("expected MaxLimit to default to 10000 for negative, got %d", actualCfg.MaxLimit)
	}
	if actualCfg.DefaultTimeout != 120*time.Second {
		t.Errorf("expected DefaultTimeout to default to 120s for negative, got %v", actualCfg.DefaultTimeout)
	}
	if actualCfg.MaxTimeout != 300*time.Second {
		t.Errorf("expected MaxTimeout to default to 300s for negative, got %v", actualCfg.MaxTimeout)
	}
}

func TestToolkit_Client(t *testing.T) {
	toolkit := NewToolkit(nil, DefaultConfig())

	client := toolkit.Client()
	if client != nil {
		t.Error("expected nil client when none provided")
	}
}

func TestConfig_Boundaries(t *testing.T) {
	tests := []struct {
		name          string
		input         Config
		expectedLimit int
		expectedMax   int
		expectedDefTO time.Duration
		expectedMaxTO time.Duration
	}{
		{
			name: "minimum positive values",
			input: Config{
				DefaultLimit:   1,
				MaxLimit:       1,
				DefaultTimeout: 1 * time.Millisecond,
				MaxTimeout:     1 * time.Millisecond,
			},
			expectedLimit: 1,
			expectedMax:   1,
			expectedDefTO: 1 * time.Millisecond,
			expectedMaxTO: 1 * time.Millisecond,
		},
		{
			name: "large values",
			input: Config{
				DefaultLimit:   100000,
				MaxLimit:       1000000,
				DefaultTimeout: 1 * time.Hour,
				MaxTimeout:     24 * time.Hour,
			},
			expectedLimit: 100000,
			expectedMax:   1000000,
			expectedDefTO: 1 * time.Hour,
			expectedMaxTO: 24 * time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			toolkit := NewToolkit(nil, tt.input)
			cfg := toolkit.Config()

			if cfg.DefaultLimit != tt.expectedLimit {
				t.Errorf("expected DefaultLimit %d, got %d", tt.expectedLimit, cfg.DefaultLimit)
			}
			if cfg.MaxLimit != tt.expectedMax {
				t.Errorf("expected MaxLimit %d, got %d", tt.expectedMax, cfg.MaxLimit)
			}
			if cfg.DefaultTimeout != tt.expectedDefTO {
				t.Errorf("expected DefaultTimeout %v, got %v", tt.expectedDefTO, cfg.DefaultTimeout)
			}
			if cfg.MaxTimeout != tt.expectedMaxTO {
				t.Errorf("expected MaxTimeout %v, got %v", tt.expectedMaxTO, cfg.MaxTimeout)
			}
		})
	}
}
