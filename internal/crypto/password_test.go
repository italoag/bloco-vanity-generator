package crypto

import (
	"errors"
	"regexp"
	"strings"
	"testing"
	"unicode"

	"github.com/tyler-smith/go-bip39/wordlists"
)

func TestNewPasswordGenerator(t *testing.T) {
	pg := NewPasswordGenerator()

	if pg.minLength != 12 {
		t.Errorf("Expected minimum length 12, got %d", pg.minLength)
	}

	expectedCharset := DefaultPasswordCharset()
	if pg.charset != expectedCharset {
		t.Errorf("Expected default charset, got different charset")
	}
}

func TestNewPasswordGeneratorWithConfig(t *testing.T) {
	tests := []struct {
		name      string
		minLength int
		expected  int
	}{
		{"valid_length", 15, 15},
		{"minimum_enforced", 8, 12}, // Should enforce minimum of 12
		{"zero_length", 0, 12},      // Should enforce minimum of 12
		{"negative_length", -5, 12}, // Should enforce minimum of 12
	}

	charset := DefaultPasswordCharset()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pg := NewPasswordGeneratorWithConfig(tt.minLength, charset)
			if pg.minLength != tt.expected {
				t.Errorf("Expected minimum length %d, got %d", tt.expected, pg.minLength)
			}
		})
	}
}

func TestGenerateSecurePassword(t *testing.T) {
	pg := NewPasswordGenerator()

	// Test basic password generation
	password, err := pg.GenerateSecurePassword()
	if err != nil {
		t.Fatalf("Failed to generate password: %v", err)
	}

	// Test password meets minimum length requirement (3.1)
	if len(password) < 12 {
		t.Errorf("Password length %d is less than minimum 12", len(password))
	}

	// Test password validation
	if err := pg.ValidatePassword(password); err != nil {
		t.Errorf("Generated password failed validation: %v", err)
	}
}

func TestGenerateSecurePassword_MultipleGenerations(t *testing.T) {
	pg := NewPasswordGenerator()

	// Generate multiple passwords and ensure they're different
	passwords := make(map[string]bool)
	const numPasswords = 100

	for i := 0; i < numPasswords; i++ {
		password, err := pg.GenerateSecurePassword()
		if err != nil {
			t.Fatalf("Failed to generate password %d: %v", i, err)
		}

		if passwords[password] {
			t.Errorf("Generated duplicate password: %s", password)
		}
		passwords[password] = true

		// Validate each password
		if err := pg.ValidatePassword(password); err != nil {
			t.Errorf("Generated password %d failed validation: %v", i, err)
		}
	}
}

func TestGenerateSecurePassword_ComplexityRequirements(t *testing.T) {
	pg := NewPasswordGenerator()

	// Generate multiple passwords and verify complexity
	for i := 0; i < 50; i++ {
		password, err := pg.GenerateSecurePassword()
		if err != nil {
			t.Fatalf("Failed to generate password %d: %v", i, err)
		}

		// Check all complexity requirements
		hasLower := false
		hasUpper := false
		hasNumber := false
		hasSpecial := false

		for _, char := range password {
			if strings.ContainsRune(pg.charset.Lowercase, char) {
				hasLower = true
			}
			if strings.ContainsRune(pg.charset.Uppercase, char) {
				hasUpper = true
			}
			if strings.ContainsRune(pg.charset.Numbers, char) {
				hasNumber = true
			}
			if strings.ContainsRune(pg.charset.Special, char) {
				hasSpecial = true
			}
		}

		// Requirement 3.2: at least one lowercase
		if !hasLower {
			t.Errorf("Password %d missing lowercase character: %s", i, password)
		}

		// Requirement 3.3: at least one uppercase
		if !hasUpper {
			t.Errorf("Password %d missing uppercase character: %s", i, password)
		}

		// Requirement 3.4: at least one number
		if !hasNumber {
			t.Errorf("Password %d missing number: %s", i, password)
		}

		// Requirement 3.5: at least one special character
		if !hasSpecial {
			t.Errorf("Password %d missing special character: %s", i, password)
		}
	}
}

func TestValidatePassword(t *testing.T) {
	pg := NewPasswordGenerator()

	tests := []struct {
		name     string
		password string
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "valid_password",
			password: "Abc123!@#def",
			wantErr:  false,
		},
		{
			name:     "too_short",
			password: "Abc123!",
			wantErr:  true,
			errMsg:   "must be at least 12 characters long",
		},
		{
			name:     "missing_lowercase",
			password: "ABC123!@#DEF",
			wantErr:  true,
			errMsg:   "must contain at least one: lowercase letter",
		},
		{
			name:     "missing_uppercase",
			password: "abc123!@#def",
			wantErr:  true,
			errMsg:   "must contain at least one: uppercase letter",
		},
		{
			name:     "missing_number",
			password: "Abc!@#DefGhi",
			wantErr:  true,
			errMsg:   "must contain at least one: number",
		},
		{
			name:     "missing_special",
			password: "Abc123DefGhi",
			wantErr:  true,
			errMsg:   "must contain at least one: special character",
		},
		{
			name:     "missing_multiple",
			password: "abcdefghijkl",
			wantErr:  true,
			errMsg:   "must contain at least one: uppercase letter, number, special character",
		},
		{
			name:     "exactly_minimum_length",
			password: "Abc123!@#def",
			wantErr:  false,
		},
		{
			name:     "longer_valid_password",
			password: "Abc123!@#DefGhi456$%^",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := pg.ValidatePassword(tt.password)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error for password %s, got nil", tt.password)
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Expected error message to contain %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for password %s, got %v", tt.password, err)
				}
			}
		})
	}
}

func TestPasswordGenerator_Getters(t *testing.T) {
	pg := NewPasswordGenerator()

	if pg.GetMinLength() != 12 {
		t.Errorf("Expected minimum length 12, got %d", pg.GetMinLength())
	}

	charset := pg.GetCharset()
	expected := DefaultPasswordCharset()
	if charset != expected {
		t.Errorf("Expected default charset, got different charset")
	}
}

func TestPasswordGenerator_SetMinLength(t *testing.T) {
	pg := NewPasswordGenerator()

	tests := []struct {
		name     string
		length   int
		expected int
	}{
		{"valid_length", 15, 15},
		{"minimum_enforced", 8, 12},
		{"zero_enforced", 0, 12},
		{"negative_enforced", -5, 12},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pg.SetMinLength(tt.length)
			if pg.GetMinLength() != tt.expected {
				t.Errorf("Expected minimum length %d, got %d", tt.expected, pg.GetMinLength())
			}
		})
	}
}

func TestPasswordGenerator_SetCharset(t *testing.T) {
	pg := NewPasswordGenerator()

	// Test valid charset
	validCharset := PasswordCharset{
		Lowercase: "abc",
		Uppercase: "ABC",
		Numbers:   "123",
		Special:   "!@#",
	}

	err := pg.SetCharset(validCharset)
	if err != nil {
		t.Errorf("Expected no error for valid charset, got %v", err)
	}

	if pg.GetCharset() != validCharset {
		t.Errorf("Charset not set correctly")
	}

	// Test invalid charsets
	invalidTests := []struct {
		name    string
		charset PasswordCharset
		errMsg  string
	}{
		{
			name: "empty_lowercase",
			charset: PasswordCharset{
				Lowercase: "",
				Uppercase: "ABC",
				Numbers:   "123",
				Special:   "!@#",
			},
			errMsg: "lowercase character set cannot be empty",
		},
		{
			name: "empty_uppercase",
			charset: PasswordCharset{
				Lowercase: "abc",
				Uppercase: "",
				Numbers:   "123",
				Special:   "!@#",
			},
			errMsg: "uppercase character set cannot be empty",
		},
		{
			name: "empty_numbers",
			charset: PasswordCharset{
				Lowercase: "abc",
				Uppercase: "ABC",
				Numbers:   "",
				Special:   "!@#",
			},
			errMsg: "numbers character set cannot be empty",
		},
		{
			name: "empty_special",
			charset: PasswordCharset{
				Lowercase: "abc",
				Uppercase: "ABC",
				Numbers:   "123",
				Special:   "",
			},
			errMsg: "special character set cannot be empty",
		},
	}

	for _, tt := range invalidTests {
		t.Run(tt.name, func(t *testing.T) {
			err := pg.SetCharset(tt.charset)
			if err == nil {
				t.Errorf("Expected error for %s, got nil", tt.name)
			} else if !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("Expected error message to contain %q, got %q", tt.errMsg, err.Error())
			}
		})
	}
}

func TestDefaultPasswordCharset(t *testing.T) {
	charset := DefaultPasswordCharset()

	// Test that all expected characters are present
	expectedLowercase := "abcdefghijklmnopqrstuvwxyz"
	expectedUppercase := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	expectedNumbers := "0123456789"
	expectedSpecial := "!@#$%^&*()_+-=[]{}|;:,.<>?"

	if charset.Lowercase != expectedLowercase {
		t.Errorf("Unexpected lowercase charset: %s", charset.Lowercase)
	}

	if charset.Uppercase != expectedUppercase {
		t.Errorf("Unexpected uppercase charset: %s", charset.Uppercase)
	}

	if charset.Numbers != expectedNumbers {
		t.Errorf("Unexpected numbers charset: %s", charset.Numbers)
	}

	if charset.Special != expectedSpecial {
		t.Errorf("Unexpected special charset: %s", charset.Special)
	}
}

// Test password entropy and randomness
func TestPasswordEntropy(t *testing.T) {
	pg := NewPasswordGenerator()

	// Generate many passwords and check for patterns
	passwords := make([]string, 1000)
	for i := 0; i < len(passwords); i++ {
		password, err := pg.GenerateSecurePassword()
		if err != nil {
			t.Fatalf("Failed to generate password %d: %v", i, err)
		}
		passwords[i] = password
	}

	// Check that first characters are well distributed
	firstChars := make(map[rune]int)
	for _, password := range passwords {
		if len(password) > 0 {
			firstChars[rune(password[0])]++
		}
	}

	// Should have reasonable distribution (not all passwords starting with same char)
	if len(firstChars) < 10 {
		t.Errorf("Poor distribution of first characters: only %d unique first chars", len(firstChars))
	}

	// Check that no single character dominates (> 50% of first positions)
	for char, count := range firstChars {
		if float64(count)/float64(len(passwords)) > 0.5 {
			t.Errorf("Character %c appears in %d%% of first positions, indicating poor randomness",
				char, (count*100)/len(passwords))
		}
	}
}

// Test that passwords don't have predictable patterns
func TestPasswordPatterns(t *testing.T) {
	pg := NewPasswordGenerator()

	passwords := make([]string, 100)
	for i := 0; i < len(passwords); i++ {
		password, err := pg.GenerateSecurePassword()
		if err != nil {
			t.Fatalf("Failed to generate password %d: %v", i, err)
		}
		passwords[i] = password
	}

	// Check that passwords don't all start with the same character type
	startsWithLower := 0
	startsWithUpper := 0
	startsWithNumber := 0
	startsWithSpecial := 0

	for _, password := range passwords {
		if len(password) > 0 {
			first := rune(password[0])
			if unicode.IsLower(first) {
				startsWithLower++
			} else if unicode.IsUpper(first) {
				startsWithUpper++
			} else if unicode.IsDigit(first) {
				startsWithNumber++
			} else {
				startsWithSpecial++
			}
		}
	}

	// Each type should appear at least once in first position (with high probability)
	if startsWithLower == 0 {
		t.Errorf("No passwords start with lowercase (possible but unlikely)")
	}
	if startsWithUpper == 0 {
		t.Errorf("No passwords start with uppercase (possible but unlikely)")
	}
	if startsWithNumber == 0 {
		t.Errorf("No passwords start with number (possible but unlikely)")
	}
	if startsWithSpecial == 0 {
		t.Errorf("No passwords start with special character (possible but unlikely)")
	}
}

// Benchmark password generation performance
func BenchmarkGenerateSecurePassword(b *testing.B) {
	pg := NewPasswordGenerator()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := pg.GenerateSecurePassword()
		if err != nil {
			b.Fatalf("Failed to generate password: %v", err)
		}
	}
}

// Benchmark password validation performance
func BenchmarkValidatePassword(b *testing.B) {
	pg := NewPasswordGenerator()

	// Generate a valid password once
	password, err := pg.GenerateSecurePassword()
	if err != nil {
		b.Fatalf("Failed to generate test password: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := pg.ValidatePassword(password)
		if err != nil {
			b.Fatalf("Password validation failed: %v", err)
		}
	}
}

// Test edge cases for password generation
func TestPasswordGeneration_EdgeCases(t *testing.T) {
	// Test with minimal character sets
	minimalCharset := PasswordCharset{
		Lowercase: "a",
		Uppercase: "A",
		Numbers:   "1",
		Special:   "!",
	}

	pg := NewPasswordGeneratorWithConfig(12, minimalCharset)

	password, err := pg.GenerateSecurePassword()
	if err != nil {
		t.Fatalf("Failed to generate password with minimal charset: %v", err)
	}

	// Should still meet all requirements
	if err := pg.ValidatePassword(password); err != nil {
		t.Errorf("Generated password with minimal charset failed validation: %v", err)
	}

	// Should contain all required character types
	if !strings.Contains(password, "a") {
		t.Errorf("Password missing required lowercase 'a': %s", password)
	}
	if !strings.Contains(password, "A") {
		t.Errorf("Password missing required uppercase 'A': %s", password)
	}
	if !strings.Contains(password, "1") {
		t.Errorf("Password missing required number '1': %s", password)
	}
	if !strings.Contains(password, "!") {
		t.Errorf("Password missing required special '!': %s", password)
	}
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) {
	return 0, errors.New("simulated entropy failure")
}

type zeroReader struct{}

func (zeroReader) Read(b []byte) (int, error) {
	for i := range b {
		b[i] = 0
	}
	return len(b), nil
}

func TestGenerateWordPasswordFormat(t *testing.T) {
	pg := NewPasswordGenerator()
	componentPattern := regexp.MustCompile(`^[A-Z][a-z]+[0-9]?$`)

	wordSet := make(map[string]bool, len(wordlists.English))
	for _, w := range wordlists.English {
		wordSet[w] = true
	}

	for i := 0; i < 100; i++ {
		password, err := pg.GenerateWordPassword()
		if err != nil {
			t.Fatalf("GenerateWordPassword failed: %v", err)
		}

		if len(password) < 12 {
			t.Fatalf("password too short: %d chars", len(password))
		}

		var separators []rune
		for _, r := range password {
			if strings.ContainsRune("+-_:", r) {
				separators = append(separators, r)
			}
		}
		if len(separators) != 2 || separators[0] != separators[1] {
			t.Fatalf("expected exactly two identical separators, got %d", len(separators))
		}

		components := strings.FieldsFunc(password, func(r rune) bool {
			return strings.ContainsRune("+-_:", r)
		})
		if len(components) != 3 {
			t.Fatalf("expected 3 word components, got %d", len(components))
		}

		digitCount := 0
		for _, component := range components {
			if !componentPattern.MatchString(component) {
				t.Fatalf("component does not match expected word format")
			}
			last := component[len(component)-1]
			base := component
			if last >= '0' && last <= '9' {
				digitCount++
				base = component[:len(component)-1]
			}
			if !wordSet[strings.ToLower(base)] {
				t.Fatalf("component is not in the BIP-39 English wordlist")
			}
		}
		if digitCount < 1 || digitCount > 3 {
			t.Fatalf("expected 1..3 digit suffixes, got %d", digitCount)
		}
	}
}

func TestGenerateWordPasswordEntropyFailure(t *testing.T) {
	password, err := generateWordPassword(failingReader{})
	if err == nil {
		t.Fatal("expected error when entropy source fails, got nil")
	}
	if password != "" {
		t.Fatal("expected empty password on failure")
	}
}

func TestGenerateWordPasswordDeterministicZeroReader(t *testing.T) {
	password, err := generateWordPassword(zeroReader{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if password != "Abandon0+Abandon+Abandon" {
		t.Fatalf("unexpected deterministic password: %q", password)
	}
}
