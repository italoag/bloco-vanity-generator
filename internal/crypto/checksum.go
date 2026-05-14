package crypto

import (
	"strconv"
	"strings"

	"bloco-vgen/pkg/errors"
)

// ChecksumValidator handles EIP-55 checksum validation for Ethereum addresses
type ChecksumValidator struct {
	poolManager *PoolManager
}

// NewChecksumValidator creates a new checksum validator
func NewChecksumValidator(poolManager *PoolManager) *ChecksumValidator {
	return &ChecksumValidator{
		poolManager: poolManager,
	}
}

// ToChecksumAddress converts an address to EIP-55 checksum format
func (cv *ChecksumValidator) ToChecksumAddress(address string) (string, error) {
	if len(address) != 40 {
		return "", errors.NewValidationError("to_checksum_address",
			"address must be 40 characters long")
	}

	// Get objects from pools
	hasherPool := cv.poolManager.GetHasherPool()
	bufferPool := cv.poolManager.GetBufferPool()

	hasher := hasherPool.GetKeccak()
	hashSB := bufferPool.GetStringBuilder()
	resultSB := bufferPool.GetStringBuilder()
	addressBytes := bufferPool.GetByteBuffer()

	defer func() {
		hasherPool.PutKeccak(hasher)
		bufferPool.PutStringBuilder(hashSB)
		bufferPool.PutStringBuilder(resultSB)
		bufferPool.PutByteBuffer(addressBytes)
	}()

	// Convert address to lowercase bytes
	addressLower := strings.ToLower(address)
	addressBytes = append(addressBytes, addressLower...)

	// Calculate Keccak256 hash of the lowercase address
	hasher.Reset()
	hasher.Write(addressBytes)

	// Use a fixed-size array for hash bytes to avoid heap allocation
	var hashBytes [32]byte
	hasher.Sum(hashBytes[:0])

	// Pre-allocate capacity in string builders
	hashSB.Grow(64)   // 32 bytes * 2 hex chars per byte
	resultSB.Grow(40) // Address length

	// Convert hash to hex string using string builder and lookup table for efficiency
	const hexChars = "0123456789abcdef"
	for _, b := range hashBytes {
		hashSB.WriteByte(hexChars[b>>4])
		hashSB.WriteByte(hexChars[b&0x0f])
	}
	hash := hashSB.String()

	// Build result using string builder for efficiency
	for i, char := range addressLower {
		// Convert hex char to int directly without ParseInt
		var hashChar int64
		hexChar := hash[i]
		if hexChar >= '0' && hexChar <= '9' {
			hashChar = int64(hexChar - '0')
		} else if hexChar >= 'a' && hexChar <= 'f' {
			hashChar = int64(hexChar - 'a' + 10)
		}

		if hashChar >= 8 {
			// Convert to uppercase if needed
			if char >= 'a' && char <= 'f' {
				resultSB.WriteByte(byte(char) - 32) // 'a' to 'A' ASCII difference
			} else {
				resultSB.WriteByte(byte(char))
			}
		} else {
			// Keep lowercase
			resultSB.WriteByte(byte(char))
		}
	}

	return resultSB.String(), nil
}

// ValidateChecksum validates the EIP-55 checksum of an address
func (cv *ChecksumValidator) ValidateChecksum(address string) (bool, error) {
	if len(address) != 40 {
		return false, errors.NewValidationError("validate_checksum",
			"address must be 40 characters long")
	}

	// Generate the correct checksum address
	checksumAddress, err := cv.ToChecksumAddress(address)
	if err != nil {
		return false, errors.WrapError(err, errors.ErrorTypeValidation,
			"validate_checksum", "failed to generate checksum address")
	}

	// Compare with the provided address
	return address == checksumAddress, nil
}

// ValidatePatternChecksum validates checksum for specific prefix/suffix patterns
func (cv *ChecksumValidator) ValidatePatternChecksum(address, prefix, suffix string) (bool, error) {
	if len(address) != 40 {
		return false, errors.NewValidationError("validate_pattern_checksum",
			"address must be 40 characters long")
	}

	// Get objects from pools
	hasherPool := cv.poolManager.GetHasherPool()
	bufferPool := cv.poolManager.GetBufferPool()

	hasher := hasherPool.GetKeccak()
	sb := bufferPool.GetStringBuilder()

	defer func() {
		hasherPool.PutKeccak(hasher)
		bufferPool.PutStringBuilder(sb)
	}()

	// Calculate Keccak256 hash of the address
	hasher.Write([]byte(strings.ToLower(address)))
	hashBytes := hasher.Sum(nil)

	// Convert hash to hex string using string builder for efficiency
	sb.Grow(64) // Pre-allocate capacity
	for _, b := range hashBytes {
		sb.WriteByte("0123456789abcdef"[b>>4])
		sb.WriteByte("0123456789abcdef"[b&0x0f])
	}
	hash := sb.String()

	// Check prefix checksum
	for i := 0; i < len(prefix); i++ {
		hashChar, err := strconv.ParseInt(string(hash[i]), 16, 64)
		if err != nil {
			return false, errors.NewValidationError("validate_pattern_checksum",
				"invalid hash character")
		}

		expectedChar := string(address[i])
		if hashChar >= 8 {
			expectedChar = strings.ToUpper(expectedChar)
		} else {
			expectedChar = strings.ToLower(expectedChar)
		}

		if string(prefix[i]) != expectedChar {
			return false, nil
		}
	}

	// Check suffix checksum
	for i := 0; i < len(suffix); i++ {
		j := i + 40 - len(suffix)
		hashChar, err := strconv.ParseInt(string(hash[j]), 16, 64)
		if err != nil {
			return false, errors.NewValidationError("validate_pattern_checksum",
				"invalid hash character")
		}

		expectedChar := string(address[j])
		if hashChar >= 8 {
			expectedChar = strings.ToUpper(expectedChar)
		} else {
			expectedChar = strings.ToLower(expectedChar)
		}

		if string(suffix[i]) != expectedChar {
			return false, nil
		}
	}

	return true, nil
}

// OptimizedChecksumValidation performs optimized checksum validation for high-throughput scenarios
func (cv *ChecksumValidator) OptimizedChecksumValidation(address, prefix, suffix string) bool {
	// Fast path validation without error handling for performance-critical code
	if len(address) != 40 {
		return false
	}

	// Get objects from pools
	hasherPool := cv.poolManager.GetHasherPool()
	hasher := hasherPool.GetKeccak()
	defer hasherPool.PutKeccak(hasher)

	// Calculate hash directly
	hasher.Write([]byte(strings.ToLower(address)))
	hashBytes := hasher.Sum(nil)

	// Check prefix checksum with direct byte operations
	for i := 0; i < len(prefix); i++ {
		// Get hash nibble for this position
		byteIndex := i / 2
		nibbleIndex := i % 2
		var hashNibble byte
		if nibbleIndex == 0 {
			hashNibble = hashBytes[byteIndex] >> 4
		} else {
			hashNibble = hashBytes[byteIndex] & 0x0f
		}

		// Check if character case matches hash
		addrChar := address[i]
		prefixChar := prefix[i]

		var expectedChar byte
		if hashNibble >= 8 {
			// Should be uppercase
			if addrChar >= 'a' && addrChar <= 'f' {
				expectedChar = addrChar - 32
			} else {
				expectedChar = addrChar
			}
		} else {
			// Should be lowercase
			if addrChar >= 'A' && addrChar <= 'F' {
				expectedChar = addrChar + 32
			} else {
				expectedChar = addrChar
			}
		}

		if prefixChar != expectedChar {
			return false
		}
	}

	// Check suffix checksum
	for i := 0; i < len(suffix); i++ {
		j := i + 40 - len(suffix)

		// Get hash nibble for this position
		byteIndex := j / 2
		nibbleIndex := j % 2
		var hashNibble byte
		if nibbleIndex == 0 {
			hashNibble = hashBytes[byteIndex] >> 4
		} else {
			hashNibble = hashBytes[byteIndex] & 0x0f
		}

		// Check if character case matches hash
		addrChar := address[j]
		suffixChar := suffix[i]

		var expectedChar byte
		if hashNibble >= 8 {
			// Should be uppercase
			if addrChar >= 'a' && addrChar <= 'f' {
				expectedChar = addrChar - 32
			} else {
				expectedChar = addrChar
			}
		} else {
			// Should be lowercase
			if addrChar >= 'A' && addrChar <= 'F' {
				expectedChar = addrChar + 32
			} else {
				expectedChar = addrChar
			}
		}

		if suffixChar != expectedChar {
			return false
		}
	}

	return true
}

// IsValidHexAddress checks if an address contains only valid hex characters
func (cv *ChecksumValidator) IsValidHexAddress(address string) bool {
	if len(address) != 40 {
		return false
	}

	for _, char := range address {
		if (char < '0' || char > '9') &&
			(char < 'a' || char > 'f') &&
			(char < 'A' || char > 'F') {
			return false
		}
	}
	return true
}

// GetChecksumErrors returns detailed information about checksum validation errors
func (cv *ChecksumValidator) GetChecksumErrors(address string) []ChecksumError {
	var errors []ChecksumError

	if len(address) != 40 {
		errors = append(errors, ChecksumError{
			Position: -1,
			Message:  "address must be 40 characters long",
		})
		return errors
	}

	checksumAddress, err := cv.ToChecksumAddress(address)
	if err != nil {
		errors = append(errors, ChecksumError{
			Position: -1,
			Message:  "failed to calculate checksum: " + err.Error(),
		})
		return errors
	}

	// Compare character by character
	for i, char := range address {
		expectedChar := rune(checksumAddress[i])
		if char != expectedChar {
			errors = append(errors, ChecksumError{
				Position:     i,
				ActualChar:   string(char),
				ExpectedChar: string(expectedChar),
				Message:      "checksum mismatch",
			})
		}
	}

	return errors
}

// ChecksumError represents a checksum validation error
type ChecksumError struct {
	Position     int    `json:"position"`
	ActualChar   string `json:"actual_char,omitempty"`
	ExpectedChar string `json:"expected_char,omitempty"`
	Message      string `json:"message"`
}
