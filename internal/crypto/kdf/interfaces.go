package kdf

import (
	"time"

	"bloco-vgen/pkg/logging"
)

// KDFHandler interface for different types of KDF implementations
type KDFHandler interface {
	// DeriveKey derives a key using the specified password and parameters
	DeriveKey(password string, params map[string]interface{}) ([]byte, error)

	// ValidateParams validates the KDF parameters for correctness and security
	ValidateParams(params map[string]interface{}) error

	// GetDefaultParams returns the default parameters for this KDF type
	GetDefaultParams() map[string]interface{}

	// GetParamRange returns the valid range for a specific parameter
	GetParamRange(param string) (min, max interface{})
}

// KDFLogger interface for logging KDF operations
// This interface adapts to the existing bloco-vgen logging system
type KDFLogger interface {
	// LogKDFAttempt logs the start of a KDF operation with sanitized parameters
	LogKDFAttempt(kdf string, params map[string]interface{})

	// LogKDFSuccess logs successful completion of a KDF operation
	LogKDFSuccess(kdf string, duration time.Duration)

	// LogKDFError logs KDF operation errors
	LogKDFError(kdf string, err error)

	// LogParamValidation logs parameter validation results
	LogParamValidation(param string, value interface{}, valid bool)
}

// SecureKDFLogger implements KDFLogger using the existing SecureLogger
type SecureKDFLogger struct {
	logger logging.SecureLogger
}

// NewSecureKDFLogger creates a new KDF logger using the existing secure logger
func NewSecureKDFLogger(logger logging.SecureLogger) KDFLogger {
	return &SecureKDFLogger{
		logger: logger,
	}
}

// LogKDFAttempt logs the start of a KDF operation
func (l *SecureKDFLogger) LogKDFAttempt(kdf string, params map[string]interface{}) {
	if l.logger == nil {
		return
	}

	// Create sanitized parameters for logging (exclude sensitive data)
	sanitizedParams := make(map[string]interface{})
	for key, value := range params {
		if isSafeKDFParameter(key) {
			sanitizedParams[key] = sanitizeKDFParameterValue(key, value)
		}
	}

	if err := l.logger.LogOperationStart("kdf_derive", map[string]interface{}{
		"kdf_type": kdf,
		"params":   sanitizedParams,
	}); err != nil {
		// Logging error shouldn't stop the operation
		_ = err
	}
}

// LogKDFSuccess logs successful completion of a KDF operation
func (l *SecureKDFLogger) LogKDFSuccess(kdf string, duration time.Duration) {
	if l.logger == nil {
		return
	}

	if err := l.logger.LogOperationComplete("kdf_derive", logging.OperationStats{
		Duration: duration,
		Success:  true,
	}); err != nil {
		// Logging error shouldn't stop the operation
		_ = err
	}
}

// LogKDFError logs KDF operation errors
func (l *SecureKDFLogger) LogKDFError(kdf string, err error) {
	if l.logger == nil {
		return
	}

	if logErr := l.logger.LogError("kdf_derive", err, map[string]interface{}{
		"kdf_type": kdf,
	}); logErr != nil {
		// Logging error shouldn't stop the operation
		_ = logErr
	}
}

// LogParamValidation logs parameter validation results
func (l *SecureKDFLogger) LogParamValidation(param string, value interface{}, valid bool) {
	if l.logger == nil {
		return
	}

	// Only log validation failures to avoid cluttering logs
	if !valid {
		sanitizedValue := sanitizeKDFParameterValue(param, value)
		if err := l.logger.Warn("KDF parameter validation failed",
			logging.NewLogField("parameter", param),
			logging.NewLogField("value", sanitizedValue),
			logging.NewLogField("valid", valid),
		); err != nil {
			// Logging error shouldn't stop the operation
			_ = err
		}
	}
}

// isSafeKDFParameter checks if a KDF parameter is safe to log
func isSafeKDFParameter(key string) bool {
	// Safe KDF parameters that don't contain sensitive data
	safeParams := map[string]bool{
		"n":     true, // scrypt N parameter
		"r":     true, // scrypt r parameter
		"p":     true, // scrypt p parameter
		"dklen": true, // derived key length
		"c":     true, // PBKDF2 iteration count
		"prf":   true, // PBKDF2 pseudo-random function
	}

	// Salt is sensitive and should not be logged
	if key == "salt" || key == "Salt" || key == "SALT" {
		return false
	}

	return safeParams[key]
}

// sanitizeKDFParameterValue sanitizes KDF parameter values for logging
func sanitizeKDFParameterValue(key string, value interface{}) interface{} {
	// For salt parameters, never log the actual value
	if key == "salt" || key == "Salt" || key == "SALT" {
		return "[REDACTED]"
	}

	// For other parameters, return the value as-is since they're not sensitive
	return value
}
