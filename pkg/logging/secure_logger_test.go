package logging

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewSecureLogger(t *testing.T) {
	t.Run("Default config", func(t *testing.T) {
		logger, err := NewSecureLogger(nil)
		if err != nil {
			t.Fatalf("NewSecureLogger(nil) failed: %v", err)
		}
		defer func() { _ = logger.Close() }()

		if !logger.IsEnabled(INFO) {
			t.Error("Default logger should have INFO level enabled")
		}
	})

	t.Run("Custom config", func(t *testing.T) {
		config := &LogConfig{
			Enabled:     true,
			Level:       DEBUG,
			Format:      JSON,
			MaxFileSize: 1024 * 1024,
			MaxFiles:    3,
			BufferSize:  500,
		}

		logger, err := NewSecureLogger(config)
		if err != nil {
			t.Fatalf("NewSecureLogger() failed: %v", err)
		}
		defer func() { _ = logger.Close() }()

		if !logger.IsEnabled(DEBUG) {
			t.Error("Logger should have DEBUG level enabled")
		}
	})

	t.Run("Disabled logging", func(t *testing.T) {
		config := &LogConfig{
			Enabled:     false,
			MaxFileSize: 1024 * 1024,
			MaxFiles:    3,
			BufferSize:  500,
		}

		logger, err := NewSecureLogger(config)
		if err != nil {
			t.Fatalf("NewSecureLogger() failed: %v", err)
		}
		defer func() { _ = logger.Close() }()

		if logger.IsEnabled(ERROR) {
			t.Error("Disabled logger should not have any level enabled")
		}
	})

	t.Run("Invalid config", func(t *testing.T) {
		config := &LogConfig{
			Enabled:     true,
			MaxFileSize: -1, // Invalid
		}

		_, err := NewSecureLogger(config)
		if err == nil {
			t.Error("NewSecureLogger() should fail with invalid config")
		}
	})

	t.Run("File output", func(t *testing.T) {
		tempDir := t.TempDir()
		logFile := filepath.Join(tempDir, "test.log")

		config := &LogConfig{
			Enabled:     true,
			Level:       INFO,
			OutputFile:  logFile,
			MaxFileSize: 1024 * 1024,
			MaxFiles:    3,
			BufferSize:  500,
		}

		logger, err := NewSecureLogger(config)
		if err != nil {
			t.Fatalf("NewSecureLogger() failed: %v", err)
		}
		defer func() { _ = logger.Close() }()

		// Write a test message
		err = logger.Info("test message")
		if err != nil {
			t.Fatalf("Info() failed: %v", err)
		}

		// Close to flush
		_ = logger.Close()

		// Check file exists and has content
		content, err := os.ReadFile(logFile)
		if err != nil {
			t.Fatalf("Failed to read log file: %v", err)
		}

		if !strings.Contains(string(content), "test message") {
			t.Error("Log file should contain the test message")
		}
	})
}

func TestFileSecureLogger_BasicLogging(t *testing.T) {
	var buf bytes.Buffer
	logger := createTestLogger(&buf, INFO, TEXT)
	defer func() { _ = logger.Close() }()

	tests := []struct {
		name    string
		logFunc func(string, ...LogField) error
		level   string
		message string
		enabled bool
	}{
		{"Error", logger.Error, "ERROR", "error message", true},
		{"Warn", logger.Warn, "WARN", "warn message", true},
		{"Info", logger.Info, "INFO", "info message", true},
		{"Debug", logger.Debug, "DEBUG", "debug message", false}, // DEBUG disabled at INFO level
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()

			err := tt.logFunc(tt.message)
			if err != nil {
				t.Fatalf("%s() failed: %v", tt.name, err)
			}

			output := buf.String()
			if tt.enabled {
				if !strings.Contains(output, tt.level) {
					t.Errorf("Output should contain level %s, got: %s", tt.level, output)
				}
				if !strings.Contains(output, tt.message) {
					t.Errorf("Output should contain message %s, got: %s", tt.message, output)
				}
			} else {
				if output != "" {
					t.Errorf("Output should be empty for disabled level, got: %s", output)
				}
			}
		})
	}
}

func TestFileSecureLogger_LogWalletGenerated(t *testing.T) {
	var buf bytes.Buffer
	logger := createTestLogger(&buf, INFO, TEXT)
	defer func() { _ = logger.Close() }()

	address := "0x1234567890abcdef1234567890abcdef12345678"
	attempts := 42
	duration := 123 * time.Millisecond
	threadID := 5

	err := logger.LogWalletGenerated(address, attempts, duration, threadID)
	if err != nil {
		t.Fatalf("LogWalletGenerated() failed: %v", err)
	}

	output := buf.String()

	// Check that safe data is logged
	if !strings.Contains(output, address) {
		t.Error("Output should contain wallet address")
	}
	if !strings.Contains(output, "42") {
		t.Error("Output should contain attempts count")
	}
	if !strings.Contains(output, "5") {
		t.Error("Output should contain thread ID")
	}
	if !strings.Contains(output, "Wallet generated") {
		t.Error("Output should contain wallet generation message")
	}

	// Verify no sensitive data patterns (this is a basic check)
	// In a real implementation, we'd check for private key patterns
	if strings.Contains(strings.ToLower(output), "private") {
		t.Error("Output should not contain 'private' keyword")
	}
}

func TestFileSecureLogger_LogOperationStart(t *testing.T) {
	var buf bytes.Buffer
	logger := createTestLogger(&buf, INFO, TEXT)
	defer func() { _ = logger.Close() }()

	operation := "wallet_generation"
	params := map[string]interface{}{
		"prefix":      "abc",
		"suffix":      "def",
		"threads":     4,
		"private_key": "should_not_be_logged", // This should be filtered out
	}

	err := logger.LogOperationStart(operation, params)
	if err != nil {
		t.Fatalf("LogOperationStart() failed: %v", err)
	}

	output := buf.String()

	// Check that safe parameters are logged
	if !strings.Contains(output, "abc") {
		t.Error("Output should contain prefix parameter")
	}
	if !strings.Contains(output, "def") {
		t.Error("Output should contain suffix parameter")
	}
	if !strings.Contains(output, "4") {
		t.Error("Output should contain threads parameter")
	}

	// Check that unsafe parameters are not logged
	if strings.Contains(output, "should_not_be_logged") {
		t.Error("Output should not contain private_key parameter")
	}

	// Check operation context
	if !strings.Contains(output, operation) {
		t.Error("Output should contain operation name")
	}
}

func TestFileSecureLogger_LogOperationComplete(t *testing.T) {
	var buf bytes.Buffer
	logger := createTestLogger(&buf, INFO, TEXT)
	defer func() { _ = logger.Close() }()

	operation := "wallet_generation"
	stats := OperationStats{
		Duration:     5 * time.Second,
		Success:      true,
		ItemsCount:   100,
		ErrorCount:   2,
		ThroughputPS: 20.5,
	}

	err := logger.LogOperationComplete(operation, stats)
	if err != nil {
		t.Fatalf("LogOperationComplete() failed: %v", err)
	}

	output := buf.String()

	// Check that stats are logged
	if !strings.Contains(output, "true") {
		t.Error("Output should contain success status")
	}
	if !strings.Contains(output, "100") {
		t.Error("Output should contain items count")
	}
	if !strings.Contains(output, "20.5") {
		t.Error("Output should contain throughput")
	}
	if !strings.Contains(output, operation) {
		t.Error("Output should contain operation name")
	}
}

func TestFileSecureLogger_LogError(t *testing.T) {
	var buf bytes.Buffer
	logger := createTestLogger(&buf, INFO, TEXT)
	defer func() { _ = logger.Close() }()

	operation := "wallet_generation"
	err := errors.New("test error message")
	context := map[string]interface{}{
		"attempts":     42,
		"thread_id":    5,
		"private_data": "should_not_be_logged", // This should be filtered out
	}

	logErr := logger.LogError(operation, err, context)
	if logErr != nil {
		t.Fatalf("LogError() failed: %v", logErr)
	}

	output := buf.String()

	// Check that error is logged
	if !strings.Contains(output, "test error message") {
		t.Error("Output should contain error message")
	}

	// Check that safe context is logged
	if !strings.Contains(output, "42") {
		t.Error("Output should contain attempts from context")
	}

	// Check that unsafe context is not logged
	if strings.Contains(output, "should_not_be_logged") {
		t.Error("Output should not contain private_data from context")
	}

	if !strings.Contains(output, operation) {
		t.Error("Output should contain operation name")
	}
}

func TestFileSecureLogger_SetLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := createTestLogger(&buf, INFO, TEXT)
	defer func() { _ = logger.Close() }()

	// Initially DEBUG should be disabled
	if logger.IsEnabled(DEBUG) {
		t.Error("DEBUG should be disabled initially")
	}

	// Set level to DEBUG
	err := logger.SetLevel(DEBUG)
	if err != nil {
		t.Fatalf("SetLevel() failed: %v", err)
	}

	// Now DEBUG should be enabled
	if !logger.IsEnabled(DEBUG) {
		t.Error("DEBUG should be enabled after SetLevel(DEBUG)")
	}

	// Test that DEBUG logging now works
	buf.Reset()
	err = logger.Debug("debug message")
	if err != nil {
		t.Fatalf("Debug() failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "debug message") {
		t.Error("Debug message should be logged after enabling DEBUG level")
	}
}

func TestFileSecureLogger_IsEnabled(t *testing.T) {
	tests := []struct {
		name        string
		configLevel LogLevel
		testLevel   LogLevel
		expected    bool
	}{
		{"ERROR at ERROR level", ERROR, ERROR, true},
		{"WARN at ERROR level", ERROR, WARN, false},
		{"INFO at ERROR level", ERROR, INFO, false},
		{"DEBUG at ERROR level", ERROR, DEBUG, false},
		{"ERROR at INFO level", INFO, ERROR, true},
		{"WARN at INFO level", INFO, WARN, true},
		{"INFO at INFO level", INFO, INFO, true},
		{"DEBUG at INFO level", INFO, DEBUG, false},
		{"All at DEBUG level", DEBUG, DEBUG, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := createTestLogger(&buf, tt.configLevel, TEXT)
			defer func() { _ = logger.Close() }()

			result := logger.IsEnabled(tt.testLevel)
			if result != tt.expected {
				t.Errorf("IsEnabled(%v) = %v, want %v", tt.testLevel, result, tt.expected)
			}
		})
	}
}

func TestFileSecureLogger_Close(t *testing.T) {
	var buf bytes.Buffer
	logger := createTestLogger(&buf, INFO, TEXT)

	// Should be able to log before closing
	err := logger.Info("before close")
	if err != nil {
		t.Fatalf("Info() before close failed: %v", err)
	}

	// Close the logger
	err = logger.Close()
	if err != nil {
		t.Fatalf("Close() failed: %v", err)
	}

	// Should not be able to log after closing
	err = logger.Info("after close")
	if err == nil {
		t.Error("Info() after close should fail")
	}

	// Multiple closes should be safe
	err = logger.Close()
	if err != nil {
		t.Fatalf("Second Close() failed: %v", err)
	}
}

func TestFileSecureLogger_JSONFormat(t *testing.T) {
	var buf bytes.Buffer
	logger := createTestLogger(&buf, INFO, JSON)
	defer func() { _ = logger.Close() }()

	err := logger.Info("test message", NewLogField("key", "value"))
	if err != nil {
		t.Fatalf("Info() failed: %v", err)
	}

	output := buf.String()

	// Parse as JSON to verify format
	var entry LogEntry
	err = json.Unmarshal([]byte(strings.TrimSpace(output)), &entry)
	if err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	if entry.Level != INFO {
		t.Errorf("JSON entry.Level = %v, want %v", entry.Level, INFO)
	}
	if entry.Message != "test message" {
		t.Errorf("JSON entry.Message = %q, want %q", entry.Message, "test message")
	}
	if entry.Fields["key"] != "value" {
		t.Errorf("JSON entry.Fields[key] = %v, want %v", entry.Fields["key"], "value")
	}
}

func TestFileSecureLogger_StructuredFormat(t *testing.T) {
	var buf bytes.Buffer
	logger := createTestLogger(&buf, INFO, STRUCTURED)
	defer func() { _ = logger.Close() }()

	err := logger.Info("test message", NewLogField("key", "value"))
	if err != nil {
		t.Fatalf("Info() failed: %v", err)
	}

	output := buf.String()

	// Check structured format components
	if !strings.Contains(output, "level=INFO") {
		t.Error("Structured output should contain level=INFO")
	}
	if !strings.Contains(output, `message="test message"`) {
		t.Error("Structured output should contain quoted message")
	}
	if !strings.Contains(output, `key="value"`) {
		t.Error("Structured output should contain quoted field value")
	}
	if !strings.Contains(output, "timestamp=") {
		t.Error("Structured output should contain timestamp")
	}
}

func TestFileSecureLogger_WithFields(t *testing.T) {
	var buf bytes.Buffer
	logger := createTestLogger(&buf, INFO, TEXT)
	defer func() { _ = logger.Close() }()

	fields := []LogField{
		NewLogField("string_field", "string_value"),
		NewLogField("int_field", 42),
		NewLogField("bool_field", true),
		NewLogField("float_field", 3.14),
	}

	err := logger.Info("test with fields", fields...)
	if err != nil {
		t.Fatalf("Info() with fields failed: %v", err)
	}

	output := buf.String()

	// Check that all fields are present
	expectedValues := []string{"string_value", "42", "true", "3.14"}
	for _, expected := range expectedValues {
		if !strings.Contains(output, expected) {
			t.Errorf("Output should contain %s, got: %s", expected, output)
		}
	}
}

func TestIsSafeParameter(t *testing.T) {
	tests := []struct {
		key      string
		expected bool
	}{
		{"prefix", true},
		{"suffix", true},
		{"count", true},
		{"threads", true},
		{"address", true},
		{"private_key", false},
		{"public_key", false},
		{"seed", false},
		{"password", false},
		{"secret", false},
		{"unknown_param", false},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			result := isSafeParameter(tt.key)
			if result != tt.expected {
				t.Errorf("isSafeParameter(%q) = %v, want %v", tt.key, result, tt.expected)
			}
		})
	}
}

func TestSanitizeError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{"Nil error", nil, ""},
		{"Simple error", errors.New("simple error"), "simple error"},
		{"Error with details", errors.New("operation failed: details"), "operation failed: details"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeError(tt.err)
			if result != tt.expected {
				t.Errorf("sanitizeError() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// Helper function to create a test logger with custom writer
func createTestLogger(writer io.Writer, level LogLevel, format LogFormat) *FileSecureLogger {
	config := &LogConfig{
		Enabled: true,
		Level:   level,
		Format:  format,
	}

	logger := &FileSecureLogger{
		config:    config,
		writer:    writer,
		formatter: GetFormatterForFormat(format),
	}

	return logger
}
func TestFileSecureLogger_SetFormatter(t *testing.T) {
	config := DefaultLogConfig()
	config.Level = INFO
	config.Format = TEXT

	logger, err := NewSecureLogger(config)
	if err != nil {
		t.Fatalf("NewSecureLogger() failed: %v", err)
	}
	defer func() { _ = logger.Close() }()

	// Test setting a custom formatter
	customFormatter := NewJSONFormatter()
	err = logger.(*FileSecureLogger).SetFormatter(customFormatter)
	if err != nil {
		t.Fatalf("SetFormatter() failed: %v", err)
	}

	// Test that nil formatter is rejected
	err = logger.(*FileSecureLogger).SetFormatter(nil)
	if err == nil {
		t.Error("SetFormatter(nil) should return an error")
	}
}

func TestFileSecureLogger_FormatterIntegration(t *testing.T) {
	tests := []struct {
		name      string
		format    LogFormat
		formatter LogFormatter
		checkFunc func(string) bool
	}{
		{
			name:      "JSON formatter",
			format:    JSON,
			formatter: NewJSONFormatter(),
			checkFunc: func(output string) bool {
				var parsed map[string]interface{}
				return json.Unmarshal([]byte(strings.TrimSpace(output)), &parsed) == nil
			},
		},
		{
			name:      "Text formatter",
			format:    TEXT,
			formatter: NewTextFormatter(),
			checkFunc: func(output string) bool {
				return strings.Contains(output, "[") && strings.Contains(output, "]") &&
					strings.Contains(output, "INFO:")
			},
		},
		{
			name:      "Structured formatter",
			format:    STRUCTURED,
			formatter: NewStructuredFormatter(),
			checkFunc: func(output string) bool {
				return strings.Contains(output, "level=INFO") &&
					strings.Contains(output, "message=")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf strings.Builder
			config := &LogConfig{
				Enabled: true,
				Level:   INFO,
				Format:  tt.format,
			}

			logger := &FileSecureLogger{
				config:    config,
				writer:    &buf,
				formatter: tt.formatter,
			}

			err := logger.Info("test message", NewLogField("key", "value"))
			if err != nil {
				t.Fatalf("Info() failed: %v", err)
			}

			output := buf.String()
			if !tt.checkFunc(output) {
				t.Errorf("Output format check failed for %s: %s", tt.name, output)
			}
		})
	}
}

func TestFileSecureLogger_CustomFormatterUsage(t *testing.T) {
	var buf strings.Builder
	config := &LogConfig{
		Enabled: true,
		Level:   DEBUG,
		Format:  TEXT, // Will be overridden by custom formatter
	}

	logger := &FileSecureLogger{
		config:    config,
		writer:    &buf,
		formatter: NewJSONFormatter(), // Use JSON formatter instead of TEXT
	}

	// Log a complex entry
	err := logger.LogWalletGenerated("0x1234567890abcdef", 42, time.Millisecond*150, 3)
	if err != nil {
		t.Fatalf("LogWalletGenerated() failed: %v", err)
	}

	output := buf.String()

	// Should be JSON format despite config saying TEXT
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &parsed); err != nil {
		t.Fatalf("Output should be JSON: %v", err)
	}

	// Verify content
	if parsed["level"] != "INFO" {
		t.Errorf("Expected level 'INFO', got %v", parsed["level"])
	}
	if parsed["message"] != "Wallet generated" {
		t.Errorf("Expected message 'Wallet generated', got %v", parsed["message"])
	}

	// Check fields
	fields, ok := parsed["fields"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected fields to be a map")
	}
	if fields["address"] != "0x1234567890abcdef" {
		t.Errorf("Expected address '0x1234567890abcdef', got %v", fields["address"])
	}
	if fields["attempts"] != float64(42) { // JSON numbers are float64
		t.Errorf("Expected attempts 42, got %v", fields["attempts"])
	}
}

// Tests for Task 4: Error sanitization and safe logging methods

func TestErrorCategory(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected ErrorCategory
	}{
		{"Crypto error - key", errors.New("invalid private key format"), ErrorCrypto},
		{"Crypto error - encrypt", errors.New("encryption failed"), ErrorCrypto},
		{"Crypto error - secp256k1", errors.New("secp256k1 signature error"), ErrorCrypto},
		{"Validation error - invalid", errors.New("invalid input format"), ErrorValidation},
		{"Validation error - parse", errors.New("failed to parse address"), ErrorValidation},
		{"IO error - file", errors.New("file not found"), ErrorIO},
		{"IO error - permission", errors.New("permission denied"), ErrorIO},
		{"Network error - connection", errors.New("connection timeout"), ErrorNetwork},
		{"Network error - dns", errors.New("dns resolution failed"), ErrorNetwork},
		{"System error - memory", errors.New("out of memory"), ErrorSystem},
		{"Unknown error", errors.New("something unexpected"), ErrorUnknown},
		{"Nil error", nil, ErrorUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := categorizeError(tt.err)
			if result != tt.expected {
				t.Errorf("categorizeError(%v) = %v, want %v", tt.err, result, tt.expected)
			}
		})
	}
}

func TestSanitizeError_SensitiveDataRemoval(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		contains []string // Strings that should NOT be in output
		missing  []string // Strings that should be in output (redacted versions)
	}{
		{
			name:     "Private key in error",
			err:      errors.New("invalid private key: 0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"),
			contains: []string{"[PRIVATE_KEY_REDACTED]"},
			missing:  []string{"1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"},
		},
		{
			name:     "Public key in error",
			err:      errors.New("public key validation failed: 0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"),
			contains: []string{"[PUBLIC_KEY_REDACTED]"},
			missing:  []string{"1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"},
		},
		{
			name:     "Password in URL",
			err:      errors.New("connection failed to https://user:secret123@example.com/api"),
			contains: []string{"[REDACTED]"},
			missing:  []string{"secret123"},
		},
		{
			name:     "Multiple sensitive patterns",
			err:      errors.New("crypto error with key 0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef and password=mysecret"),
			contains: []string{"[PRIVATE_KEY_REDACTED]", "password=[REDACTED]"},
			missing:  []string{"1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef", "mysecret"},
		},
		{
			name:     "Safe error message",
			err:      errors.New("validation failed: invalid address format"),
			contains: []string{"validation failed: invalid address format"},
			missing:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeError(tt.err)

			// Check that expected strings are present
			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("sanitizeError() result should contain %q, got: %q", expected, result)
				}
			}

			// Check that sensitive strings are removed
			for _, sensitive := range tt.missing {
				if strings.Contains(result, sensitive) {
					t.Errorf("sanitizeError() result should not contain %q, got: %q", sensitive, result)
				}
			}
		})
	}
}

func TestSanitizeParameterValue(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		value    interface{}
		expected interface{}
	}{
		{"Safe string value", "prefix", "abc", "abc"},
		{"Safe integer value", "count", 42, 42},
		{"Safe boolean value", "success", true, true},
		{"Safe float value", "throughput", 3.14, 3.14},
		{"String with private key", "message", "error: key 0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef", "error: key [PRIVATE_KEY_REDACTED]"},
		{"Path sanitization", "path", "/home/user/keystore/wallet.json", "/home./[REDACTED]"}, // Path with keystore should be sanitized
		{"URL with password", "url", "https://user:pass123@example.com", "https://user:[REDACTED]@example.com"},
		{"Non-string complex type", "data", map[string]int{"count": 5}, "map[count:5]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeParameterValue(tt.key, tt.value)

			// For string comparisons
			if expectedStr, ok := tt.expected.(string); ok {
				if resultStr, ok := result.(string); ok {
					if resultStr != expectedStr {
						t.Errorf("sanitizeParameterValue(%q, %v) = %q, want %q", tt.key, tt.value, resultStr, expectedStr)
					}
				} else {
					t.Errorf("sanitizeParameterValue(%q, %v) returned non-string %v, want string %q", tt.key, tt.value, result, expectedStr)
				}
			} else {
				// For non-string comparisons
				if result != tt.expected {
					t.Errorf("sanitizeParameterValue(%q, %v) = %v, want %v", tt.key, tt.value, result, tt.expected)
				}
			}
		})
	}
}

func TestIsSafeParameter_Enhanced(t *testing.T) {
	tests := []struct {
		key      string
		expected bool
	}{
		// Explicitly safe parameters
		{"prefix", true},
		{"suffix", true},
		{"count", true},
		{"threads", true},
		{"address", true},
		{"attempts", true},
		{"duration", true},
		{"thread_id", true},
		{"error_category", true},
		{"throughput_per_second", true},

		// Explicitly unsafe parameters
		{"private_key", false},
		{"privatekey", false},
		{"public_key", false},
		{"publickey", false},
		{"seed", false},
		{"mnemonic", false},
		{"password", false},
		{"pwd", false},
		{"pass", false},
		{"secret", false},
		{"token", false},
		{"auth", false},
		{"credential", false},
		{"key", false},
		{"keystore", false},

		// Parameters with unsafe patterns
		{"user_password", false},
		{"api_key", false},
		{"secret_data", false},
		{"private_info", false},
		{"auth_token", false},

		// Case insensitive checks
		{"PRIVATE_KEY", false},
		{"Password", false},
		{"SECRET", false},

		// Unknown parameters (conservative approach)
		{"unknown_param", false},
		{"custom_field", false},
		{"random_data", false},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			result := isSafeParameter(tt.key)
			if result != tt.expected {
				t.Errorf("isSafeParameter(%q) = %v, want %v", tt.key, result, tt.expected)
			}
		})
	}
}

func TestFileSecureLogger_LogError_Enhanced(t *testing.T) {
	var buf bytes.Buffer
	logger := createTestLogger(&buf, INFO, JSON)
	defer func() { _ = logger.Close() }()

	// Test error with sensitive data
	operation := "wallet_generation"
	sensitiveErr := errors.New("crypto error: private key 0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef is invalid")
	context := map[string]interface{}{
		"attempts":    42,
		"thread_id":   5,
		"private_key": "should_not_be_logged",
		"safe_param":  "should_be_logged",
	}

	err := logger.LogError(operation, sensitiveErr, context)
	if err != nil {
		t.Fatalf("LogError() failed: %v", err)
	}

	output := buf.String()

	// Parse JSON output
	var logEntry map[string]interface{}
	err = json.Unmarshal([]byte(strings.TrimSpace(output)), &logEntry)
	if err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	// Check that error message is sanitized
	errorMessage, ok := logEntry["fields"].(map[string]interface{})["error_message"].(string)
	if !ok {
		t.Fatal("Expected error_message field in log entry")
	}

	if strings.Contains(errorMessage, "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef") {
		t.Error("Error message should not contain private key")
	}

	if !strings.Contains(errorMessage, "[PRIVATE_KEY_REDACTED]") {
		t.Error("Error message should contain redacted private key placeholder")
	}

	// Check that error category is present
	errorCategory, ok := logEntry["fields"].(map[string]interface{})["error_category"].(string)
	if !ok {
		t.Fatal("Expected error_category field in log entry")
	}

	if errorCategory != string(ErrorCrypto) {
		t.Errorf("Expected error category %s, got %s", ErrorCrypto, errorCategory)
	}

	// Check that safe context is logged
	fields := logEntry["fields"].(map[string]interface{})
	if fields["attempts"] != float64(42) {
		t.Error("Safe parameter 'attempts' should be logged")
	}

	// Check that unsafe context is not logged
	if _, exists := fields["private_key"]; exists {
		t.Error("Unsafe parameter 'private_key' should not be logged")
	}

	// Check that unknown parameters are not logged (conservative approach)
	if _, exists := fields["safe_param"]; exists {
		t.Error("Unknown parameter 'safe_param' should not be logged (conservative approach)")
	}
}

func TestFileSecureLogger_LogOperationStart_Enhanced(t *testing.T) {
	var buf bytes.Buffer
	logger := createTestLogger(&buf, INFO, JSON)
	defer func() { _ = logger.Close() }()

	operation := "wallet_generation"
	params := map[string]interface{}{
		"prefix":           "abc",
		"suffix":           "def",
		"threads":          4,
		"private_key":      "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
		"password":         "secret123",
		"message_with_key": "error: key 0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
		"safe_url":         "https://example.com/api",
		"unsafe_url":       "https://user:pass123@example.com/api",
	}

	err := logger.LogOperationStart(operation, params)
	if err != nil {
		t.Fatalf("LogOperationStart() failed: %v", err)
	}

	output := buf.String()

	// Parse JSON output
	var logEntry map[string]interface{}
	err = json.Unmarshal([]byte(strings.TrimSpace(output)), &logEntry)
	if err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	fields := logEntry["fields"].(map[string]interface{})

	// Check that safe parameters are logged
	if fields["prefix"] != "abc" {
		t.Error("Safe parameter 'prefix' should be logged")
	}
	if fields["suffix"] != "def" {
		t.Error("Safe parameter 'suffix' should be logged")
	}
	if fields["threads"] != float64(4) {
		t.Error("Safe parameter 'threads' should be logged")
	}

	// Check that unsafe parameters are not logged
	if _, exists := fields["private_key"]; exists {
		t.Error("Unsafe parameter 'private_key' should not be logged")
	}
	if _, exists := fields["password"]; exists {
		t.Error("Unsafe parameter 'password' should not be logged")
	}

	// Check that unknown parameters are not logged (conservative approach)
	if _, exists := fields["message_with_key"]; exists {
		t.Error("Unknown parameter 'message_with_key' should not be logged")
	}
	if _, exists := fields["safe_url"]; exists {
		t.Error("Unknown parameter 'safe_url' should not be logged")
	}
	if _, exists := fields["unsafe_url"]; exists {
		t.Error("Unknown parameter 'unsafe_url' should not be logged")
	}
}

func TestFileSecureLogger_LogOperationComplete_Enhanced(t *testing.T) {
	var buf bytes.Buffer
	logger := createTestLogger(&buf, INFO, JSON)
	defer func() { _ = logger.Close() }()

	operation := "wallet_generation"

	// Test with zero values to ensure they're not logged
	statsWithZeros := OperationStats{
		Duration:     5 * time.Second,
		Success:      true,
		ItemsCount:   0, // Should not be logged
		ErrorCount:   0, // Should not be logged
		ThroughputPS: 0, // Should not be logged
	}

	err := logger.LogOperationComplete(operation, statsWithZeros)
	if err != nil {
		t.Fatalf("LogOperationComplete() failed: %v", err)
	}

	output := buf.String()

	// Parse JSON output
	var logEntry map[string]interface{}
	err = json.Unmarshal([]byte(strings.TrimSpace(output)), &logEntry)
	if err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	fields := logEntry["fields"].(map[string]interface{})

	// Check that required fields are present
	if fields["status"] != "completed" {
		t.Error("Status should be 'completed'")
	}
	if fields["success"] != true {
		t.Error("Success should be true")
	}
	if fields["duration_ns"] == nil {
		t.Error("Duration should be present")
	}

	// Check that zero values are not logged
	if _, exists := fields["items_count"]; exists {
		t.Error("Zero items_count should not be logged")
	}
	if _, exists := fields["error_count"]; exists {
		t.Error("Zero error_count should not be logged")
	}
	if _, exists := fields["throughput_per_second"]; exists {
		t.Error("Zero throughput_per_second should not be logged")
	}

	// Test with non-zero values
	buf.Reset()
	statsWithValues := OperationStats{
		Duration:     3 * time.Second,
		Success:      false,
		ItemsCount:   100,
		ErrorCount:   5,
		ThroughputPS: 33.33,
	}

	err = logger.LogOperationComplete(operation, statsWithValues)
	if err != nil {
		t.Fatalf("LogOperationComplete() with values failed: %v", err)
	}

	output = buf.String()
	err = json.Unmarshal([]byte(strings.TrimSpace(output)), &logEntry)
	if err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	fields = logEntry["fields"].(map[string]interface{})

	// Check that non-zero values are logged
	if fields["items_count"] != float64(100) {
		t.Error("Non-zero items_count should be logged")
	}
	if fields["error_count"] != float64(5) {
		t.Error("Non-zero error_count should be logged")
	}
	if fields["throughput_per_second"] != 33.33 {
		t.Error("Non-zero throughput_per_second should be logged")
	}
}

func TestSensitiveDataDetection_ComprehensivePatterns(t *testing.T) {
	// This test ensures no sensitive data patterns leak through our sanitization
	sensitiveInputs := []struct {
		name  string
		input string
	}{
		{"Private key with 0x", "private key: 0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"},
		{"Private key without 0x", "key 1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef found"},
		{"Public key", "public key 0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"},
		{"Password in URL", "failed to connect to https://user:mypassword123@api.example.com"},
		{"Multiple keys", "keys: 0x1111111111111111111111111111111111111111111111111111111111111111 and 0x2222222222222222222222222222222222222222222222222222222222222222"},
	}

	for _, tt := range sensitiveInputs {
		t.Run(tt.name, func(t *testing.T) {
			err := errors.New(tt.input)
			sanitized := sanitizeError(err)

			// Check for common sensitive patterns that should be redacted
			sensitivePatterns := []string{
				"1234567890abcdef",
				"mypassword123",
				"1111111111111111",
				"2222222222222222",
			}

			for _, pattern := range sensitivePatterns {
				if strings.Contains(strings.ToLower(sanitized), strings.ToLower(pattern)) {
					t.Errorf("Sanitized output still contains sensitive pattern %q: %s", pattern, sanitized)
				}
			}

			// Check that redaction placeholders are present
			redactionPatterns := []string{
				"[PRIVATE_KEY_REDACTED]",
				"[PUBLIC_KEY_REDACTED]",
				"[REDACTED]",
			}

			hasRedaction := false
			for _, pattern := range redactionPatterns {
				if strings.Contains(sanitized, pattern) {
					hasRedaction = true
					break
				}
			}

			if !hasRedaction {
				t.Errorf("Sanitized output should contain redaction placeholder: %s", sanitized)
			}
		})
	}
}

func TestLogError_SecurityAudit(t *testing.T) {
	// This test performs a security audit to ensure no sensitive data leaks
	var buf bytes.Buffer
	logger := createTestLogger(&buf, INFO, TEXT)
	defer func() { _ = logger.Close() }()

	// Create errors with various sensitive data patterns
	sensitiveErrors := []error{
		errors.New("private key 0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890 is invalid"),
		errors.New("seed phrase: abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"),
		errors.New("connection failed: https://user:secretpass@wallet.example.com/api"),
		errors.New("keystore file /home/user/.ethereum/keystore/UTC--2023-01-01T00-00-00.000000000Z--abcdef1234567890abcdef1234567890abcdef12 corrupted"),
	}

	for i, err := range sensitiveErrors {
		buf.Reset()

		logErr := logger.LogError("test_operation", err, map[string]interface{}{
			"attempt": i + 1,
		})
		if logErr != nil {
			t.Fatalf("LogError() failed: %v", logErr)
		}

		output := buf.String()

		// Security audit: check for sensitive patterns
		forbiddenPatterns := []string{
			"abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
			"abandon abandon abandon",
			"secretpass",
			"UTC--2023-01-01T00-00-00.000000000Z--abcdef1234567890abcdef1234567890abcdef12",
		}

		for _, pattern := range forbiddenPatterns {
			if strings.Contains(strings.ToLower(output), strings.ToLower(pattern)) {
				t.Errorf("Log output contains forbidden sensitive pattern %q in test %d: %s", pattern, i+1, output)
			}
		}

		// Ensure the log still contains useful information
		if !strings.Contains(output, "test_operation") {
			t.Errorf("Log output should contain operation name in test %d", i+1)
		}
		if !strings.Contains(output, "Operation failed") {
			t.Errorf("Log output should contain failure message in test %d", i+1)
		}
	}
}

func TestSanitizeFilePathsCanonicalWalletFilename(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		blocked string
	}{
		{"unix_json", "failed /tmp/out/0x1234567890abcdef1234567890abcdef12345678.json now", "1234567890abcdef1234567890abcdef12345678.json"},
		{"unix_pwd", "failed /tmp/out/0x1234567890abcdef1234567890abcdef12345678.pwd now", "1234567890abcdef1234567890abcdef12345678.pwd"},
		{"unix_mnemonic", "failed /tmp/out/0x1234567890abcdef1234567890abcdef12345678.mnemonic now", "1234567890abcdef1234567890abcdef12345678.mnemonic"},
		{"unix_key", "failed /tmp/out/0x1234567890abcdef1234567890abcdef12345678.key now", "1234567890abcdef1234567890abcdef12345678.key"},
		{"windows_json", `failed C:\wallets\0x1234567890abcdef1234567890abcdef12345678.json now`, "1234567890abcdef1234567890abcdef12345678.json"},
		{"basename_only", "failed 0x1234567890abcdef1234567890abcdef12345678.pwd now", "1234567890abcdef1234567890abcdef12345678.pwd"},
		{"checksum_case", "failed /tmp/out/0xAbCdEf1234567890ABCDEF1234567890aBcDeF12.json now", "AbCdEf1234567890ABCDEF1234567890aBcDeF12.json"},
		{"no_prefix", "failed /tmp/out/1234567890abcdef1234567890abcdef12345678.mnemonic now", "1234567890abcdef1234567890abcdef12345678.mnemonic"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := sanitizeFilePaths(tc.input)
			if strings.Contains(got, tc.blocked) {
				t.Errorf("canonical filename leaked: %s", got)
			}
			if !strings.Contains(got, "[KEYSTORE_FILE_REDACTED]") {
				t.Errorf("expected redaction placeholder: %s", got)
			}
		})
	}

	if got := sanitizeFilePaths("writing /tmp/out/config.json"); !strings.Contains(got, "config.json") {
		t.Errorf("ordinary config.json must remain: %s", got)
	}
	legacy := "UTC--2023-01-01T00-00-00.000000000Z--abcdef1234567890abcdef1234567890abcdef12"
	if got := sanitizeFilePaths("keystore " + legacy); strings.Contains(got, legacy) {
		t.Errorf("legacy UTC filename must remain redacted: %s", got)
	}
}
