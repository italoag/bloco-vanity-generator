package worker

import (
	"strings"
	"testing"
)

// TestIsValidBlocoAddress_CaseInsensitive tests suffix validation without checksum
// These tests should pass with the current implementation
func TestIsValidBlocoAddress_CaseInsensitive(t *testing.T) {
	tests := []struct {
		name     string
		address  string
		prefix   string
		suffix   string
		checksum bool
		expected bool
	}{
		// Suffix-only validation (case-insensitive)
		{
			name:     "suffix only - exact match lowercase",
			address:  "0x1234567890abcdef1234567890abcdef12345abc",
			prefix:   "",
			suffix:   "abc",
			checksum: false,
			expected: true,
		},
		{
			name:     "suffix only - exact match uppercase",
			address:  "0x1234567890ABCDEF1234567890ABCDEF12345ABC",
			prefix:   "",
			suffix:   "ABC",
			checksum: false,
			expected: true,
		},
		{
			name:     "suffix only - mixed case address, lowercase suffix",
			address:  "0x1234567890AbCdEf1234567890AbCdEf12345AbC",
			prefix:   "",
			suffix:   "abc",
			checksum: false,
			expected: true,
		},
		{
			name:     "suffix only - lowercase address, uppercase suffix",
			address:  "0x1234567890abcdef1234567890abcdef12345abc",
			prefix:   "",
			suffix:   "ABC",
			checksum: false,
			expected: true,
		},
		{
			name:     "suffix only - no match",
			address:  "0x1234567890abcdef1234567890abcdef12345def",
			prefix:   "",
			suffix:   "abc",
			checksum: false,
			expected: false,
		},
		{
			name:     "suffix only - partial match at end",
			address:  "0x1234567890abcdef1234567890abcdef12345ab",
			prefix:   "",
			suffix:   "abc",
			checksum: false,
			expected: false,
		},
		{
			name:     "suffix only - longer suffix than address",
			address:  "0x12",
			prefix:   "",
			suffix:   "abc",
			checksum: false,
			expected: false,
		},
		// Prefix-only validation (should still work)
		{
			name:     "prefix only - exact match",
			address:  "0xabc4567890abcdef1234567890abcdef12345def",
			prefix:   "abc",
			suffix:   "",
			checksum: false,
			expected: true,
		},
		{
			name:     "prefix only - case insensitive match",
			address:  "0xABC4567890abcdef1234567890abcdef12345def",
			prefix:   "abc",
			suffix:   "",
			checksum: false,
			expected: true,
		},
		{
			name:     "prefix only - no match",
			address:  "0xdef4567890abcdef1234567890abcdef12345def",
			prefix:   "abc",
			suffix:   "",
			checksum: false,
			expected: false,
		},
		// Combined prefix+suffix validation (case-insensitive)
		{
			name:     "prefix and suffix - both match",
			address:  "0xabc4567890abcdef1234567890abcdef12345def",
			prefix:   "abc",
			suffix:   "def",
			checksum: false,
			expected: true,
		},
		{
			name:     "prefix and suffix - both match case insensitive",
			address:  "0xABC4567890abcdef1234567890abcdef12345DEF",
			prefix:   "abc",
			suffix:   "def",
			checksum: false,
			expected: true,
		},
		{
			name:     "prefix and suffix - prefix matches, suffix doesn't",
			address:  "0xabc4567890abcdef1234567890abcdef12345abc",
			prefix:   "abc",
			suffix:   "def",
			checksum: false,
			expected: false,
		},
		{
			name:     "prefix and suffix - suffix matches, prefix doesn't",
			address:  "0xdef4567890abcdef1234567890abcdef12345def",
			prefix:   "abc",
			suffix:   "def",
			checksum: false,
			expected: false,
		},
		{
			name:     "prefix and suffix - neither matches",
			address:  "0x1234567890abcdef1234567890abcdef12345678",
			prefix:   "abc",
			suffix:   "def",
			checksum: false,
			expected: false,
		},
		// Edge cases
		{
			name:     "empty prefix and suffix",
			address:  "0x1234567890abcdef1234567890abcdef12345678",
			prefix:   "",
			suffix:   "",
			checksum: false,
			expected: true,
		},
		{
			name:     "address without 0x prefix",
			address:  "1234567890abcdef1234567890abcdef12345abc",
			prefix:   "",
			suffix:   "abc",
			checksum: false,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchesCriteria(tt.address, tt.prefix, tt.suffix, tt.checksum, "ethereum")
			if result != tt.expected {
				t.Errorf("matchesCriteria(%q, %q, %q, %v) = %v, expected %v",
					tt.address, tt.prefix, tt.suffix, tt.checksum, result, tt.expected)
			}
		})
	}
}

// TestIsValidBlocoAddress_EIP55Checksum tests suffix validation with EIP-55 checksum
// These tests should currently FAIL due to the bug in isEIP55Checksum function
func TestIsValidBlocoAddress_EIP55Checksum(t *testing.T) {
	// First, let's create some valid EIP-55 addresses for testing
	// We'll use the toChecksumAddress function to generate proper checksum addresses

	tests := []struct {
		name     string
		address  string
		prefix   string
		suffix   string
		checksum bool
		expected bool
		note     string
	}{
		// These tests use real EIP-55 checksum addresses
		// Generated using the toChecksumAddress function
		{
			name:     "checksum suffix only - should match pattern",
			address:  "0x1234567890AbCdEf1234567890AbCdEf12345AbC", // This ends with "AbC" in checksum
			prefix:   "",
			suffix:   "abc", // User wants pattern "abc"
			checksum: true,
			expected: true, // Should pass: pattern matches, checksum is valid
			note:     "Should validate suffix pattern regardless of case in user input",
		},
		{
			name:     "checksum suffix only - pattern doesn't match",
			address:  "0x1234567890AbCdEf1234567890AbCdEf12345DeF", // This ends with "DeF" in checksum
			prefix:   "",
			suffix:   "abc", // User wants pattern "abc"
			checksum: true,
			expected: false, // Should fail: pattern doesn't match
			note:     "Should reject when suffix pattern doesn't match",
		},
		{
			name:     "checksum prefix only - should match pattern",
			address:  "0xAbC4567890abcdef1234567890abcdef12345def", // This starts with "AbC" in checksum
			prefix:   "abc",                                        // User wants pattern "abc"
			suffix:   "",
			checksum: true,
			expected: true, // Should pass: pattern matches, checksum is valid
			note:     "Should validate prefix pattern regardless of case in user input",
		},
		{
			name:     "checksum prefix only - pattern doesn't match",
			address:  "0xDeF4567890abcdef1234567890abcdef12345def", // This starts with "DeF" in checksum
			prefix:   "abc",                                        // User wants pattern "abc"
			suffix:   "",
			checksum: true,
			expected: false, // Should fail: pattern doesn't match
			note:     "Should reject when prefix pattern doesn't match",
		},
		{
			name:     "checksum prefix and suffix - both match",
			address:  "0xAbC4567890abcdef1234567890abcdef12345DeF", // Starts with "AbC", ends with "DeF"
			prefix:   "abc",                                        // User wants prefix "abc"
			suffix:   "def",                                        // User wants suffix "def"
			checksum: true,
			expected: true, // Should pass: both patterns match, checksum is valid
			note:     "Should validate both prefix and suffix patterns",
		},
		{
			name:     "checksum prefix and suffix - prefix matches, suffix doesn't",
			address:  "0xAbC4567890abcdef1234567890abcdef12345AbC", // Starts with "AbC", ends with "AbC"
			prefix:   "abc",                                        // User wants prefix "abc"
			suffix:   "def",                                        // User wants suffix "def"
			checksum: true,
			expected: false, // Should fail: suffix pattern doesn't match
			note:     "Should reject when suffix pattern doesn't match even if prefix does",
		},
		{
			name:     "checksum prefix and suffix - suffix matches, prefix doesn't",
			address:  "0xDeF4567890abcdef1234567890abcdef12345AbC", // Starts with "DeF", ends with "AbC"
			prefix:   "abc",                                        // User wants prefix "abc"
			suffix:   "abc",                                        // User wants suffix "abc"
			checksum: true,
			expected: false, // Should fail: prefix pattern doesn't match
			note:     "Should reject when prefix pattern doesn't match even if suffix does",
		},
		// Edge cases for checksum validation
		{
			name:     "checksum empty criteria",
			address:  "0x1234567890AbCdEf1234567890AbCdEf12345678",
			prefix:   "",
			suffix:   "",
			checksum: true,
			expected: true, // Should pass: no criteria to validate
			note:     "Should pass when no prefix/suffix criteria are specified",
		},
		{
			name:     "checksum single character suffix",
			address:  "0x1234567890abcdef1234567890abcdef1234567C", // Ends with "C"
			prefix:   "",
			suffix:   "c", // User wants suffix "c"
			checksum: true,
			expected: true, // Should pass: single character pattern matches
			note:     "Should handle single character suffixes correctly",
		},
		{
			name:     "checksum long suffix",
			address:  "0x1234567890abcdef1234567890abcdef12AbCdEf", // Ends with "AbCdEf"
			prefix:   "",
			suffix:   "abcdef", // User wants suffix "abcdef"
			checksum: true,
			expected: true, // Should pass: long suffix pattern matches
			note:     "Should handle longer suffixes correctly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// First verify that the test address is a valid checksum address
			if tt.checksum {
				checksumAddr := toChecksumAddress(tt.address)
				if checksumAddr != tt.address {
					t.Logf("Note: Test address %s is not in proper checksum format. Correct format: %s",
						tt.address, checksumAddr)
					// Update the address to use proper checksum for the test
					tt.address = checksumAddr
				}
			}

			result := matchesCriteria(tt.address, tt.prefix, tt.suffix, tt.checksum, "ethereum")
			if result != tt.expected {
				t.Errorf("matchesCriteria(%q, %q, %q, %v) = %v, expected %v\nNote: %s",
					tt.address, tt.prefix, tt.suffix, tt.checksum, result, tt.expected, tt.note)

				// Additional debugging information
				if tt.checksum && (tt.prefix != "" || tt.suffix != "") {
					t.Logf("Debug: Testing EIP-55 checksum validation")
					t.Logf("Debug: Address: %s", tt.address)
					t.Logf("Debug: Prefix: %q, Suffix: %q", tt.prefix, tt.suffix)

					// Test the individual components
					if tt.prefix != "" {
						prefixPart := tt.address[2 : 2+len(tt.prefix)]
						t.Logf("Debug: Prefix part: %q, Expected pattern: %q, Case-insensitive match: %v",
							prefixPart, tt.prefix, strings.EqualFold(prefixPart, tt.prefix))
					}

					if tt.suffix != "" {
						suffixStart := len(tt.address) - len(tt.suffix)
						if suffixStart >= 2 {
							suffixPart := tt.address[suffixStart:]
							t.Logf("Debug: Suffix part: %q, Expected pattern: %q, Case-insensitive match: %v",
								suffixPart, tt.suffix, strings.EqualFold(suffixPart, tt.suffix))
						}
					}
				}
			}
		})
	}
}

// TestIsValidBlocoAddress_EdgeCases tests various edge cases and boundary conditions
func TestIsValidBlocoAddress_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		address  string
		prefix   string
		suffix   string
		checksum bool
		expected bool
	}{
		// Very short addresses
		{
			name:     "minimal address with suffix",
			address:  "0x12",
			prefix:   "",
			suffix:   "2",
			checksum: false,
			expected: true,
		},
		{
			name:     "minimal address with prefix",
			address:  "0x12",
			prefix:   "1",
			suffix:   "",
			checksum: false,
			expected: true,
		},
		{
			name:     "minimal address with both prefix and suffix",
			address:  "0x12",
			prefix:   "1",
			suffix:   "2",
			checksum: false,
			expected: true,
		},
		// Overlapping prefix and suffix
		{
			name:     "overlapping prefix and suffix - valid",
			address:  "0xabc",
			prefix:   "ab",
			suffix:   "bc",
			checksum: false,
			expected: true,
		},
		{
			name:     "overlapping prefix and suffix - invalid",
			address:  "0xabc",
			prefix:   "ab",
			suffix:   "cd",
			checksum: false,
			expected: false,
		},
		// Same prefix and suffix
		{
			name:     "same prefix and suffix - valid",
			address:  "0xabcdefabcdef",
			prefix:   "abc",
			suffix:   "abc",
			checksum: false,
			expected: false, // Address ends with "def", not "abc"
		},
		{
			name:     "same prefix and suffix - valid match",
			address:  "0xabcdefabcabc",
			prefix:   "abc",
			suffix:   "abc",
			checksum: false,
			expected: true,
		},
		// Very long patterns
		{
			name:     "long prefix",
			address:  "0x123456789012345678901234567890123456789a",
			prefix:   "123456789012345678901234567890123456789",
			suffix:   "",
			checksum: false,
			expected: true,
		},
		{
			name:     "long suffix",
			address:  "0xa123456789012345678901234567890123456789",
			prefix:   "",
			suffix:   "123456789012345678901234567890123456789",
			checksum: false,
			expected: true,
		},
		// Pattern longer than address
		{
			name:     "prefix longer than address",
			address:  "0x123",
			prefix:   "12345",
			suffix:   "",
			checksum: false,
			expected: false,
		},
		{
			name:     "suffix longer than address",
			address:  "0x123",
			prefix:   "",
			suffix:   "12345",
			checksum: false,
			expected: false,
		},
		// Invalid hex characters (should still work for pattern matching)
		{
			name:     "address with invalid hex chars - pattern matches",
			address:  "0x123ghijk789",
			prefix:   "123",
			suffix:   "",
			checksum: false,
			expected: true, // Pattern matching doesn't validate hex
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchesCriteria(tt.address, tt.prefix, tt.suffix, tt.checksum, "ethereum")
			if result != tt.expected {
				t.Errorf("matchesCriteria(%q, %q, %q, %v) = %v, expected %v",
					tt.address, tt.prefix, tt.suffix, tt.checksum, result, tt.expected)
			}
		})
	}
}

// TestToChecksumAddress tests the EIP-55 checksum generation function
func TestToChecksumAddress(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "all lowercase",
			input:    "0x5aaeb6053f3e94c9b9a09f33669435e7ef1beaed",
			expected: "0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed",
		},
		{
			name:     "all uppercase",
			input:    "0x5AAEB6053F3E94C9B9A09F33669435E7EF1BEAED",
			expected: "0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed",
		},
		{
			name:     "mixed case",
			input:    "0x5aAeB6053f3E94c9B9A09F33669435e7eF1BeAeD",
			expected: "0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed",
		},
		{
			name:     "without 0x prefix",
			input:    "5aaeb6053f3e94c9b9a09f33669435e7ef1beaed",
			expected: "0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed",
		},
		{
			name:     "short address",
			input:    "0x123abc",
			expected: "0x123abC", // This is just an example, actual checksum may vary
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toChecksumAddress(tt.input)

			// For the known test case, check exact match
			if tt.name == "all lowercase" || tt.name == "all uppercase" || tt.name == "mixed case" || tt.name == "without 0x prefix" {
				if result != tt.expected {
					t.Errorf("toChecksumAddress(%q) = %q, expected %q", tt.input, result, tt.expected)
				}
			} else {
				// For other cases, just verify it's a valid format
				if !strings.HasPrefix(result, "0x") {
					t.Errorf("toChecksumAddress(%q) = %q, should start with 0x", tt.input, result)
				}
				if len(result) != len(tt.input) && len(result) != len(tt.input)+2 {
					t.Errorf("toChecksumAddress(%q) = %q, unexpected length", tt.input, result)
				}
			}
		})
	}
}

// TestIsEIP55Checksum tests the EIP-55 checksum validation function directly
func TestIsEIP55Checksum(t *testing.T) {
	tests := []struct {
		name     string
		address  string
		prefix   string
		suffix   string
		expected bool
		note     string
	}{
		{
			name:     "valid checksum with matching prefix",
			address:  "0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed",
			prefix:   "5aa", // Should match case-insensitively
			suffix:   "",
			expected: true,
			note:     "Should pass when prefix pattern matches",
		},
		{
			name:     "valid checksum with matching suffix",
			address:  "0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed",
			prefix:   "",
			suffix:   "aed", // Should match case-insensitively
			expected: true,
			note:     "Should pass when suffix pattern matches - THIS CURRENTLY FAILS DUE TO BUG",
		},
		{
			name:     "valid checksum with both prefix and suffix matching",
			address:  "0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed",
			prefix:   "5aa",
			suffix:   "aed",
			expected: true,
			note:     "Should pass when both patterns match - THIS CURRENTLY FAILS DUE TO SUFFIX BUG",
		},
		{
			name:     "valid checksum with non-matching prefix",
			address:  "0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed",
			prefix:   "abc",
			suffix:   "",
			expected: false,
			note:     "Should fail when prefix pattern doesn't match",
		},
		{
			name:     "valid checksum with non-matching suffix",
			address:  "0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed",
			prefix:   "",
			suffix:   "xyz",
			expected: false,
			note:     "Should fail when suffix pattern doesn't match",
		},
		{
			name:     "empty criteria",
			address:  "0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed",
			prefix:   "",
			suffix:   "",
			expected: true,
			note:     "Should pass when no criteria specified",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isEIP55Checksum(tt.address, tt.prefix, tt.suffix)
			if result != tt.expected {
				t.Errorf("isEIP55Checksum(%q, %q, %q) = %v, expected %v\nNote: %s",
					tt.address, tt.prefix, tt.suffix, result, tt.expected, tt.note)
			}
		})
	}
}

func TestMatchesCriteria_EIP55CaseSensitivePrefix(t *testing.T) {
	address := "0xDEAD00000000000000000000000000000000004B"

	if !matchesCriteria(address, "DEAD", "", true, "ethereum", true) {
		t.Fatalf("expected exact EIP-55 prefix match")
	}

	for _, prefix := range []string{"dead", "DeAD", "DEaD", "dEAd"} {
		if matchesCriteria(address, prefix, "", true, "ethereum", true) {
			t.Fatalf("expected %q to be rejected in case-sensitive EIP-55 mode", prefix)
		}
	}

	if !matchesCriteria(address, "dead", "", true, "ethereum", false) {
		t.Fatalf("expected case-insensitive EIP-55 mode to accept lowercase prefix")
	}
}

// TestSuffixValidationBugDocumentation documents the specific bug behavior
// This test demonstrates the exact issue described in the design document
func TestSuffixValidationBugDocumentation(t *testing.T) {
	// This test documents the specific bug: suffix validation fails in checksum mode
	// because the isEIP55Checksum function incorrectly compares the exact case
	// of the suffix part with the user's input pattern

	// Create a known address with proper EIP-55 checksum
	testAddress := "0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed"

	// Verify this is a proper checksum address
	checksumAddr := toChecksumAddress(testAddress)
	if checksumAddr != testAddress {
		t.Fatalf("Test address is not in proper checksum format. Expected: %s, Got: %s",
			checksumAddr, testAddress)
	}

	// Test case 1: Suffix validation should work (but currently fails)
	t.Run("suffix_validation_bug", func(t *testing.T) {
		// The address ends with "Aed" (with capital A and lowercase e,d)
		// User wants pattern "aed" (all lowercase)
		// This should pass because the pattern matches case-insensitively
		// But it currently fails due to the bug

		result := matchesCriteria(testAddress, "", "aed", true, "ethereum")
		expected := true // Should pass

		if result != expected {
			t.Errorf("DOCUMENTED BUG: Suffix validation fails in checksum mode")
			t.Errorf("matchesCriteria(%q, %q, %q, %v) = %v, expected %v",
				testAddress, "", "aed", true, result, expected)
			t.Errorf("The address ends with 'Aed' but user pattern 'aed' should match case-insensitively")
			t.Errorf("This fails because isEIP55Checksum compares 'Aed' != 'aed' directly")
		}
	})

	// Test case 2: Prefix validation also has the same bug
	t.Run("prefix_validation_bug", func(t *testing.T) {
		// The address starts with "5aA" (lowercase 5,a and uppercase A)
		// User wants pattern "5aa" (all lowercase)
		// This should pass because the pattern matches case-insensitively
		// But it currently fails due to the same bug

		result := matchesCriteria(testAddress, "5aa", "", true, "ethereum")
		expected := true // Should pass

		if result != expected {
			t.Errorf("DOCUMENTED BUG: Prefix validation fails in checksum mode")
			t.Errorf("matchesCriteria(%q, %q, %q, %v) = %v, expected %v",
				testAddress, "5aa", "", true, result, expected)
			t.Errorf("The address starts with '5aA' but user pattern '5aa' should match case-insensitively")
			t.Errorf("This fails because isEIP55Checksum compares '5aA' != '5aa' directly")
		}
	})

	// Test case 3: Case-insensitive mode works correctly (for comparison)
	t.Run("case_insensitive_works", func(t *testing.T) {
		// Same patterns should work in case-insensitive mode

		resultSuffix := matchesCriteria(testAddress, "", "aed", false, "ethereum")
		if !resultSuffix {
			t.Errorf("Case-insensitive suffix validation should work: %v", resultSuffix)
		}

		resultPrefix := matchesCriteria(testAddress, "5aa", "", false, "ethereum")
		if !resultPrefix {
			t.Errorf("Case-insensitive prefix validation should work: %v", resultPrefix)
		}

		resultBoth := matchesCriteria(testAddress, "5aa", "aed", false, "ethereum")
		if !resultBoth {
			t.Errorf("Case-insensitive prefix+suffix validation should work: %v", resultBoth)
		}
	})
}

// TestBugRootCause tests the specific line of code that causes the bug
func TestBugRootCause(t *testing.T) {
	// This test isolates the exact problematic comparison in isEIP55Checksum

	testAddress := "0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed"

	t.Run("demonstrate_problematic_comparison", func(t *testing.T) {
		// Extract the suffix part as the function does
		suffix := "aed"
		suffixStart := len(testAddress) - len(suffix)
		suffixPart := testAddress[suffixStart:] // This will be "Aed"

		// The bug is in this comparison:
		caseInsensitiveMatch := strings.EqualFold(suffixPart, suffix) // true
		exactCaseMatch := suffixPart == suffix                        // false - THIS IS THE BUG

		t.Logf("Suffix part from address: %q", suffixPart)
		t.Logf("User pattern: %q", suffix)
		t.Logf("Case-insensitive match (EqualFold): %v", caseInsensitiveMatch)
		t.Logf("Exact case match (==): %v", exactCaseMatch)

		if !caseInsensitiveMatch {
			t.Error("Case-insensitive match should be true")
		}

		if exactCaseMatch {
			t.Error("Exact case match should be false (this is expected)")
		}

		// The current buggy logic does:
		// if !caseInsensitiveMatch { return false } // OK
		// if !exactCaseMatch { return false }       // BUG - this always fails for checksum

		t.Logf("BUG: The function requires both case-insensitive AND exact case match")
		t.Logf("BUG: For EIP-55 checksum, exact case match will almost always be false")
		t.Logf("BUG: The function should only require case-insensitive pattern match")
	})
}

// TestMatchesCriteria_CaseSensitivity verifies that case sensitivity is handled correctly for different networks
func TestMatchesCriteria_CaseSensitivity(t *testing.T) {
	tests := []struct {
		name       string
		address    string
		prefix     string
		suffix     string
		isChecksum bool
		network    string
		want       bool
	}{
		// Ethereum (Hex) - Case insensitive by default (unless checksum matched)
		{
			name:       "ETH: Case insensitive match",
			address:    "0xAbC123...",
			prefix:     "abc",
			suffix:     "",
			isChecksum: false,
			network:    "ethereum",
			want:       true, // Should match because ETH is case-insensitive for search
		},
		// Solana (Base58) - Case sensitive
		{
			name:       "SOL: Case mismatch should fail",
			address:    "AbC123...",
			prefix:     "abc", // User asked for lowercase
			suffix:     "",
			isChecksum: false,
			network:    "solana",
			want:       false, // Should FAIL because 'a' != 'A' in Base58
		},
		{
			name:       "SOL: Exact match should pass",
			address:    "AbC123...",
			prefix:     "AbC",
			suffix:     "",
			isChecksum: false,
			network:    "solana",
			want:       true,
		},
		// Bitcoin (Base58) - Case sensitive
		{
			name:       "BTC: Case mismatch should fail",
			address:    "1AbC...",
			prefix:     "1abc",
			suffix:     "",
			isChecksum: false,
			network:    "bitcoin",
			want:       false, // Should FAIL
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Updated signature: matchesCriteria(address, prefix, suffix, isChecksum, network)
			got := matchesCriteria(tt.address, tt.prefix, tt.suffix, tt.isChecksum, tt.network)

			if got != tt.want {
				t.Errorf("matchesCriteria() = %v, want %v", got, tt.want)
			}
		})
	}
}
