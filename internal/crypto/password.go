package crypto

import (
	"bloco-vgen/pkg/errors"
	"crypto/rand"
	"fmt"
	"strings"
)

// PasswordCharset defines the character sets used for password generation
type PasswordCharset struct {
	Lowercase string
	Uppercase string
	Numbers   string
	Special   string
}

// DefaultPasswordCharset returns the default character set for password generation
func DefaultPasswordCharset() PasswordCharset {
	return PasswordCharset{
		Lowercase: "abcdefghijklmnopqrstuvwxyz",
		Uppercase: "ABCDEFGHIJKLMNOPQRSTUVWXYZ",
		Numbers:   "0123456789",
		Special:   "!@#$%^&*()_+-=[]{}|;:,.<>?",
	}
}

// PasswordGenerator handles secure password generation with complexity validation
type PasswordGenerator struct {
	minLength int
	charset   PasswordCharset
}

// NewPasswordGenerator creates a new password generator with default settings
func NewPasswordGenerator() *PasswordGenerator {
	return &PasswordGenerator{
		minLength: 12, // Requirement 3.1: minimum 12 characters
		charset:   DefaultPasswordCharset(),
	}
}

// NewPasswordGeneratorWithConfig creates a password generator with custom configuration
func NewPasswordGeneratorWithConfig(minLength int, charset PasswordCharset) *PasswordGenerator {
	if minLength < 12 {
		minLength = 12 // Enforce minimum requirement
	}
	return &PasswordGenerator{
		minLength: minLength,
		charset:   charset,
	}
}

// GenerateSecurePassword generates a cryptographically secure password
// that meets all complexity requirements
func (pg *PasswordGenerator) GenerateSecurePassword() (string, error) {
	// Calculate minimum length to ensure all character types are included
	minRequiredLength := 4 // At least one from each character set
	if pg.minLength < minRequiredLength {
		return "", errors.NewValidationError("generate_secure_password",
			"minimum length must be at least 4 to include all character types")
	}

	// Create combined character set for random selection
	allChars := pg.charset.Lowercase + pg.charset.Uppercase + pg.charset.Numbers + pg.charset.Special
	if len(allChars) == 0 {
		return "", errors.NewValidationError("generate_secure_password",
			"character set cannot be empty")
	}

	// Generate password with guaranteed complexity
	password := make([]byte, pg.minLength)

	// Ensure at least one character from each required set
	// Requirement 3.2: at least one lowercase
	if err := pg.setRandomCharFromSet(password, 0, pg.charset.Lowercase); err != nil {
		return "", errors.WrapError(err, errors.ErrorTypeCrypto,
			"generate_secure_password", "failed to add lowercase character")
	}

	// Requirement 3.3: at least one uppercase
	if err := pg.setRandomCharFromSet(password, 1, pg.charset.Uppercase); err != nil {
		return "", errors.WrapError(err, errors.ErrorTypeCrypto,
			"generate_secure_password", "failed to add uppercase character")
	}

	// Requirement 3.4: at least one number
	if err := pg.setRandomCharFromSet(password, 2, pg.charset.Numbers); err != nil {
		return "", errors.WrapError(err, errors.ErrorTypeCrypto,
			"generate_secure_password", "failed to add number character")
	}

	// Requirement 3.5: at least one special character
	if err := pg.setRandomCharFromSet(password, 3, pg.charset.Special); err != nil {
		return "", errors.WrapError(err, errors.ErrorTypeCrypto,
			"generate_secure_password", "failed to add special character")
	}

	// Fill remaining positions with random characters from all sets
	for i := 4; i < len(password); i++ {
		if err := pg.setRandomCharFromSet(password, i, allChars); err != nil {
			return "", errors.WrapError(err, errors.ErrorTypeCrypto,
				"generate_secure_password", "failed to add random character")
		}
	}

	// Shuffle the password to avoid predictable patterns
	if err := pg.shufflePassword(password); err != nil {
		return "", errors.WrapError(err, errors.ErrorTypeCrypto,
			"generate_secure_password", "failed to shuffle password")
	}

	result := string(password)

	// Validate the generated password meets all requirements
	if err := pg.ValidatePassword(result); err != nil {
		return "", errors.WrapError(err, errors.ErrorTypeValidation,
			"generate_secure_password", "generated password failed validation")
	}

	return result, nil
}

// setRandomCharFromSet sets a random character from the given set at the specified position
func (pg *PasswordGenerator) setRandomCharFromSet(password []byte, position int, charset string) error {
	if len(charset) == 0 {
		return errors.NewValidationError("set_random_char_from_set",
			"character set cannot be empty")
	}

	// Generate cryptographically secure random index
	// Requirement 3.6: use crypto/rand for cryptographically secure generation
	randomBytes := make([]byte, 1)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return errors.NewCryptoError("set_random_char_from_set",
			"failed to generate random bytes", err)
	}

	// Use rejection sampling to avoid modulo bias
	maxValid := 256 - (256 % len(charset))
	for int(randomBytes[0]) >= maxValid {
		_, err := rand.Read(randomBytes)
		if err != nil {
			return errors.NewCryptoError("set_random_char_from_set",
				"failed to generate random bytes in rejection sampling", err)
		}
	}

	index := int(randomBytes[0]) % len(charset)
	password[position] = charset[index]
	return nil
}

// shufflePassword shuffles the password bytes using Fisher-Yates algorithm
func (pg *PasswordGenerator) shufflePassword(password []byte) error {
	n := len(password)
	for i := n - 1; i > 0; i-- {
		// Generate random index from 0 to i (inclusive)
		randomBytes := make([]byte, 1)
		_, err := rand.Read(randomBytes)
		if err != nil {
			return errors.NewCryptoError("shuffle_password",
				"failed to generate random bytes for shuffling", err)
		}

		// Use rejection sampling for uniform distribution
		maxValid := 256 - (256 % (i + 1))
		for int(randomBytes[0]) >= maxValid {
			_, err := rand.Read(randomBytes)
			if err != nil {
				return errors.NewCryptoError("shuffle_password",
					"failed to generate random bytes in rejection sampling", err)
			}
		}

		j := int(randomBytes[0]) % (i + 1)
		password[i], password[j] = password[j], password[i]
	}
	return nil
}

// ValidatePassword validates that a password meets all complexity requirements
func (pg *PasswordGenerator) ValidatePassword(password string) error {
	// Requirement 3.1: minimum 12 characters
	if len(password) < pg.minLength {
		return errors.NewValidationError("validate_password",
			fmt.Sprintf("password must be at least %d characters long, got %d",
				pg.minLength, len(password)))
	}

	// Check for required character types
	var hasLower, hasUpper, hasNumber, hasSpecial bool

	for _, char := range password {
		// Requirement 3.2: at least one lowercase
		if strings.ContainsRune(pg.charset.Lowercase, char) {
			hasLower = true
		}

		// Requirement 3.3: at least one uppercase
		if strings.ContainsRune(pg.charset.Uppercase, char) {
			hasUpper = true
		}

		// Requirement 3.4: at least one number
		if strings.ContainsRune(pg.charset.Numbers, char) {
			hasNumber = true
		}

		// Requirement 3.5: at least one special character
		if strings.ContainsRune(pg.charset.Special, char) {
			hasSpecial = true
		}
	}

	// Collect missing requirements
	var missing []string
	if !hasLower {
		missing = append(missing, "lowercase letter")
	}
	if !hasUpper {
		missing = append(missing, "uppercase letter")
	}
	if !hasNumber {
		missing = append(missing, "number")
	}
	if !hasSpecial {
		missing = append(missing, "special character")
	}

	if len(missing) > 0 {
		return errors.NewValidationError("validate_password",
			fmt.Sprintf("password must contain at least one: %s",
				strings.Join(missing, ", ")))
	}

	return nil
}

// GetMinLength returns the minimum password length
func (pg *PasswordGenerator) GetMinLength() int {
	return pg.minLength
}

// GetCharset returns the character set used for password generation
func (pg *PasswordGenerator) GetCharset() PasswordCharset {
	return pg.charset
}

// SetMinLength sets the minimum password length (minimum 12)
func (pg *PasswordGenerator) SetMinLength(length int) {
	if length < 12 {
		length = 12 // Enforce minimum requirement
	}
	pg.minLength = length
}

// SetCharset sets the character set for password generation
func (pg *PasswordGenerator) SetCharset(charset PasswordCharset) error {
	// Validate that all character sets are non-empty
	if len(charset.Lowercase) == 0 {
		return errors.NewValidationError("set_charset",
			"lowercase character set cannot be empty")
	}
	if len(charset.Uppercase) == 0 {
		return errors.NewValidationError("set_charset",
			"uppercase character set cannot be empty")
	}
	if len(charset.Numbers) == 0 {
		return errors.NewValidationError("set_charset",
			"numbers character set cannot be empty")
	}
	if len(charset.Special) == 0 {
		return errors.NewValidationError("set_charset",
			"special character set cannot be empty")
	}

	pg.charset = charset
	return nil
}
