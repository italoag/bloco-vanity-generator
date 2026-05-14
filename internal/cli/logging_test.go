package cli

import (
	"testing"

	"github.com/spf13/cobra"

	"bloco-vgen/internal/config"
)

func TestParseLoggingFlags(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected config.LoggingConfig
	}{
		{
			name: "default logging config",
			args: []string{},
			expected: config.LoggingConfig{
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
			name: "no-logging flag",
			args: []string{"--no-logging"},
			expected: config.LoggingConfig{
				Enabled:     false,
				Level:       "info",
				Format:      "text",
				OutputFile:  "",
				MaxFileSize: 10 * 1024 * 1024,
				MaxFiles:    5,
				BufferSize:  1000,
			},
		},
		{
			name: "custom log level",
			args: []string{"--log-level", "debug"},
			expected: config.LoggingConfig{
				Enabled:     true,
				Level:       "debug",
				Format:      "text",
				OutputFile:  "",
				MaxFileSize: 10 * 1024 * 1024,
				MaxFiles:    5,
				BufferSize:  1000,
			},
		},
		{
			name: "custom log format",
			args: []string{"--log-format", "json"},
			expected: config.LoggingConfig{
				Enabled:     true,
				Level:       "info",
				Format:      "json",
				OutputFile:  "",
				MaxFileSize: 10 * 1024 * 1024,
				MaxFiles:    5,
				BufferSize:  1000,
			},
		},
		{
			name: "custom log file",
			args: []string{"--log-file", "/tmp/test.log"},
			expected: config.LoggingConfig{
				Enabled:     true,
				Level:       "info",
				Format:      "text",
				OutputFile:  "/tmp/test.log",
				MaxFileSize: 10 * 1024 * 1024,
				MaxFiles:    5,
				BufferSize:  1000,
			},
		},
		{
			name: "all logging flags",
			args: []string{"--log-level", "error", "--log-format", "structured", "--log-file", "/var/log/app.log"},
			expected: config.LoggingConfig{
				Enabled:     true,
				Level:       "error",
				Format:      "structured",
				OutputFile:  "/var/log/app.log",
				MaxFileSize: 10 * 1024 * 1024,
				MaxFiles:    5,
				BufferSize:  1000,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test application with default config
			cfg := config.DefaultConfig()
			app := &Application{
				config: cfg,
			}

			// Create a test command with logging flags
			cmd := &cobra.Command{
				Use: "test",
			}

			// Add the same flags as the real application
			flags := cmd.Flags()
			flags.String("log-level", "info", "Logging level (error, warn, info, debug)")
			flags.Bool("no-logging", false, "Disable logging completely")
			flags.String("log-file", "", "Log file path (default: stdout)")
			flags.String("log-format", "text", "Log format (text, json, structured)")

			// Parse the test arguments
			cmd.SetArgs(tt.args)
			err := cmd.ParseFlags(tt.args)
			if err != nil {
				t.Fatalf("Failed to parse flags: %v", err)
			}

			// Call the parseLoggingFlags method
			err = app.parseLoggingFlags(cmd)
			if err != nil {
				t.Fatalf("parseLoggingFlags() error: %v", err)
			}

			// Check the results
			if app.config.Logging.Enabled != tt.expected.Enabled {
				t.Errorf("Enabled = %v, want %v", app.config.Logging.Enabled, tt.expected.Enabled)
			}

			if app.config.Logging.Level != tt.expected.Level {
				t.Errorf("Level = %v, want %v", app.config.Logging.Level, tt.expected.Level)
			}

			if app.config.Logging.Format != tt.expected.Format {
				t.Errorf("Format = %v, want %v", app.config.Logging.Format, tt.expected.Format)
			}

			if app.config.Logging.OutputFile != tt.expected.OutputFile {
				t.Errorf("OutputFile = %v, want %v", app.config.Logging.OutputFile, tt.expected.OutputFile)
			}
		})
	}
}

func TestParseLoggingFlagsValidation(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "valid log level",
			args:    []string{"--log-level", "debug"},
			wantErr: false,
		},
		{
			name:    "valid log format",
			args:    []string{"--log-format", "json"},
			wantErr: false,
		},
		{
			name:    "no-logging overrides other flags",
			args:    []string{"--no-logging", "--log-level", "debug"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test application with default config
			cfg := config.DefaultConfig()
			app := &Application{
				config: cfg,
			}

			// Create a test command with logging flags
			cmd := &cobra.Command{
				Use: "test",
			}

			// Add the same flags as the real application
			flags := cmd.Flags()
			flags.String("log-level", "info", "Logging level (error, warn, info, debug)")
			flags.Bool("no-logging", false, "Disable logging completely")
			flags.String("log-file", "", "Log file path (default: stdout)")
			flags.String("log-format", "text", "Log format (text, json, structured)")

			// Parse the test arguments
			cmd.SetArgs(tt.args)
			err := cmd.ParseFlags(tt.args)
			if err != nil {
				t.Fatalf("Failed to parse flags: %v", err)
			}

			// Call the parseLoggingFlags method
			err = app.parseLoggingFlags(cmd)

			if tt.wantErr && err == nil {
				t.Errorf("parseLoggingFlags() expected error, got nil")
			}

			if !tt.wantErr && err != nil {
				t.Errorf("parseLoggingFlags() unexpected error: %v", err)
			}
		})
	}
}

func TestNoLoggingOverridesOtherFlags(t *testing.T) {
	// Create a test application with default config
	cfg := config.DefaultConfig()
	app := &Application{
		config: cfg,
	}

	// Create a test command with logging flags
	cmd := &cobra.Command{
		Use: "test",
	}

	// Add the same flags as the real application
	flags := cmd.Flags()
	flags.String("log-level", "info", "Logging level (error, warn, info, debug)")
	flags.Bool("no-logging", false, "Disable logging completely")
	flags.String("log-file", "", "Log file path (default: stdout)")
	flags.String("log-format", "text", "Log format (text, json, structured)")

	// Test that --no-logging overrides other logging flags
	args := []string{"--no-logging", "--log-level", "debug", "--log-format", "json", "--log-file", "/tmp/test.log"}
	cmd.SetArgs(args)
	err := cmd.ParseFlags(args)
	if err != nil {
		t.Fatalf("Failed to parse flags: %v", err)
	}

	// Call the parseLoggingFlags method
	err = app.parseLoggingFlags(cmd)
	if err != nil {
		t.Fatalf("parseLoggingFlags() error: %v", err)
	}

	// Logging should be disabled regardless of other flags
	if app.config.Logging.Enabled {
		t.Errorf("Expected logging to be disabled when --no-logging is set")
	}

	// Other flags should retain their original values since --no-logging was processed first
	if app.config.Logging.Level != "info" {
		t.Errorf("Expected level to remain 'info', got %v", app.config.Logging.Level)
	}
}
