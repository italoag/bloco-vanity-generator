package validation

import (
	"strings"

	"bloco-vgen/internal/crypto"
	"bloco-vgen/pkg/errors"
	"bloco-vgen/pkg/wallet"
)

// ValidationStrategy defines the interface for address validation strategies
type ValidationStrategy interface {
	Validate(address, prefix, suffix string) (bool, error)
	Name() string
	Description() string
}

// AddressValidator manages address validation using different strategies
type AddressValidator struct {
	strategy          ValidationStrategy
	checksumValidator *crypto.ChecksumValidator
}

// NewAddressValidator creates a new address validator
func NewAddressValidator(checksumValidator *crypto.ChecksumValidator) *AddressValidator {
	return &AddressValidator{
		checksumValidator: checksumValidator,
		strategy:          NewCaseInsensitiveStrategy(), // Default strategy
	}
}

// SetStrategy sets the validation strategy
func (av *AddressValidator) SetStrategy(strategy ValidationStrategy) {
	av.strategy = strategy
}

// Validate validates an address using the current strategy
func (av *AddressValidator) Validate(address, prefix, suffix string) (bool, error) {
	// Basic validation first
	if err := av.validateBasicFormat(address, prefix, suffix); err != nil {
		return false, err
	}

	// Use strategy for specific validation
	return av.strategy.Validate(address, prefix, suffix)
}

// ValidateWithCriteria validates an address against generation criteria
func (av *AddressValidator) ValidateWithCriteria(address string, criteria wallet.GenerationCriteria) (bool, error) {
	// Set appropriate strategy based on criteria
	if criteria.IsChecksum {
		av.SetStrategy(NewChecksumStrategy(av.checksumValidator))
	} else {
		av.SetStrategy(NewCaseInsensitiveStrategy())
	}

	return av.Validate(address, criteria.Prefix, criteria.Suffix)
}

// validateBasicFormat performs basic format validation
func (av *AddressValidator) validateBasicFormat(address, prefix, suffix string) error {
	// Check address length
	if len(address) != 40 {
		return errors.NewValidationError("validate_basic_format",
			"address must be 40 characters long")
	}

	// Check if address contains only hex characters
	if !av.checksumValidator.IsValidHexAddress(address) {
		return errors.NewValidationError("validate_basic_format",
			"address contains invalid hex characters")
	}

	// Check prefix/suffix lengths
	if len(prefix) > len(address) {
		return errors.NewValidationError("validate_basic_format",
			"prefix longer than address")
	}

	if len(suffix) > len(address) {
		return errors.NewValidationError("validate_basic_format",
			"suffix longer than address")
	}

	// Check for overlap
	if len(prefix)+len(suffix) > len(address) {
		return errors.NewValidationError("validate_basic_format",
			"prefix and suffix overlap")
	}

	return nil
}

// CaseInsensitiveStrategy validates addresses without checksum consideration
type CaseInsensitiveStrategy struct{}

// NewCaseInsensitiveStrategy creates a new case-insensitive validation strategy
func NewCaseInsensitiveStrategy() *CaseInsensitiveStrategy {
	return &CaseInsensitiveStrategy{}
}

// Validate performs case-insensitive validation
func (cis *CaseInsensitiveStrategy) Validate(address, prefix, suffix string) (bool, error) {
	// Fast path: if both prefix and suffix are empty, return true
	if len(prefix) == 0 && len(suffix) == 0 {
		return true, nil
	}

	// Extract prefix and suffix from address
	var addressPrefix, addressSuffix string
	if len(prefix) > 0 {
		addressPrefix = address[:len(prefix)]
	}
	if len(suffix) > 0 {
		addressSuffix = address[len(address)-len(suffix):]
	}

	// Compare case-insensitively
	prefixMatch := len(prefix) == 0 || strings.EqualFold(prefix, addressPrefix)
	suffixMatch := len(suffix) == 0 || strings.EqualFold(suffix, addressSuffix)

	return prefixMatch && suffixMatch, nil
}

// Name returns the strategy name
func (cis *CaseInsensitiveStrategy) Name() string {
	return "case_insensitive"
}

// Description returns the strategy description
func (cis *CaseInsensitiveStrategy) Description() string {
	return "Validates addresses without considering case (no checksum validation)"
}

// ChecksumStrategy validates addresses with EIP-55 checksum consideration
type ChecksumStrategy struct {
	checksumValidator *crypto.ChecksumValidator
}

// NewChecksumStrategy creates a new checksum validation strategy
func NewChecksumStrategy(checksumValidator *crypto.ChecksumValidator) *ChecksumStrategy {
	return &ChecksumStrategy{
		checksumValidator: checksumValidator,
	}
}

// Validate performs checksum-aware validation
func (cs *ChecksumStrategy) Validate(address, prefix, suffix string) (bool, error) {
	// Fast path: if both prefix and suffix are empty, return true
	if len(prefix) == 0 && len(suffix) == 0 {
		return true, nil
	}

	// Extract prefix and suffix from address
	var addressPrefix, addressSuffix string
	if len(prefix) > 0 {
		addressPrefix = address[:len(prefix)]
	}
	if len(suffix) > 0 {
		addressSuffix = address[len(address)-len(suffix):]
	}

	// First check if lowercase versions match (quick elimination)
	prefixMatch := len(prefix) == 0 || strings.EqualFold(prefix, addressPrefix)
	suffixMatch := len(suffix) == 0 || strings.EqualFold(suffix, addressSuffix)

	if !prefixMatch || !suffixMatch {
		return false, nil
	}

	// Perform checksum validation for the pattern
	return cs.checksumValidator.ValidatePatternChecksum(address, prefix, suffix)
}

// Name returns the strategy name
func (cs *ChecksumStrategy) Name() string {
	return "checksum"
}

// Description returns the strategy description
func (cs *ChecksumStrategy) Description() string {
	return "Validates addresses with EIP-55 checksum consideration"
}

// ExactMatchStrategy validates addresses with exact string matching
type ExactMatchStrategy struct{}

// NewExactMatchStrategy creates a new exact match validation strategy
func NewExactMatchStrategy() *ExactMatchStrategy {
	return &ExactMatchStrategy{}
}

// Validate performs exact string matching validation
func (ems *ExactMatchStrategy) Validate(address, prefix, suffix string) (bool, error) {
	// Fast path: if both prefix and suffix are empty, return true
	if len(prefix) == 0 && len(suffix) == 0 {
		return true, nil
	}

	// Extract prefix and suffix from address
	var addressPrefix, addressSuffix string
	if len(prefix) > 0 {
		addressPrefix = address[:len(prefix)]
	}
	if len(suffix) > 0 {
		addressSuffix = address[len(address)-len(suffix):]
	}

	// Exact string comparison
	prefixMatch := len(prefix) == 0 || prefix == addressPrefix
	suffixMatch := len(suffix) == 0 || suffix == addressSuffix

	return prefixMatch && suffixMatch, nil
}

// Name returns the strategy name
func (ems *ExactMatchStrategy) Name() string {
	return "exact_match"
}

// Description returns the strategy description
func (ems *ExactMatchStrategy) Description() string {
	return "Validates addresses with exact string matching (case-sensitive)"
}

// OptimizedStrategy provides high-performance validation for worker threads
type OptimizedStrategy struct {
	checksumValidator *crypto.ChecksumValidator
	isChecksum        bool
}

// NewOptimizedStrategy creates a new optimized validation strategy
func NewOptimizedStrategy(checksumValidator *crypto.ChecksumValidator, isChecksum bool) *OptimizedStrategy {
	return &OptimizedStrategy{
		checksumValidator: checksumValidator,
		isChecksum:        isChecksum,
	}
}

// Validate performs optimized validation with minimal allocations
func (os *OptimizedStrategy) Validate(address, prefix, suffix string) (bool, error) {
	// Inline basic checks for performance
	if len(prefix) > len(address) || len(suffix) > len(address) {
		return false, nil
	}

	// Fast path: if both prefix and suffix are empty, return true
	if len(prefix) == 0 && len(suffix) == 0 {
		return true, nil
	}

	if !os.isChecksum {
		// Case-insensitive comparison with minimal allocations
		return os.validateCaseInsensitive(address, prefix, suffix), nil
	}

	// Checksum validation
	return os.validateChecksum(address, prefix, suffix), nil
}

// validateCaseInsensitive performs optimized case-insensitive validation
func (os *OptimizedStrategy) validateCaseInsensitive(address, prefix, suffix string) bool {
	// Check prefix
	for i := 0; i < len(prefix); i++ {
		addrChar := address[i]
		prefixChar := prefix[i]

		// Convert to lowercase for comparison
		if addrChar >= 'A' && addrChar <= 'F' {
			addrChar += 32
		}
		if prefixChar >= 'A' && prefixChar <= 'F' {
			prefixChar += 32
		}

		if addrChar != prefixChar {
			return false
		}
	}

	// Check suffix
	for i := 0; i < len(suffix); i++ {
		addrIndex := len(address) - len(suffix) + i
		addrChar := address[addrIndex]
		suffixChar := suffix[i]

		// Convert to lowercase for comparison
		if addrChar >= 'A' && addrChar <= 'F' {
			addrChar += 32
		}
		if suffixChar >= 'A' && suffixChar <= 'F' {
			suffixChar += 32
		}

		if addrChar != suffixChar {
			return false
		}
	}

	return true
}

// validateChecksum performs optimized checksum validation
func (os *OptimizedStrategy) validateChecksum(address, prefix, suffix string) bool {
	// First do case-insensitive check
	if !os.validateCaseInsensitive(address, prefix, suffix) {
		return false
	}

	// Then validate checksum using optimized method
	return os.checksumValidator.OptimizedChecksumValidation(address, prefix, suffix)
}

// Name returns the strategy name
func (os *OptimizedStrategy) Name() string {
	if os.isChecksum {
		return "optimized_checksum"
	}
	return "optimized_case_insensitive"
}

// Description returns the strategy description
func (os *OptimizedStrategy) Description() string {
	if os.isChecksum {
		return "High-performance checksum validation for worker threads"
	}
	return "High-performance case-insensitive validation for worker threads"
}

// GetAvailableStrategies returns a list of all available validation strategies
func GetAvailableStrategies(checksumValidator *crypto.ChecksumValidator) []ValidationStrategy {
	return []ValidationStrategy{
		NewCaseInsensitiveStrategy(),
		NewChecksumStrategy(checksumValidator),
		NewExactMatchStrategy(),
		NewOptimizedStrategy(checksumValidator, false),
		NewOptimizedStrategy(checksumValidator, true),
	}
}
