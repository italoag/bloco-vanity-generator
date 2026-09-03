package config

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"time"
)

// Config holds all application configuration
type Config struct {
	Worker   WorkerConfig   `yaml:"worker"`
	TUI      TUIConfig      `yaml:"tui"`
	Crypto   CryptoConfig   `yaml:"crypto"`
	CLI      CLIConfig      `yaml:"cli"`
	KeyStore KeyStoreConfig `yaml:"keystore"`
	Logging  LoggingConfig  `yaml:"logging"`
}

// WorkerConfig contains worker-related configuration
type WorkerConfig struct {
	ThreadCount       int           `yaml:"thread_count"`
	MinBatchSize      int           `yaml:"min_batch_size"`
	MaxBatchSize      int           `yaml:"max_batch_size"`
	UpdateInterval    time.Duration `yaml:"update_interval"`
	HealthCheckPeriod time.Duration `yaml:"health_check_period"`
	ShutdownTimeout   time.Duration `yaml:"shutdown_timeout"`
}

// TUIConfig contains TUI-related configuration
type TUIConfig struct {
	Enabled          bool          `yaml:"enabled"`
	RefreshRate      time.Duration `yaml:"refresh_rate"`
	ProgressBarWidth int           `yaml:"progress_bar_width"`
	MaxTableRows     int           `yaml:"max_table_rows"`
	ColorSupport     string        `yaml:"color_support"`   // auto, enabled, disabled
	UnicodeSupport   string        `yaml:"unicode_support"` // auto, enabled, disabled
}

// CryptoConfig contains cryptographic configuration
type CryptoConfig struct {
	PoolSize         int  `yaml:"pool_size"`
	SecureRandom     bool `yaml:"secure_random"`
	OptimizedHashing bool `yaml:"optimized_hashing"`
	MemoryClearing   bool `yaml:"memory_clearing"`
}

// CLIConfig contains CLI-related configuration
type CLIConfig struct {
	ProgressUpdateInterval time.Duration `yaml:"progress_update_interval"`
	VerboseOutput          bool          `yaml:"verbose_output"`
	QuietMode              bool          `yaml:"quiet_mode"`
}

// KeyStoreConfig contains keystore generation configuration
type KeyStoreConfig struct {
	Enabled       bool                   `yaml:"enabled"`
	OutputDir     string                 `yaml:"output_dir"`
	KDFAlgorithm  string                 `yaml:"kdf_algorithm"`
	KDFParams     map[string]interface{} `yaml:"kdf_params"`
	CreateDirs    bool                   `yaml:"create_dirs"`
	FileMode      int                    `yaml:"file_mode"`
	ShowAnalysis  bool                   `yaml:"show_analysis"`
	SecurityLevel string                 `yaml:"security_level"`

	// WritePasswordFile writes the keystore password as <address>.pwd next to
	// the keystore. Off by default: it nullifies the KDF for anyone who copies
	// the directory.
	WritePasswordFile bool `yaml:"write_password_file"`

	// WritePlaintextKey writes the raw private key as <address>.key. Off by
	// default for the same reason.
	WritePlaintextKey bool `yaml:"write_plaintext_key"`
}

// LoggingConfig contains logging configuration
type LoggingConfig struct {
	Enabled     bool   `yaml:"enabled"`
	Level       string `yaml:"level"`
	Format      string `yaml:"format"`
	OutputFile  string `yaml:"output_file"`
	MaxFileSize int64  `yaml:"max_file_size"`
	MaxFiles    int    `yaml:"max_files"`
	BufferSize  int    `yaml:"buffer_size"`
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		Worker: WorkerConfig{
			ThreadCount:       runtime.NumCPU(),
			MinBatchSize:      100,
			MaxBatchSize:      10000,
			UpdateInterval:    100 * time.Millisecond,
			HealthCheckPeriod: time.Second,
			ShutdownTimeout:   5 * time.Second,
		},
		TUI: TUIConfig{
			Enabled:          true,
			RefreshRate:      500 * time.Millisecond,
			ProgressBarWidth: 40,
			MaxTableRows:     8,
			ColorSupport:     "auto",
			UnicodeSupport:   "auto",
		},
		Crypto: CryptoConfig{
			PoolSize:         runtime.NumCPU() * 2,
			SecureRandom:     true,
			OptimizedHashing: true,
			MemoryClearing:   true,
		},
		CLI: CLIConfig{
			ProgressUpdateInterval: 500 * time.Millisecond,
			VerboseOutput:          false,
			QuietMode:              false,
		},
		KeyStore: KeyStoreConfig{
			Enabled:       true,
			OutputDir:     "./keystores",
			KDFAlgorithm:  "scrypt",
			KDFParams:     make(map[string]interface{}),
			CreateDirs:    true,
			FileMode:      0600,
			ShowAnalysis:  false,
			SecurityLevel: "medium",
		},
		Logging: LoggingConfig{
			Enabled:     true,
			Level:       "info",
			Format:      "text",
			OutputFile:  "",
			MaxFileSize: 10 * 1024 * 1024, // 10MB
			MaxFiles:    5,
			BufferSize:  1000,
		},
	}
}

// LoadFromEnvironment loads configuration from environment variables
func (c *Config) LoadFromEnvironment() {
	// Worker configuration
	if threads := os.Getenv("BLOCO_THREADS"); threads != "" {
		if val, err := strconv.Atoi(threads); err == nil && val > 0 {
			c.Worker.ThreadCount = val
		}
	}

	if batchSize := os.Getenv("BLOCO_BATCH_SIZE"); batchSize != "" {
		if val, err := strconv.Atoi(batchSize); err == nil && val > 0 {
			c.Worker.MaxBatchSize = val
		}
	}

	// TUI configuration
	if tuiEnabled := os.Getenv("BLOCO_TUI"); tuiEnabled != "" {
		c.TUI.Enabled = parseBoolEnv(tuiEnabled, c.TUI.Enabled)
	}

	if colorSupport := os.Getenv("BLOCO_COLOR"); colorSupport != "" {
		c.TUI.ColorSupport = colorSupport
	}

	// Check NO_COLOR standard
	if os.Getenv("NO_COLOR") != "" {
		c.TUI.ColorSupport = "disabled"
	}

	// CLI configuration
	if verbose := os.Getenv("BLOCO_VERBOSE"); verbose != "" {
		c.CLI.VerboseOutput = parseBoolEnv(verbose, c.CLI.VerboseOutput)
	}

	if quiet := os.Getenv("BLOCO_QUIET"); quiet != "" {
		c.CLI.QuietMode = parseBoolEnv(quiet, c.CLI.QuietMode)
	}

	// KeyStore configuration
	if keystoreEnabled := os.Getenv("BLOCO_KEYSTORE_ENABLED"); keystoreEnabled != "" {
		c.KeyStore.Enabled = parseBoolEnv(keystoreEnabled, c.KeyStore.Enabled)
	}

	if keystoreDir := os.Getenv("BLOCO_KEYSTORE_DIR"); keystoreDir != "" {
		c.KeyStore.OutputDir = keystoreDir
	}

	if keystoreKDF := os.Getenv("BLOCO_KEYSTORE_KDF"); keystoreKDF != "" {
		c.KeyStore.KDFAlgorithm = keystoreKDF
	}

	if showAnalysis := os.Getenv("BLOCO_KDF_ANALYSIS"); showAnalysis != "" {
		c.KeyStore.ShowAnalysis = parseBoolEnv(showAnalysis, c.KeyStore.ShowAnalysis)
	}

	if securityLevel := os.Getenv("BLOCO_SECURITY_LEVEL"); securityLevel != "" {
		c.KeyStore.SecurityLevel = securityLevel
	}

	// Logging configuration
	if loggingEnabled := os.Getenv("BLOCO_LOGGING_ENABLED"); loggingEnabled != "" {
		c.Logging.Enabled = parseBoolEnv(loggingEnabled, c.Logging.Enabled)
	}

	if logLevel := os.Getenv("BLOCO_LOG_LEVEL"); logLevel != "" {
		c.Logging.Level = logLevel
	}

	if logFormat := os.Getenv("BLOCO_LOG_FORMAT"); logFormat != "" {
		c.Logging.Format = logFormat
	}

	if logFile := os.Getenv("BLOCO_LOG_FILE"); logFile != "" {
		c.Logging.OutputFile = logFile
	}
}

// Validate validates the configuration and returns any errors
func (c *Config) Validate() error {
	// Validate worker configuration
	if c.Worker.ThreadCount <= 0 {
		return fmt.Errorf("worker thread count must be positive, got %d", c.Worker.ThreadCount)
	}

	if c.Worker.ThreadCount > 128 {
		return fmt.Errorf("worker thread count too high (max 128), got %d", c.Worker.ThreadCount)
	}

	if c.Worker.MinBatchSize <= 0 {
		return fmt.Errorf("worker min batch size must be positive, got %d", c.Worker.MinBatchSize)
	}

	if c.Worker.MaxBatchSize < c.Worker.MinBatchSize {
		return fmt.Errorf("worker max batch size (%d) must be >= min batch size (%d)",
			c.Worker.MaxBatchSize, c.Worker.MinBatchSize)
	}

	// Validate TUI configuration
	if c.TUI.ProgressBarWidth <= 0 {
		return fmt.Errorf("TUI progress bar width must be positive, got %d", c.TUI.ProgressBarWidth)
	}

	if c.TUI.MaxTableRows <= 0 {
		return fmt.Errorf("TUI max table rows must be positive, got %d", c.TUI.MaxTableRows)
	}

	// Validate color support setting
	validColorSettings := []string{"auto", "enabled", "disabled"}
	if !contains(validColorSettings, c.TUI.ColorSupport) {
		return fmt.Errorf("invalid color support setting: %s (valid: %v)",
			c.TUI.ColorSupport, validColorSettings)
	}

	// Validate unicode support setting
	validUnicodeSettings := []string{"auto", "enabled", "disabled"}
	if !contains(validUnicodeSettings, c.TUI.UnicodeSupport) {
		return fmt.Errorf("invalid unicode support setting: %s (valid: %v)",
			c.TUI.UnicodeSupport, validUnicodeSettings)
	}

	// Validate crypto configuration
	if c.Crypto.PoolSize <= 0 {
		return fmt.Errorf("crypto pool size must be positive, got %d", c.Crypto.PoolSize)
	}

	// Validate CLI configuration - quiet and verbose are mutually exclusive
	if c.CLI.QuietMode && c.CLI.VerboseOutput {
		return fmt.Errorf("quiet mode and verbose output are mutually exclusive")
	}

	// Validate KeyStore configuration
	if c.KeyStore.OutputDir == "" {
		return fmt.Errorf("keystore output directory cannot be empty")
	}

	validKDFAlgorithms := []string{"scrypt", "pbkdf2", "pbkdf2-sha256", "pbkdf2-sha512"}
	if !contains(validKDFAlgorithms, c.KeyStore.KDFAlgorithm) {
		return fmt.Errorf("invalid KDF algorithm: %s (valid: %v)",
			c.KeyStore.KDFAlgorithm, validKDFAlgorithms)
	}

	validSecurityLevels := []string{"low", "medium", "high", "very-high"}
	if !contains(validSecurityLevels, c.KeyStore.SecurityLevel) {
		return fmt.Errorf("invalid security level: %s (valid: %v)",
			c.KeyStore.SecurityLevel, validSecurityLevels)
	}

	// Keystore artifacts hold private key material, so the mode may not grant
	// anything to group or other, and may not exceed owner read/write.
	if c.KeyStore.FileMode <= 0 || c.KeyStore.FileMode&^0o600 != 0 {
		return fmt.Errorf(
			"invalid keystore file mode: %04o (private key material must stay owner-only; use 0600 or 0400)",
			c.KeyStore.FileMode)
	}

	// Validate Logging configuration
	validLogLevels := []string{"error", "warn", "info", "debug"}
	if !contains(validLogLevels, c.Logging.Level) {
		return fmt.Errorf("invalid log level: %s (valid: %v)",
			c.Logging.Level, validLogLevels)
	}

	validLogFormats := []string{"text", "json", "structured"}
	if !contains(validLogFormats, c.Logging.Format) {
		return fmt.Errorf("invalid log format: %s (valid: %v)",
			c.Logging.Format, validLogFormats)
	}

	if c.Logging.MaxFileSize <= 0 {
		return fmt.Errorf("log max file size must be positive, got %d", c.Logging.MaxFileSize)
	}

	if c.Logging.MaxFiles < 0 {
		return fmt.Errorf("log max files must be non-negative, got %d", c.Logging.MaxFiles)
	}

	if c.Logging.BufferSize < 0 {
		return fmt.Errorf("log buffer size must be non-negative, got %d", c.Logging.BufferSize)
	}

	return nil
}

// ApplyOverrides applies command-line overrides to the configuration
func (c *Config) ApplyOverrides(overrides ConfigOverrides) {
	if overrides.ThreadCount != nil {
		c.Worker.ThreadCount = *overrides.ThreadCount
	}

	if overrides.TUIEnabled != nil {
		c.TUI.Enabled = *overrides.TUIEnabled
	}

	if overrides.VerboseOutput != nil {
		c.CLI.VerboseOutput = *overrides.VerboseOutput
	}

	if overrides.QuietMode != nil {
		c.CLI.QuietMode = *overrides.QuietMode
	}

	if overrides.KeyStoreEnabled != nil {
		c.KeyStore.Enabled = *overrides.KeyStoreEnabled
	}

	if overrides.KeyStoreOutputDir != nil {
		c.KeyStore.OutputDir = *overrides.KeyStoreOutputDir
	}

	if overrides.KeyStoreKDF != nil {
		c.KeyStore.KDFAlgorithm = *overrides.KeyStoreKDF
	}

	if overrides.KDFParams != nil {
		c.KeyStore.KDFParams = *overrides.KDFParams
	}

	if overrides.ShowKDFAnalysis != nil {
		c.KeyStore.ShowAnalysis = *overrides.ShowKDFAnalysis
	}

	if overrides.SecurityLevel != nil {
		c.KeyStore.SecurityLevel = *overrides.SecurityLevel
	}

	if overrides.LoggingEnabled != nil {
		c.Logging.Enabled = *overrides.LoggingEnabled
	}

	if overrides.LogLevel != nil {
		c.Logging.Level = *overrides.LogLevel
	}

	if overrides.LogFormat != nil {
		c.Logging.Format = *overrides.LogFormat
	}

	if overrides.LogFile != nil {
		c.Logging.OutputFile = *overrides.LogFile
	}
}

// ConfigOverrides represents command-line configuration overrides
type ConfigOverrides struct {
	ThreadCount       *int
	TUIEnabled        *bool
	VerboseOutput     *bool
	QuietMode         *bool
	KeyStoreEnabled   *bool
	KeyStoreOutputDir *string
	KeyStoreKDF       *string
	KDFParams         *map[string]interface{}
	ShowKDFAnalysis   *bool
	SecurityLevel     *string
	LoggingEnabled    *bool
	LogLevel          *string
	LogFormat         *string
	LogFile           *string
}

// parseBoolEnv parses a boolean environment variable with fallback
func parseBoolEnv(value string, fallback bool) bool {
	switch value {
	case "true", "1", "yes", "on", "enabled", "force":
		return true
	case "false", "0", "no", "off", "disabled":
		return false
	default:
		return fallback
	}
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// GetEffectiveThreadCount returns the effective thread count considering system limits
func (c *Config) GetEffectiveThreadCount() int {
	maxRecommended := runtime.NumCPU() * 2
	if c.Worker.ThreadCount > maxRecommended {
		return maxRecommended
	}
	return c.Worker.ThreadCount
}

// IsTUIEnabled returns whether TUI should be enabled based on configuration and environment
func (c *Config) IsTUIEnabled() bool {
	if !c.TUI.Enabled {
		return false
	}

	// Check if we're in a CI environment
	ciEnvVars := []string{
		"CI", "CONTINUOUS_INTEGRATION", "BUILD_NUMBER", "JENKINS_URL",
		"TRAVIS", "CIRCLECI", "APPVEYOR", "GITLAB_CI", "BUILDKITE",
		"DRONE", "GITHUB_ACTIONS", "TF_BUILD", "TEAMCITY_VERSION",
	}

	for _, envVar := range ciEnvVars {
		if os.Getenv(envVar) != "" {
			return false
		}
	}

	return true
}
