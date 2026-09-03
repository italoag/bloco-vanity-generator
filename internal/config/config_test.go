package config

import (
	"os"
	"testing"
)

func TestDefaultConfig_LoggingConfig(t *testing.T) {
	cfg := DefaultConfig()

	// Test default logging configuration
	if !cfg.Logging.Enabled {
		t.Errorf("Expected logging to be enabled by default")
	}

	if cfg.Logging.Level != "info" {
		t.Errorf("Expected default log level to be 'info', got %s", cfg.Logging.Level)
	}

	if cfg.Logging.Format != "text" {
		t.Errorf("Expected default log format to be 'text', got %s", cfg.Logging.Format)
	}

	if cfg.Logging.OutputFile != "" {
		t.Errorf("Expected default output file to be empty, got %s", cfg.Logging.OutputFile)
	}

	if cfg.Logging.MaxFileSize != 10*1024*1024 {
		t.Errorf("Expected default max file size to be 10MB, got %d", cfg.Logging.MaxFileSize)
	}

	if cfg.Logging.MaxFiles != 5 {
		t.Errorf("Expected default max files to be 5, got %d", cfg.Logging.MaxFiles)
	}

	if cfg.Logging.BufferSize != 1000 {
		t.Errorf("Expected default buffer size to be 1000, got %d", cfg.Logging.BufferSize)
	}
}

func TestConfig_Validate_LoggingConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid logging config",
			config: &Config{
				Worker:   DefaultConfig().Worker,
				TUI:      DefaultConfig().TUI,
				Crypto:   DefaultConfig().Crypto,
				CLI:      DefaultConfig().CLI,
				KeyStore: DefaultConfig().KeyStore,
				Logging: LoggingConfig{
					Enabled:     true,
					Level:       "info",
					Format:      "text",
					OutputFile:  "",
					MaxFileSize: 1024,
					MaxFiles:    5,
					BufferSize:  100,
				},
			},
			wantErr: false,
		},
		{
			name: "invalid log level",
			config: &Config{
				Worker:   DefaultConfig().Worker,
				TUI:      DefaultConfig().TUI,
				Crypto:   DefaultConfig().Crypto,
				CLI:      DefaultConfig().CLI,
				KeyStore: DefaultConfig().KeyStore,
				Logging: LoggingConfig{
					Enabled:     true,
					Level:       "invalid",
					Format:      "text",
					OutputFile:  "",
					MaxFileSize: 1024,
					MaxFiles:    5,
					BufferSize:  100,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid log format",
			config: &Config{
				Worker:   DefaultConfig().Worker,
				TUI:      DefaultConfig().TUI,
				Crypto:   DefaultConfig().Crypto,
				CLI:      DefaultConfig().CLI,
				KeyStore: DefaultConfig().KeyStore,
				Logging: LoggingConfig{
					Enabled:     true,
					Level:       "info",
					Format:      "invalid",
					OutputFile:  "",
					MaxFileSize: 1024,
					MaxFiles:    5,
					BufferSize:  100,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid max file size",
			config: &Config{
				Worker:   DefaultConfig().Worker,
				TUI:      DefaultConfig().TUI,
				Crypto:   DefaultConfig().Crypto,
				CLI:      DefaultConfig().CLI,
				KeyStore: DefaultConfig().KeyStore,
				Logging: LoggingConfig{
					Enabled:     true,
					Level:       "info",
					Format:      "text",
					OutputFile:  "",
					MaxFileSize: -1,
					MaxFiles:    5,
					BufferSize:  100,
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr && err == nil {
				t.Errorf("Validate() expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Validate() unexpected error: %v", err)
			}
		})
	}
}

func TestConfig_LoadFromEnvironment_LoggingConfig(t *testing.T) {
	// Save original environment
	originalVars := map[string]string{
		"BLOCO_LOGGING_ENABLED": os.Getenv("BLOCO_LOGGING_ENABLED"),
		"BLOCO_LOG_LEVEL":       os.Getenv("BLOCO_LOG_LEVEL"),
		"BLOCO_LOG_FORMAT":      os.Getenv("BLOCO_LOG_FORMAT"),
		"BLOCO_LOG_FILE":        os.Getenv("BLOCO_LOG_FILE"),
	}

	// Clean up after test
	defer func() {
		for key, value := range originalVars {
			if value == "" {
				_ = os.Unsetenv(key)
			} else {
				_ = os.Setenv(key, value)
			}
		}
	}()

	tests := []struct {
		name     string
		envVars  map[string]string
		expected LoggingConfig
	}{
		{
			name: "default values",
			envVars: map[string]string{
				"BLOCO_LOGGING_ENABLED": "",
				"BLOCO_LOG_LEVEL":       "",
				"BLOCO_LOG_FORMAT":      "",
				"BLOCO_LOG_FILE":        "",
			},
			expected: LoggingConfig{
				Enabled:     true,
				Level:       "info",
				Format:      "text",
				OutputFile:  "",
				MaxFileSize: 10 * 1024 * 1024,
				MaxFiles:    5,
				BufferSize:  1000,
			},
		},
		{
			name: "environment overrides",
			envVars: map[string]string{
				"BLOCO_LOGGING_ENABLED": "false",
				"BLOCO_LOG_LEVEL":       "debug",
				"BLOCO_LOG_FORMAT":      "json",
				"BLOCO_LOG_FILE":        "/tmp/test.log",
			},
			expected: LoggingConfig{
				Enabled:     false,
				Level:       "debug",
				Format:      "json",
				OutputFile:  "/tmp/test.log",
				MaxFileSize: 10 * 1024 * 1024,
				MaxFiles:    5,
				BufferSize:  1000,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for key, value := range tt.envVars {
				if value == "" {
					_ = os.Unsetenv(key)
				} else {
					_ = os.Setenv(key, value)
				}
			}

			// Create config and load from environment
			cfg := DefaultConfig()
			cfg.LoadFromEnvironment()

			// Check results
			if cfg.Logging.Enabled != tt.expected.Enabled {
				t.Errorf("Enabled = %v, want %v", cfg.Logging.Enabled, tt.expected.Enabled)
			}

			if cfg.Logging.Level != tt.expected.Level {
				t.Errorf("Level = %v, want %v", cfg.Logging.Level, tt.expected.Level)
			}

			if cfg.Logging.Format != tt.expected.Format {
				t.Errorf("Format = %v, want %v", cfg.Logging.Format, tt.expected.Format)
			}

			if cfg.Logging.OutputFile != tt.expected.OutputFile {
				t.Errorf("OutputFile = %v, want %v", cfg.Logging.OutputFile, tt.expected.OutputFile)
			}
		})
	}
}

func TestConfig_ApplyOverrides_LoggingConfig(t *testing.T) {
	cfg := DefaultConfig()

	// Test overrides
	enabled := false
	level := "error"
	format := "json"
	file := "/var/log/app.log"

	overrides := ConfigOverrides{
		LoggingEnabled: &enabled,
		LogLevel:       &level,
		LogFormat:      &format,
		LogFile:        &file,
	}

	cfg.ApplyOverrides(overrides)

	// Check results
	if cfg.Logging.Enabled != enabled {
		t.Errorf("Enabled = %v, want %v", cfg.Logging.Enabled, enabled)
	}

	if cfg.Logging.Level != level {
		t.Errorf("Level = %v, want %v", cfg.Logging.Level, level)
	}

	if cfg.Logging.Format != format {
		t.Errorf("Format = %v, want %v", cfg.Logging.Format, format)
	}

	if cfg.Logging.OutputFile != file {
		t.Errorf("OutputFile = %v, want %v", cfg.Logging.OutputFile, file)
	}
}

// TestValidateRejectsPermissiveKeystoreFileMode is a regression test for
// GitHub issue #18: the file mode used to be validated up to 0777, which would
// have allowed private key material to be written world-readable once the
// field was actually applied.
func TestValidateRejectsPermissiveKeystoreFileMode(t *testing.T) {
	tests := []struct {
		name      string
		mode      int
		wantError bool
	}{
		{"owner read/write", 0o600, false},
		{"owner read-only", 0o400, false},
		{"group readable", 0o640, true},
		{"world readable", 0o644, true},
		{"world writable", 0o666, true},
		{"everything", 0o777, true},
		{"zero", 0, true},
		{"negative", -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.KeyStore.FileMode = tt.mode

			err := cfg.Validate()
			if tt.wantError && err == nil {
				t.Errorf("expected file mode %04o to be rejected", tt.mode)
			}
			if !tt.wantError && err != nil {
				t.Errorf("expected file mode %04o to be accepted, got %v", tt.mode, err)
			}
		})
	}
}
