package logging

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// SecureLogger defines the interface for secure logging operations
type SecureLogger interface {
	// Basic logging methods with structured fields
	Error(msg string, fields ...LogField) error
	Warn(msg string, fields ...LogField) error
	Info(msg string, fields ...LogField) error
	Debug(msg string, fields ...LogField) error

	// Specialized logging methods for wallet operations
	LogWalletGenerated(address string, attempts int, duration time.Duration, threadID int) error
	LogOperationStart(operation string, params map[string]interface{}) error
	LogOperationComplete(operation string, stats OperationStats) error
	LogError(operation string, err error, context map[string]interface{}) error

	// Performance metrics logging methods
	LogWorkerStartup(threadID int, config map[string]interface{}) error
	LogPerformanceMetrics(metrics PerformanceMetrics) error
	LogResourceUsage(cpuPercent float64, memoryBytes int64, threadCount int) error

	// Logger management
	Close() error
	SetLevel(level LogLevel) error
	IsEnabled(level LogLevel) bool
	Flush() error
}

// ErrorCategory represents different types of errors for categorization
type ErrorCategory string

const (
	// ErrorCrypto represents cryptographic operation errors
	ErrorCrypto ErrorCategory = "crypto"
	// ErrorValidation represents input validation errors
	ErrorValidation ErrorCategory = "validation"
	// ErrorIO represents input/output operation errors
	ErrorIO ErrorCategory = "io"
	// ErrorNetwork represents network operation errors
	ErrorNetwork ErrorCategory = "network"
	// ErrorSystem represents system-level errors
	ErrorSystem ErrorCategory = "system"
	// ErrorUnknown represents uncategorized errors
	ErrorUnknown ErrorCategory = "unknown"
)

// OperationStats represents statistics for completed operations
type OperationStats struct {
	Duration     time.Duration `json:"duration_ns"`
	Success      bool          `json:"success"`
	ItemsCount   int64         `json:"items_count,omitempty"`
	ErrorCount   int64         `json:"error_count,omitempty"`
	ThroughputPS float64       `json:"throughput_per_second,omitempty"`
}

// PerformanceMetrics represents system performance metrics for logging
type PerformanceMetrics struct {
	// Throughput metrics
	WalletsPerSecond float64 `json:"wallets_per_second"`
	TotalWallets     int64   `json:"total_wallets"`
	TotalAttempts    int64   `json:"total_attempts"`

	// Timing metrics
	AverageDuration float64 `json:"avg_duration_ms"`
	MinDuration     float64 `json:"min_duration_ms,omitempty"`
	MaxDuration     float64 `json:"max_duration_ms,omitempty"`

	// System metrics
	ThreadCount int     `json:"thread_count"`
	CPUUsage    float64 `json:"cpu_usage_percent,omitempty"`
	MemoryUsage int64   `json:"memory_usage_bytes,omitempty"`

	// Operational metrics
	SuccessRate float64 `json:"success_rate_percent,omitempty"`
	ErrorRate   float64 `json:"error_rate_percent,omitempty"`

	// Time window for metrics
	WindowStart    time.Time     `json:"window_start"`
	WindowDuration time.Duration `json:"window_duration_ns"`
}

// FileSecureLogger implements SecureLogger with thread-safe file operations
type FileSecureLogger struct {
	config       *LogConfig
	writer       io.Writer
	file         *os.File
	formatter    LogFormatter
	mutex        sync.RWMutex
	closed       bool
	currentSize  int64
	buffer       chan *LogEntry
	bufferWriter *bufio.Writer
	flushTicker  *time.Ticker
	flushChan    chan chan struct{}
	done         chan struct{}
	wg           sync.WaitGroup
}

// NewSecureLoggerFromConfig creates a new SecureLogger from application configuration
func NewSecureLoggerFromConfig(enabled bool, level, format, outputFile string) (SecureLogger, error) {
	if !enabled {
		// Return a no-op logger when logging is disabled
		return &FileSecureLogger{
			config: &LogConfig{Enabled: false},
			writer: io.Discard,
		}, nil
	}

	// Parse log level
	logLevel, err := ParseLogLevel(level)
	if err != nil {
		return nil, fmt.Errorf("invalid log level %s: %w", level, err)
	}

	// Parse log format
	var logFormat LogFormat
	switch strings.ToLower(format) {
	case "json":
		logFormat = JSON
	case "structured":
		logFormat = STRUCTURED
	case "text":
		logFormat = TEXT
	default:
		return nil, fmt.Errorf("invalid log format: %s", format)
	}

	// Create configuration
	config := &LogConfig{
		Enabled:     true,
		Level:       logLevel,
		Format:      logFormat,
		OutputFile:  outputFile,
		MaxFileSize: 10 * 1024 * 1024, // 10MB
		MaxFiles:    5,
		BufferSize:  1000,
	}

	return NewSecureLogger(config)
}

// NewSecureLogger creates a new SecureLogger with the given configuration
func NewSecureLogger(config *LogConfig) (SecureLogger, error) {
	if config == nil {
		config = DefaultLogConfig()
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	logger := &FileSecureLogger{
		config:    config,
		formatter: GetFormatterForFormat(config.Format),
	}

	// If logging is disabled, return a no-op logger
	if !config.Enabled {
		logger.writer = io.Discard
		return logger, nil
	}

	// Set up the writer and initialize buffering if needed
	if config.OutputFile == "" {
		// Check if we should avoid stdout (e.g., when TUI is active)
		if shouldAvoidStdout() {
			// Create a temporary log file to avoid interfering with TUI
			tempLogFile := "/tmp/bloco-vgen.log"
			if err := logger.initializeFileWriterWithPath(tempLogFile); err != nil {
				// If temp file fails, use discard to avoid TUI interference
				logger.writer = io.Discard
			} else {
				// Initialize async buffering for file output
				if config.BufferSize > 0 {
					logger.initializeAsyncBuffering()
				}
			}
		} else {
			logger.writer = os.Stdout
			// No buffering for stdout
		}
	} else {
		if err := logger.initializeFileWriter(); err != nil {
			// Fallback behavior depends on TUI state
			if shouldAvoidStdout() {
				fmt.Fprintf(os.Stderr, "Warning: Failed to open log file %s: %v. Disabling logging to avoid TUI interference.\n", config.OutputFile, err)
				logger.writer = io.Discard
			} else {
				fmt.Fprintf(os.Stderr, "Warning: Failed to open log file %s: %v. Falling back to stdout.\n", config.OutputFile, err)
				logger.writer = os.Stdout
			}
		} else {
			// Initialize async buffering for file output
			if config.BufferSize > 0 {
				logger.initializeAsyncBuffering()
			}
		}
	}

	return logger, nil
}

// Error logs an error message with optional fields
func (l *FileSecureLogger) Error(msg string, fields ...LogField) error {
	return l.log(ERROR, msg, fields...)
}

// Warn logs a warning message with optional fields
func (l *FileSecureLogger) Warn(msg string, fields ...LogField) error {
	return l.log(WARN, msg, fields...)
}

// Info logs an info message with optional fields
func (l *FileSecureLogger) Info(msg string, fields ...LogField) error {
	return l.log(INFO, msg, fields...)
}

// Debug logs a debug message with optional fields
func (l *FileSecureLogger) Debug(msg string, fields ...LogField) error {
	return l.log(DEBUG, msg, fields...)
}

// LogWalletGenerated logs wallet generation with only safe data
func (l *FileSecureLogger) LogWalletGenerated(address string, attempts int, duration time.Duration, threadID int) error {
	return l.log(INFO, "Wallet generated",
		NewLogField("address", address),
		NewLogField("attempts", attempts),
		NewLogField("duration_ns", duration.Nanoseconds()),
		NewLogField("thread_id", threadID),
	)
}

// LogOperationStart logs the start of an operation with sanitized parameters
func (l *FileSecureLogger) LogOperationStart(operation string, params map[string]interface{}) error {
	fields := []LogField{
		NewLogField("status", "started"),
	}

	// Add sanitized parameters
	for key, value := range params {
		// Only log safe parameters, skip sensitive data
		if isSafeParameter(key) {
			sanitizedValue := sanitizeParameterValue(key, value)
			fields = append(fields, NewLogField(key, sanitizedValue))
		}
	}

	return l.logWithOperation(INFO, "Operation started", operation, fields...)
}

// LogOperationComplete logs the completion of an operation with statistics
func (l *FileSecureLogger) LogOperationComplete(operation string, stats OperationStats) error {
	fields := []LogField{
		NewLogField("status", "completed"),
		NewLogField("success", stats.Success),
		NewLogField("duration_ns", stats.Duration.Nanoseconds()),
	}

	// Only include non-zero statistics to keep logs clean
	if stats.ItemsCount > 0 {
		fields = append(fields, NewLogField("items_count", stats.ItemsCount))
	}
	if stats.ErrorCount > 0 {
		fields = append(fields, NewLogField("error_count", stats.ErrorCount))
	}
	if stats.ThroughputPS > 0 {
		fields = append(fields, NewLogField("throughput_per_second", stats.ThroughputPS))
	}

	return l.logWithOperation(INFO, "Operation completed", operation, fields...)
}

// LogError logs an error with sanitized context and error categorization
func (l *FileSecureLogger) LogError(operation string, err error, context map[string]interface{}) error {
	sanitizedError := sanitizeError(err)
	category := categorizeError(err)

	fields := []LogField{
		NewLogField("error_message", sanitizedError),
		NewLogField("error_category", string(category)),
	}

	// Add sanitized context
	for key, value := range context {
		if isSafeParameter(key) {
			sanitizedValue := sanitizeParameterValue(key, value)
			fields = append(fields, NewLogField(key, sanitizedValue))
		}
	}

	return l.logWithOperation(ERROR, "Operation failed", operation, fields...)
}

// LogWorkerStartup logs worker thread startup with configuration
func (l *FileSecureLogger) LogWorkerStartup(threadID int, config map[string]interface{}) error {
	fields := []LogField{
		NewLogField("thread_id", threadID),
		NewLogField("status", "started"),
	}

	// Add sanitized configuration parameters
	for key, value := range config {
		if isSafeParameter(key) {
			sanitizedValue := sanitizeParameterValue(key, value)
			fields = append(fields, NewLogField(key, sanitizedValue))
		}
	}

	return l.log(INFO, "Worker thread started", fields...)
}

// LogPerformanceMetrics logs system performance metrics safely
func (l *FileSecureLogger) LogPerformanceMetrics(metrics PerformanceMetrics) error {
	fields := []LogField{
		NewLogField("wallets_per_second", metrics.WalletsPerSecond),
		NewLogField("total_wallets", metrics.TotalWallets),
		NewLogField("total_attempts", metrics.TotalAttempts),
		NewLogField("avg_duration_ms", metrics.AverageDuration),
		NewLogField("thread_count", metrics.ThreadCount),
		NewLogField("window_start", metrics.WindowStart.Format(time.RFC3339)),
		NewLogField("window_duration_ns", metrics.WindowDuration.Nanoseconds()),
	}

	// Add optional metrics only if they have meaningful values
	if metrics.MinDuration > 0 {
		fields = append(fields, NewLogField("min_duration_ms", metrics.MinDuration))
	}
	if metrics.MaxDuration > 0 {
		fields = append(fields, NewLogField("max_duration_ms", metrics.MaxDuration))
	}
	if metrics.CPUUsage > 0 {
		fields = append(fields, NewLogField("cpu_usage_percent", metrics.CPUUsage))
	}
	if metrics.MemoryUsage > 0 {
		fields = append(fields, NewLogField("memory_usage_bytes", metrics.MemoryUsage))
	}
	if metrics.SuccessRate >= 0 {
		fields = append(fields, NewLogField("success_rate_percent", metrics.SuccessRate))
	}
	if metrics.ErrorRate >= 0 {
		fields = append(fields, NewLogField("error_rate_percent", metrics.ErrorRate))
	}

	return l.log(INFO, "Performance metrics", fields...)
}

// LogResourceUsage logs current system resource usage
func (l *FileSecureLogger) LogResourceUsage(cpuPercent float64, memoryBytes int64, threadCount int) error {
	fields := []LogField{
		NewLogField("thread_count", threadCount),
	}

	// Only log resource metrics if they are available and meaningful
	if cpuPercent >= 0 {
		fields = append(fields, NewLogField("cpu_usage_percent", cpuPercent))
	}
	if memoryBytes > 0 {
		fields = append(fields, NewLogField("memory_usage_bytes", memoryBytes))
	}

	return l.log(DEBUG, "Resource usage", fields...)
}

// Close closes the logger and any associated resources
func (l *FileSecureLogger) Close() error {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if l.closed {
		return nil
	}

	l.closed = true

	// Stop async buffering if it's running
	if l.done != nil {
		close(l.done)
		l.wg.Wait() // Wait for buffer worker to finish
	}

	// Stop flush ticker
	if l.flushTicker != nil {
		l.flushTicker.Stop()
	}

	// Close buffer channel
	if l.buffer != nil {
		close(l.buffer)
	}

	// Flush any remaining buffered data
	if l.bufferWriter != nil {
		if err := l.bufferWriter.Flush(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to flush buffer during close: %v\n", err)
		}
	}

	// Close the file
	if l.file != nil {
		return l.file.Close()
	}

	return nil
}

// SetLevel sets the minimum log level
func (l *FileSecureLogger) SetLevel(level LogLevel) error {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	l.config.Level = level
	return nil
}

// SetFormatter sets the log formatter
func (l *FileSecureLogger) SetFormatter(formatter LogFormatter) error {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if formatter == nil {
		return fmt.Errorf("formatter cannot be nil")
	}

	l.formatter = formatter
	return nil
}

// IsEnabled checks if a log level is enabled
func (l *FileSecureLogger) IsEnabled(level LogLevel) bool {
	l.mutex.RLock()
	defer l.mutex.RUnlock()

	return l.config.Enabled && level <= l.config.Level
}

// Flush flushes any buffered log entries to disk
func (l *FileSecureLogger) Flush() error {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if l.closed {
		return fmt.Errorf("logger is closed")
	}

	// If async buffering is enabled, request a flush and wait for completion
	if l.buffer != nil && l.flushChan != nil {
		responseChan := make(chan struct{})
		select {
		case l.flushChan <- responseChan:
			// Wait for flush to complete
			<-responseChan
		default:
			// Flush channel is busy, skip async flush
		}
	}

	// Flush buffered writer if it exists
	if l.bufferWriter != nil {
		if err := l.bufferWriter.Flush(); err != nil {
			return err
		}
	}

	// If using file output, sync to disk
	if l.file != nil {
		return l.file.Sync()
	}

	return nil
}

// log is the internal logging method
func (l *FileSecureLogger) log(level LogLevel, msg string, fields ...LogField) error {
	if !l.IsEnabled(level) {
		return nil
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()

	if l.closed {
		return fmt.Errorf("logger is closed")
	}

	entry := NewLogEntry(level, msg).WithFields(fields...)
	return l.writeEntry(entry)
}

// logWithOperation logs with operation context
func (l *FileSecureLogger) logWithOperation(level LogLevel, msg string, operation string, fields ...LogField) error {
	if !l.IsEnabled(level) {
		return nil
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()

	if l.closed {
		return fmt.Errorf("logger is closed")
	}

	entry := NewLogEntry(level, msg).WithOperation(operation).WithFields(fields...)
	return l.writeEntry(entry)
}

// writeEntry writes a log entry to the output
func (l *FileSecureLogger) writeEntry(entry *LogEntry) error {
	// If async buffering is enabled, send to buffer channel
	if l.buffer != nil {
		select {
		case l.buffer <- entry:
			return nil
		default:
			// Buffer is full, write synchronously as fallback
			return l.writeEntrySync(entry)
		}
	}

	// Synchronous write
	return l.writeEntrySync(entry)
}

// writeEntrySync writes a log entry synchronously
func (l *FileSecureLogger) writeEntrySync(entry *LogEntry) error {
	output, err := l.formatter.Format(entry)
	if err != nil {
		return fmt.Errorf("failed to format log entry: %w", err)
	}

	outputBytes := []byte(output)

	// Check if rotation is needed before writing
	if l.file != nil && l.config.MaxFileSize > 0 {
		if err := l.checkAndRotateFile(int64(len(outputBytes))); err != nil {
			// Log rotation failed, but continue with write
			fmt.Fprintf(os.Stderr, "Warning: Log rotation failed: %v\n", err)
		}
	}

	// Write to the appropriate writer
	writer := l.writer
	if l.bufferWriter != nil {
		writer = l.bufferWriter
	}

	n, err := writer.Write(outputBytes)
	if err == nil && l.file != nil {
		l.currentSize += int64(n)
	}

	// Flush buffered writer if it exists
	if l.bufferWriter != nil {
		if flushErr := l.bufferWriter.Flush(); flushErr != nil {
			return fmt.Errorf("failed to flush buffer: %w", flushErr)
		}
	}

	return err
}

// shouldAvoidStdout checks if stdout should be avoided (e.g., when TUI is active)
func shouldAvoidStdout() bool {
	// Check if TUI is likely active by looking for terminal characteristics
	// This is a heuristic approach to detect TUI usage

	// Check if TERM suggests an interactive terminal
	term := os.Getenv("TERM")
	if term == "dumb" || term == "" {
		return false // Not a TUI-capable terminal
	}

	// Check if we're in a CI environment (no TUI)
	ciEnvVars := []string{
		"CI", "CONTINUOUS_INTEGRATION", "BUILD_NUMBER", "JENKINS_URL",
		"TRAVIS", "CIRCLECI", "APPVEYOR", "GITLAB_CI", "BUILDKITE",
		"DRONE", "GITHUB_ACTIONS", "TF_BUILD", "TEAMCITY_VERSION",
	}

	for _, envVar := range ciEnvVars {
		if os.Getenv(envVar) != "" {
			return false // In CI, no TUI interference
		}
	}

	// Check if TUI is explicitly disabled
	if tuiEnv := os.Getenv("BLOCO_TUI"); tuiEnv != "" {
		switch tuiEnv {
		case "false", "0", "no", "off", "disabled":
			return false
		}
	}

	// Check if NO_COLOR is set (often indicates non-interactive usage)
	if os.Getenv("NO_COLOR") != "" {
		return false
	}

	// Default to avoiding stdout to prevent TUI interference
	return true
}

// initializeFileWriter sets up the file writer and gets current file size
func (l *FileSecureLogger) initializeFileWriter() error {
	return l.initializeFileWriterWithPath(l.config.OutputFile)
}

// initializeFileWriterWithPath sets up the file writer with a specific path
func (l *FileSecureLogger) initializeFileWriterWithPath(filePath string) error {
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	l.file = file
	l.writer = file

	// Update config to reflect actual file path
	if l.config.OutputFile == "" {
		l.config.OutputFile = filePath
	}

	// Get current file size for rotation tracking
	if stat, err := file.Stat(); err == nil {
		l.currentSize = stat.Size()
	}

	// Initialize buffered writer if buffer size is configured
	if l.config.BufferSize > 0 {
		l.bufferWriter = bufio.NewWriterSize(file, l.config.BufferSize)
		l.writer = l.bufferWriter
	}

	return nil
}

// initializeAsyncBuffering sets up async buffering with a background goroutine
func (l *FileSecureLogger) initializeAsyncBuffering() {
	l.buffer = make(chan *LogEntry, l.config.BufferSize)
	l.flushChan = make(chan chan struct{}, 1)
	l.done = make(chan struct{})
	l.flushTicker = time.NewTicker(1 * time.Second) // Flush every second

	l.wg.Add(1)
	go l.bufferWorker()
}

// bufferWorker processes log entries from the buffer channel
func (l *FileSecureLogger) bufferWorker() {
	defer l.wg.Done()

	for {
		select {
		case entry := <-l.buffer:
			if err := l.writeEntrySync(entry); err != nil {
				// Log buffer write errors to stderr
				fmt.Fprintf(os.Stderr, "Warning: Failed to write buffered log entry: %v\n", err)
			}

		case <-l.flushTicker.C:
			// Periodic flush
			if l.bufferWriter != nil {
				if err := l.bufferWriter.Flush(); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: Failed to flush log buffer: %v\n", err)
				}
			}

		case responseChan := <-l.flushChan:
			// Manual flush request - drain buffer and flush
		drainLoop:
			for {
				select {
				case entry := <-l.buffer:
					if err := l.writeEntrySync(entry); err != nil {
						fmt.Fprintf(os.Stderr, "Warning: Failed to write buffered log entry during flush: %v\n", err)
					}
				default:
					// No more entries, flush and signal completion
					if l.bufferWriter != nil {
						_ = l.bufferWriter.Flush()
					}
					close(responseChan)
					break drainLoop
				}
			}

		case <-l.done:
			// Drain remaining entries before shutdown
			for {
				select {
				case entry := <-l.buffer:
					if err := l.writeEntrySync(entry); err != nil {
						fmt.Fprintf(os.Stderr, "Warning: Failed to write buffered log entry during shutdown: %v\n", err)
					}
				default:
					// Final flush before exit
					if l.bufferWriter != nil {
						_ = l.bufferWriter.Flush()
					}
					return
				}
			}
		}
	}
}

// checkAndRotateFile checks if the log file needs rotation and performs it
func (l *FileSecureLogger) checkAndRotateFile(additionalSize int64) error {
	if l.file == nil || l.config.MaxFileSize <= 0 {
		return nil
	}

	// Check if rotation is needed
	if l.currentSize+additionalSize <= l.config.MaxFileSize {
		return nil
	}

	return l.rotateFile()
}

// rotateFile performs log file rotation
func (l *FileSecureLogger) rotateFile() error {
	if l.file == nil {
		return nil
	}

	// Flush any buffered data before rotation
	if l.bufferWriter != nil {
		if err := l.bufferWriter.Flush(); err != nil {
			return fmt.Errorf("failed to flush buffer before rotation: %w", err)
		}
	}

	// Close current file
	if err := l.file.Close(); err != nil {
		return fmt.Errorf("failed to close current log file: %w", err)
	}

	// Rotate existing files
	if err := l.rotateExistingFiles(); err != nil {
		return fmt.Errorf("failed to rotate existing files: %w", err)
	}

	// Create new log file
	file, err := os.OpenFile(l.config.OutputFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to create new log file: %w", err)
	}

	l.file = file
	l.currentSize = 0

	// Reinitialize buffered writer if needed
	if l.config.BufferSize > 0 {
		l.bufferWriter = bufio.NewWriterSize(file, l.config.BufferSize)
		l.writer = l.bufferWriter
	} else {
		l.writer = file
	}

	return nil
}

// rotateExistingFiles moves existing log files to numbered backups
func (l *FileSecureLogger) rotateExistingFiles() error {
	baseFile := l.config.OutputFile

	// Find existing rotated files
	existingFiles, err := l.findRotatedFiles(baseFile)
	if err != nil {
		return err
	}

	// Sort files by number (highest first)
	sort.Sort(sort.Reverse(sort.StringSlice(existingFiles)))

	// Rotate existing numbered files
	for _, file := range existingFiles {
		num := l.extractFileNumber(file)
		if num >= l.config.MaxFiles {
			// Remove files beyond the limit
			if err := os.Remove(file); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("failed to remove old log file %s: %w", file, err)
			}
		} else {
			// Rename to next number
			newName := fmt.Sprintf("%s.%d", baseFile, num+1)
			if err := os.Rename(file, newName); err != nil {
				return fmt.Errorf("failed to rotate log file %s to %s: %w", file, newName, err)
			}
		}
	}

	// Move current file to .1
	if _, err := os.Stat(baseFile); err == nil {
		rotatedName := fmt.Sprintf("%s.1", baseFile)
		if err := os.Rename(baseFile, rotatedName); err != nil {
			return fmt.Errorf("failed to rotate current log file to %s: %w", rotatedName, err)
		}
	}

	return nil
}

// findRotatedFiles finds all rotated log files for the given base file
func (l *FileSecureLogger) findRotatedFiles(baseFile string) ([]string, error) {
	dir := filepath.Dir(baseFile)
	baseName := filepath.Base(baseFile)

	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var rotatedFiles []string
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		name := file.Name()
		if strings.HasPrefix(name, baseName+".") && l.isRotatedFile(name, baseName) {
			rotatedFiles = append(rotatedFiles, filepath.Join(dir, name))
		}
	}

	return rotatedFiles, nil
}

// isRotatedFile checks if a filename is a rotated log file
func (l *FileSecureLogger) isRotatedFile(filename, baseName string) bool {
	suffix := strings.TrimPrefix(filename, baseName+".")
	if suffix == filename {
		return false
	}

	// Check if suffix is a number
	_, err := strconv.Atoi(suffix)
	return err == nil
}

// extractFileNumber extracts the rotation number from a rotated log file
func (l *FileSecureLogger) extractFileNumber(filename string) int {
	parts := strings.Split(filepath.Base(filename), ".")
	if len(parts) < 2 {
		return 0
	}

	num, err := strconv.Atoi(parts[len(parts)-1])
	if err != nil {
		return 0
	}

	return num
}

// isSafeParameter checks if a parameter key is safe to log
func isSafeParameter(key string) bool {
	keyLower := strings.ToLower(key)

	// List of explicitly unsafe parameter keys that contain sensitive data
	unsafeKeys := map[string]bool{
		"private_key": true,
		"privatekey":  true,
		"public_key":  true,
		"publickey":   true,
		"seed":        true,
		"mnemonic":    true,
		"password":    true,
		"pwd":         true,
		"pass":        true,
		"secret":      true,
		"token":       true,
		"auth":        true,
		"credential":  true,
		"key":         true,
		"keystore":    true,
	}

	// Check if the key is explicitly unsafe
	if unsafeKeys[keyLower] {
		return false
	}

	// Check for unsafe patterns in key names
	unsafePatterns := []string{
		"private", "secret", "password", "pwd", "pass", "auth", "token",
		"credential", "key", "seed", "mnemonic",
	}

	for _, pattern := range unsafePatterns {
		if strings.Contains(keyLower, pattern) {
			return false
		}
	}

	// List of safe parameter keys that don't contain sensitive data
	safeKeys := map[string]bool{
		"prefix":                true,
		"suffix":                true,
		"count":                 true,
		"threads":               true,
		"timeout":               true,
		"attempts":              true,
		"duration":              true,
		"duration_ns":           true,
		"thread_id":             true,
		"address":               true,
		"pattern":               true,
		"operation":             true,
		"status":                true,
		"success":               true,
		"items_count":           true,
		"error_count":           true,
		"throughput":            true,
		"throughput_per_second": true,
		"cpu_usage":             true,
		"cpu_usage_percent":     true,
		"memory_usage":          true,
		"memory_usage_bytes":    true,
		"error_category":        true,
		"error_message":         true,
		"level":                 true,
		"timestamp":             true,
		"message":               true,
		"file":                  true,
		"line":                  true,
		"function":              true,
		"checksum":              true,
		"format":                true,
		"output":                true,
		"input":                 true,
		"size":                  true,
		"length":                true,
		"version":               true,
		"type":                  true,
		"method":                true,
		"algorithm":             true,
		// Performance metrics parameters
		"wallets_per_second":   true,
		"total_wallets":        true,
		"total_attempts":       true,
		"avg_duration_ms":      true,
		"min_duration_ms":      true,
		"max_duration_ms":      true,
		"thread_count":         true,
		"success_rate_percent": true,
		"error_rate_percent":   true,
		"window_start":         true,
		"window_duration_ns":   true,
	}

	// If explicitly listed as safe, allow it
	if safeKeys[keyLower] {
		return true
	}

	// For unknown keys, be conservative and don't log them
	return false
}

// Patterns for detecting sensitive data in error messages
var (
	// Private key patterns (32 bytes hex, with or without 0x prefix)
	privateKeyPattern = regexp.MustCompile(`(?i)\b(0x)?[a-f0-9]{64}\b`)
	// Public key patterns (64 bytes hex, with or without 0x prefix)
	publicKeyPattern = regexp.MustCompile(`(?i)\b(0x)?[a-f0-9]{128}\b`)
	// Seed phrase patterns (12-24 words)
	seedPhrasePattern = regexp.MustCompile(`(?i)\b([a-z]+\s+){11,23}[a-z]+\b`)
	// Password patterns in URLs or connection strings
	passwordURLPattern   = regexp.MustCompile(`(?i)://([^:]+):([^@]+)@`)
	passwordParamPattern = regexp.MustCompile(`(?i)(password|pwd|pass)=([^&\s]+)`)
	// UTC keystore filename pattern
	keystoreFilenamePattern = regexp.MustCompile(`UTC--[0-9T\-:.Z]+--[a-fA-F0-9]+`)
)

// sanitizeError removes sensitive information from error messages
func sanitizeError(err error) string {
	if err == nil {
		return ""
	}

	errMsg := err.Error()

	// Remove private keys
	errMsg = privateKeyPattern.ReplaceAllString(errMsg, "[PRIVATE_KEY_REDACTED]")

	// Remove public keys
	errMsg = publicKeyPattern.ReplaceAllString(errMsg, "[PUBLIC_KEY_REDACTED]")

	// Remove seed phrases
	errMsg = seedPhrasePattern.ReplaceAllString(errMsg, "[SEED_PHRASE_REDACTED]")

	// Remove passwords from URLs (user:pass@host format)
	errMsg = passwordURLPattern.ReplaceAllString(errMsg, "://${1}:[REDACTED]@")

	// Remove passwords from parameter format
	errMsg = passwordParamPattern.ReplaceAllString(errMsg, "${1}=[REDACTED]")

	// Sanitize sensitive file paths
	errMsg = sanitizeFilePaths(errMsg)

	return errMsg
}

// sanitizeFilePaths removes or redacts sensitive file paths
func sanitizeFilePaths(text string) string {
	// Replace UTC keystore filenames
	text = keystoreFilenamePattern.ReplaceAllString(text, "[KEYSTORE_FILE_REDACTED]")

	// Replace paths containing sensitive directory names
	text = regexp.MustCompile(`/[^/\s]*/keystore(/[^/\s]*)*`).ReplaceAllString(text, "./[REDACTED]")
	text = regexp.MustCompile(`/[^/\s]*/wallet(/[^/\s]*)*`).ReplaceAllString(text, "./[REDACTED]")
	text = regexp.MustCompile(`/[^/\s]*/private(/[^/\s]*)*`).ReplaceAllString(text, "./[REDACTED]")
	text = regexp.MustCompile(`/[^/\s]*/secret(/[^/\s]*)*`).ReplaceAllString(text, "./[REDACTED]")
	text = regexp.MustCompile(`[A-Z]:[^\\s]*\\keystore(\\[^\\s]*)*`).ReplaceAllString(text, ".\\[REDACTED]")
	text = regexp.MustCompile(`[A-Z]:[^\\s]*\\wallet(\\[^\\s]*)*`).ReplaceAllString(text, ".\\[REDACTED]")
	text = regexp.MustCompile(`[A-Z]:[^\\s]*\\private(\\[^\\s]*)*`).ReplaceAllString(text, ".\\[REDACTED]")
	text = regexp.MustCompile(`[A-Z]:[^\\s]*\\secret(\\[^\\s]*)*`).ReplaceAllString(text, ".\\[REDACTED]")

	return text
}

// categorizeError determines the category of an error based on its content and type
func categorizeError(err error) ErrorCategory {
	if err == nil {
		return ErrorUnknown
	}

	errMsg := strings.ToLower(err.Error())

	// Crypto-related errors
	if strings.Contains(errMsg, "crypto") ||
		strings.Contains(errMsg, "key") ||
		strings.Contains(errMsg, "encrypt") ||
		strings.Contains(errMsg, "decrypt") ||
		strings.Contains(errMsg, "hash") ||
		strings.Contains(errMsg, "signature") ||
		strings.Contains(errMsg, "secp256k1") ||
		strings.Contains(errMsg, "keccak") {
		return ErrorCrypto
	}

	// Validation errors
	if strings.Contains(errMsg, "invalid") ||
		strings.Contains(errMsg, "validation") ||
		strings.Contains(errMsg, "format") ||
		strings.Contains(errMsg, "parse") ||
		strings.Contains(errMsg, "malformed") {
		return ErrorValidation
	}

	// I/O errors
	if strings.Contains(errMsg, "file") ||
		strings.Contains(errMsg, "directory") ||
		strings.Contains(errMsg, "permission") ||
		strings.Contains(errMsg, "no such") ||
		strings.Contains(errMsg, "read") ||
		strings.Contains(errMsg, "write") ||
		strings.Contains(errMsg, "open") ||
		strings.Contains(errMsg, "close") {
		return ErrorIO
	}

	// Network errors
	if strings.Contains(errMsg, "network") ||
		strings.Contains(errMsg, "connection") ||
		strings.Contains(errMsg, "timeout") ||
		strings.Contains(errMsg, "dns") ||
		strings.Contains(errMsg, "http") ||
		strings.Contains(errMsg, "tcp") ||
		strings.Contains(errMsg, "udp") {
		return ErrorNetwork
	}

	// System errors
	if strings.Contains(errMsg, "system") ||
		strings.Contains(errMsg, "memory") ||
		strings.Contains(errMsg, "resource") ||
		strings.Contains(errMsg, "process") ||
		strings.Contains(errMsg, "signal") {
		return ErrorSystem
	}

	return ErrorUnknown
}

// sanitizeParameterValue sanitizes parameter values based on their key
func sanitizeParameterValue(key string, value interface{}) interface{} {
	// Convert to string for pattern matching
	strValue, ok := value.(string)
	if !ok {
		// For non-string values, return as-is if they're safe types
		switch value.(type) {
		case int, int32, int64, uint, uint32, uint64, float32, float64, bool:
			return value
		default:
			return fmt.Sprintf("%v", value)
		}
	}

	// Apply sanitization patterns to string values
	sanitized := sanitizeError(fmt.Errorf("%s", strValue))

	// Additional sanitization for specific parameter types
	switch strings.ToLower(key) {
	case "path", "file", "directory":
		return sanitizeFilePaths(sanitized)
	case "url", "endpoint":
		// Apply URL password sanitization
		sanitized = passwordURLPattern.ReplaceAllString(sanitized, "://${1}:[REDACTED]@")
		return passwordParamPattern.ReplaceAllString(sanitized, "${1}=[REDACTED]")
	default:
		return sanitized
	}
}
