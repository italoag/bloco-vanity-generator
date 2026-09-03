package crypto

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// bytes32 returns a 32-byte slice filled with the given byte value.
func bytes32(v byte) []byte {
	b := make([]byte, 32)
	for i := range b {
		b[i] = v
	}
	return b
}

// bytes16 returns a 16-byte slice filled with the given byte value.
func bytes16(v byte) []byte {
	b := make([]byte, 16)
	for i := range b {
		b[i] = v
	}
	return b
}

func TestNewKeyStoreV3(t *testing.T) {
	tests := []struct {
		name     string
		address  string
		expected string
	}{
		{
			name:     "address with 0x prefix",
			address:  "0x1234567890AbCdEf1234567890AbCdEf12345678",
			expected: "1234567890abcdef1234567890abcdef12345678",
		},
		{
			name:     "address without 0x prefix",
			address:  "1234567890AbCdEf1234567890AbCdEf12345678",
			expected: "1234567890abcdef1234567890abcdef12345678",
		},
		{
			name:     "lowercase address",
			address:  "0x1234567890abcdef1234567890abcdef12345678",
			expected: "1234567890abcdef1234567890abcdef12345678",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ks := NewKeyStoreV3(tt.address, "ethereum")

			if ks.Address != tt.expected {
				t.Errorf("Expected address %s, got %s", tt.expected, ks.Address)
			}

			if ks.Version != 3 {
				t.Errorf("Expected version 3, got %d", ks.Version)
			}

			if ks.Crypto.Cipher != "aes-128-ctr" {
				t.Errorf("Expected cipher aes-128-ctr, got %s", ks.Crypto.Cipher)
			}

			if ks.Crypto.KDF != "scrypt" {
				t.Errorf("Expected KDF scrypt, got %s", ks.Crypto.KDF)
			}

			if ks.ID == "" {
				t.Error("Expected non-empty ID")
			}

			// Validate UUID format
			if _, err := uuid.Parse(ks.ID); err != nil {
				t.Errorf("Expected valid UUID, got %s: %v", ks.ID, err)
			}
		})
	}
}

func TestKeyStoreV3_SetScryptParams(t *testing.T) {
	ks := NewKeyStoreV3("0x1234567890abcdef1234567890abcdef12345678", "ethereum")
	salt := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}

	ks.SetScryptParams(262144, 8, 1, 32, salt)

	if ks.Crypto.KDF != "scrypt" {
		t.Errorf("Expected KDF scrypt, got %s", ks.Crypto.KDF)
	}

	params, err := ks.GetScryptParams()
	if err != nil {
		t.Fatalf("Failed to get scrypt params: %v", err)
	}

	if params.N != 262144 {
		t.Errorf("Expected N=262144, got %d", params.N)
	}

	if params.R != 8 {
		t.Errorf("Expected R=8, got %d", params.R)
	}

	if params.P != 1 {
		t.Errorf("Expected P=1, got %d", params.P)
	}

	if params.DKLen != 32 {
		t.Errorf("Expected DKLen=32, got %d", params.DKLen)
	}

	expectedSalt := hex.EncodeToString(salt)
	if params.Salt != expectedSalt {
		t.Errorf("Expected salt %s, got %s", expectedSalt, params.Salt)
	}
}

func TestKeyStoreV3_SetPBKDF2Params(t *testing.T) {
	ks := NewKeyStoreV3("0x1234567890abcdef1234567890abcdef12345678", "ethereum")
	salt := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}

	ks.SetPBKDF2Params(262144, 32, "hmac-sha256", salt)

	if ks.Crypto.KDF != "pbkdf2" {
		t.Errorf("Expected KDF pbkdf2, got %s", ks.Crypto.KDF)
	}

	params, err := ks.GetPBKDF2Params()
	if err != nil {
		t.Fatalf("Failed to get PBKDF2 params: %v", err)
	}

	if params.C != 262144 {
		t.Errorf("Expected C=262144, got %d", params.C)
	}

	if params.DKLen != 32 {
		t.Errorf("Expected DKLen=32, got %d", params.DKLen)
	}

	if params.PRF != "hmac-sha256" {
		t.Errorf("Expected PRF=hmac-sha256, got %s", params.PRF)
	}

	expectedSalt := hex.EncodeToString(salt)
	if params.Salt != expectedSalt {
		t.Errorf("Expected salt %s, got %s", expectedSalt, params.Salt)
	}
}

func TestKeyStoreV3_SetCipherParams(t *testing.T) {
	ks := NewKeyStoreV3("0x1234567890abcdef1234567890abcdef12345678", "ethereum")
	iv := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	ciphertext := []byte{0xde, 0xad, 0xbe, 0xef}

	ks.SetCipherParams(iv, ciphertext)

	expectedIV := hex.EncodeToString(iv)
	if ks.Crypto.CipherParams.IV != expectedIV {
		t.Errorf("Expected IV %s, got %s", expectedIV, ks.Crypto.CipherParams.IV)
	}

	expectedCiphertext := hex.EncodeToString(ciphertext)
	if ks.Crypto.CipherText != expectedCiphertext {
		t.Errorf("Expected ciphertext %s, got %s", expectedCiphertext, ks.Crypto.CipherText)
	}
}

func TestKeyStoreV3_SetMAC(t *testing.T) {
	ks := NewKeyStoreV3("0x1234567890abcdef1234567890abcdef12345678", "ethereum")
	mac := []byte{0xca, 0xfe, 0xba, 0xbe}

	ks.SetMAC(mac)

	expectedMAC := hex.EncodeToString(mac)
	if ks.Crypto.MAC != expectedMAC {
		t.Errorf("Expected MAC %s, got %s", expectedMAC, ks.Crypto.MAC)
	}
}

func TestGenerateRandomBytes(t *testing.T) {
	tests := []struct {
		name   string
		length int
	}{
		{"16 bytes", 16},
		{"32 bytes", 32},
		{"64 bytes", 64},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bytes, err := GenerateRandomBytes(tt.length)
			if err != nil {
				t.Fatalf("Failed to generate random bytes: %v", err)
			}

			if len(bytes) != tt.length {
				t.Errorf("Expected %d bytes, got %d", tt.length, len(bytes))
			}

			// Generate another set and ensure they're different
			bytes2, err := GenerateRandomBytes(tt.length)
			if err != nil {
				t.Fatalf("Failed to generate second set of random bytes: %v", err)
			}

			if string(bytes) == string(bytes2) {
				t.Error("Generated bytes should be different (extremely unlikely to be the same)")
			}
		})
	}
}

func TestKeyStoreV3_Validate(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func() *KeyStoreV3
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid keystore",
			setupFunc: func() *KeyStoreV3 {
				ks := NewKeyStoreV3("0x1234567890abcdef1234567890abcdef12345678", "ethereum")
				ks.SetScryptParams(262144, 8, 1, 32, bytes32(1))
				ks.SetCipherParams(bytes16(1), bytes32(2))
				ks.SetMAC(bytes32(9))
				return ks
			},
			wantError: false,
		},
		{
			name: "empty address",
			setupFunc: func() *KeyStoreV3 {
				ks := NewKeyStoreV3("0x1234567890abcdef1234567890abcdef12345678", "ethereum")
				ks.Address = ""
				return ks
			},
			wantError: true,
			errorMsg:  "address cannot be empty",
		},

		{
			name: "invalid version",
			setupFunc: func() *KeyStoreV3 {
				ks := NewKeyStoreV3("0x1234567890abcdef1234567890abcdef12345678", "ethereum")
				ks.Version = 2
				return ks
			},
			wantError: true,
			errorMsg:  "version must be 3",
		},
		{
			name: "empty ID",
			setupFunc: func() *KeyStoreV3 {
				ks := NewKeyStoreV3("0x1234567890abcdef1234567890abcdef12345678", "ethereum")
				ks.ID = ""
				return ks
			},
			wantError: true,
			errorMsg:  "ID cannot be empty",
		},
		{
			name: "unsupported cipher",
			setupFunc: func() *KeyStoreV3 {
				ks := NewKeyStoreV3("0x1234567890abcdef1234567890abcdef12345678", "ethereum")
				ks.Crypto.Cipher = "aes-256-cbc"
				return ks
			},
			wantError: true,
			errorMsg:  "unsupported cipher",
		},
		{
			name: "unsupported KDF",
			setupFunc: func() *KeyStoreV3 {
				ks := NewKeyStoreV3("0x1234567890abcdef1234567890abcdef12345678", "ethereum")
				ks.Crypto.KDF = "bcrypt"
				return ks
			},
			wantError: true,
			errorMsg:  "unsupported KDF",
		},
		{
			name: "empty ciphertext",
			setupFunc: func() *KeyStoreV3 {
				ks := NewKeyStoreV3("0x1234567890abcdef1234567890abcdef12345678", "ethereum")
				ks.SetScryptParams(262144, 8, 1, 32, bytes32(1))
				ks.SetCipherParams(bytes16(1), []byte{})
				return ks
			},
			wantError: true,
			errorMsg:  "ciphertext cannot be empty",
		},
		{
			name: "empty MAC",
			setupFunc: func() *KeyStoreV3 {
				ks := NewKeyStoreV3("0x1234567890abcdef1234567890abcdef12345678", "ethereum")
				ks.SetScryptParams(262144, 8, 1, 32, bytes32(1))
				ks.SetCipherParams(bytes16(1), bytes32(2))
				return ks
			},
			wantError: true,
			errorMsg:  "MAC cannot be empty",
		},
		{
			name: "empty IV",
			setupFunc: func() *KeyStoreV3 {
				ks := NewKeyStoreV3("0x1234567890abcdef1234567890abcdef12345678", "ethereum")
				ks.SetScryptParams(262144, 8, 1, 32, bytes32(1))
				ks.Crypto.CipherText = hex.EncodeToString(bytes32(3))
				ks.SetMAC(bytes32(9))
				return ks
			},
			wantError: true,
			errorMsg:  "IV cannot be empty",
		},
		{
			name: "nil KDF params",
			setupFunc: func() *KeyStoreV3 {
				ks := NewKeyStoreV3("0x1234567890abcdef1234567890abcdef12345678", "ethereum")
				ks.Crypto.KDFParams = nil
				ks.SetCipherParams(bytes16(1), bytes32(2))
				ks.SetMAC(bytes32(9))
				return ks
			},
			wantError: true,
			errorMsg:  "KDF parameters cannot be nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ks := tt.setupFunc()
			err := ks.Validate()

			if tt.wantError {
				if err == nil {
					t.Error("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestKeyStoreV3_JSONSerialization(t *testing.T) {
	// Create a complete keystore
	ks := NewKeyStoreV3("0x1234567890abcdef1234567890abcdef12345678", "ethereum")
	salt := bytes32(1)
	iv := bytes16(16)
	ciphertext := bytes32(0xde)
	mac := bytes32(0x12)

	ks.SetScryptParams(262144, 8, 1, 32, salt)
	ks.SetCipherParams(iv, ciphertext)
	ks.SetMAC(mac)

	// Test ToJSON
	jsonData, err := ks.ToJSON()
	if err != nil {
		t.Fatalf("Failed to serialize to JSON: %v", err)
	}

	// Verify JSON structure
	var jsonObj map[string]interface{}
	if err := json.Unmarshal(jsonData, &jsonObj); err != nil {
		t.Fatalf("Failed to parse generated JSON: %v", err)
	}

	// Check required fields
	if jsonObj["address"] != ks.Address {
		t.Errorf("JSON address mismatch: expected %s, got %v", ks.Address, jsonObj["address"])
	}

	if jsonObj["version"] != float64(3) {
		t.Errorf("JSON version mismatch: expected 3, got %v", jsonObj["version"])
	}

	if jsonObj["id"] != ks.ID {
		t.Errorf("JSON ID mismatch: expected %s, got %v", ks.ID, jsonObj["id"])
	}

	// Test FromJSON
	ks2, err := FromJSON(jsonData)
	if err != nil {
		t.Fatalf("Failed to deserialize from JSON: %v", err)
	}

	// Compare original and deserialized
	if ks2.Address != ks.Address {
		t.Errorf("Address mismatch after deserialization: expected %s, got %s", ks.Address, ks2.Address)
	}

	if ks2.Version != ks.Version {
		t.Errorf("Version mismatch after deserialization: expected %d, got %d", ks.Version, ks2.Version)
	}

	if ks2.ID != ks.ID {
		t.Errorf("ID mismatch after deserialization: expected %s, got %s", ks.ID, ks2.ID)
	}

	if ks2.Crypto.Cipher != ks.Crypto.Cipher {
		t.Errorf("Cipher mismatch after deserialization: expected %s, got %s", ks.Crypto.Cipher, ks2.Crypto.Cipher)
	}

	if ks2.Crypto.KDF != ks.Crypto.KDF {
		t.Errorf("KDF mismatch after deserialization: expected %s, got %s", ks.Crypto.KDF, ks2.Crypto.KDF)
	}

	if ks2.Crypto.CipherText != ks.Crypto.CipherText {
		t.Errorf("CipherText mismatch after deserialization: expected %s, got %s", ks.Crypto.CipherText, ks2.Crypto.CipherText)
	}

	if ks2.Crypto.MAC != ks.Crypto.MAC {
		t.Errorf("MAC mismatch after deserialization: expected %s, got %s", ks.Crypto.MAC, ks2.Crypto.MAC)
	}
}

func TestKeyStoreV3_JSONSerializationWithPBKDF2(t *testing.T) {
	// Create a keystore with PBKDF2
	ks := NewKeyStoreV3("0x1234567890abcdef1234567890abcdef12345678", "ethereum")
	salt := bytes32(1)
	iv := bytes16(16)
	ciphertext := bytes32(0xde)
	mac := bytes32(0x12)

	ks.SetPBKDF2Params(262144, 32, "hmac-sha256", salt)
	ks.SetCipherParams(iv, ciphertext)
	ks.SetMAC(mac)

	// Test ToJSON
	jsonData, err := ks.ToJSON()
	if err != nil {
		t.Fatalf("Failed to serialize to JSON: %v", err)
	}

	// Test FromJSON
	ks2, err := FromJSON(jsonData)
	if err != nil {
		t.Fatalf("Failed to deserialize from JSON: %v", err)
	}

	// Verify PBKDF2 parameters
	params, err := ks2.GetPBKDF2Params()
	if err != nil {
		t.Fatalf("Failed to get PBKDF2 params: %v", err)
	}

	if params.C != 262144 {
		t.Errorf("Expected C=262144, got %d", params.C)
	}

	if params.DKLen != 32 {
		t.Errorf("Expected DKLen=32, got %d", params.DKLen)
	}

	if params.PRF != "hmac-sha256" {
		t.Errorf("Expected PRF=hmac-sha256, got %s", params.PRF)
	}
}

func TestFromJSON_InvalidJSON(t *testing.T) {
	tests := []struct {
		name     string
		jsonData string
		wantErr  bool
	}{
		{
			name:     "invalid JSON syntax",
			jsonData: `{"address": "invalid json"`,
			wantErr:  true,
		},
		{
			name:     "missing required fields",
			jsonData: `{"address": "1234567890abcdef1234567890abcdef12345678"}`,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := FromJSON([]byte(tt.jsonData))
			if (err != nil) != tt.wantErr {
				t.Errorf("FromJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestKeyStoreV3_GetParams_WrongKDF(t *testing.T) {
	ks := NewKeyStoreV3("0x1234567890abcdef1234567890abcdef12345678", "ethereum")

	// Test getting scrypt params when KDF is scrypt (should work)
	ks.SetScryptParams(262144, 8, 1, 32, bytes32(1))
	_, err := ks.GetScryptParams()
	if err != nil {
		t.Errorf("Expected no error for scrypt params with scrypt KDF, got: %v", err)
	}

	// Test getting PBKDF2 params when KDF is scrypt (should fail)
	_, err = ks.GetPBKDF2Params()
	if err == nil {
		t.Error("Expected error for PBKDF2 params with scrypt KDF")
	}

	// Switch to PBKDF2
	ks.SetPBKDF2Params(262144, 32, "hmac-sha256", []byte{1, 2, 3, 4})

	// Test getting PBKDF2 params when KDF is pbkdf2 (should work)
	_, err = ks.GetPBKDF2Params()
	if err != nil {
		t.Errorf("Expected no error for PBKDF2 params with pbkdf2 KDF, got: %v", err)
	}

	// Test getting scrypt params when KDF is pbkdf2 (should fail)
	_, err = ks.GetScryptParams()
	if err == nil {
		t.Error("Expected error for scrypt params with pbkdf2 KDF")
	}
}

// Tests for cryptographic functions

func TestDeriveKeyScrypt(t *testing.T) {
	tests := []struct {
		name      string
		password  []byte
		salt      []byte
		n         int
		r         int
		p         int
		dkLen     int
		wantError bool
		errorMsg  string
	}{
		{
			name:      "valid scrypt parameters",
			password:  []byte("testpassword"),
			salt:      []byte("testsalt12345678"),
			n:         16384,
			r:         8,
			p:         1,
			dkLen:     32,
			wantError: false,
		},
		{
			name:      "empty password",
			password:  []byte{},
			salt:      []byte("testsalt12345678"),
			n:         16384,
			r:         8,
			p:         1,
			dkLen:     32,
			wantError: true,
			errorMsg:  "password cannot be empty",
		},
		{
			name:      "empty salt",
			password:  []byte("testpassword"),
			salt:      []byte{},
			n:         16384,
			r:         8,
			p:         1,
			dkLen:     32,
			wantError: true,
			errorMsg:  "salt cannot be empty",
		},
		{
			name:      "invalid dkLen",
			password:  []byte("testpassword"),
			salt:      []byte("testsalt12345678"),
			n:         16384,
			r:         8,
			p:         1,
			dkLen:     0,
			wantError: true,
			errorMsg:  "derived key length must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := DeriveKeyScrypt(tt.password, tt.salt, tt.n, tt.r, tt.p, tt.dkLen)

			if tt.wantError {
				if err == nil {
					t.Error("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
				if len(key) != tt.dkLen {
					t.Errorf("Expected key length %d, got %d", tt.dkLen, len(key))
				}

				// Test deterministic behavior - same inputs should produce same output
				key2, err2 := DeriveKeyScrypt(tt.password, tt.salt, tt.n, tt.r, tt.p, tt.dkLen)
				if err2 != nil {
					t.Errorf("Second derivation failed: %v", err2)
				}
				if !strings.EqualFold(hex.EncodeToString(key), hex.EncodeToString(key2)) {
					t.Error("Scrypt should be deterministic - same inputs should produce same output")
				}
			}
		})
	}
}

func TestDeriveKeyPBKDF2(t *testing.T) {
	tests := []struct {
		name       string
		password   []byte
		salt       []byte
		iterations int
		dkLen      int
		wantError  bool
		errorMsg   string
	}{
		{
			name:       "valid PBKDF2 parameters",
			password:   []byte("testpassword"),
			salt:       []byte("testsalt12345678"),
			iterations: 10000,
			dkLen:      32,
			wantError:  false,
		},
		{
			name:       "empty password",
			password:   []byte{},
			salt:       []byte("testsalt12345678"),
			iterations: 10000,
			dkLen:      32,
			wantError:  true,
			errorMsg:   "password cannot be empty",
		},
		{
			name:       "empty salt",
			password:   []byte("testpassword"),
			salt:       []byte{},
			iterations: 10000,
			dkLen:      32,
			wantError:  true,
			errorMsg:   "salt cannot be empty",
		},
		{
			name:       "invalid iterations",
			password:   []byte("testpassword"),
			salt:       []byte("testsalt12345678"),
			iterations: 0,
			dkLen:      32,
			wantError:  true,
			errorMsg:   "iterations must be positive",
		},
		{
			name:       "invalid dkLen",
			password:   []byte("testpassword"),
			salt:       []byte("testsalt12345678"),
			iterations: 10000,
			dkLen:      0,
			wantError:  true,
			errorMsg:   "derived key length must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := DeriveKeyPBKDF2(tt.password, tt.salt, tt.iterations, tt.dkLen)

			if tt.wantError {
				if err == nil {
					t.Error("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
				if len(key) != tt.dkLen {
					t.Errorf("Expected key length %d, got %d", tt.dkLen, len(key))
				}

				// Test deterministic behavior - same inputs should produce same output
				key2, err2 := DeriveKeyPBKDF2(tt.password, tt.salt, tt.iterations, tt.dkLen)
				if err2 != nil {
					t.Errorf("Second derivation failed: %v", err2)
				}
				if !strings.EqualFold(hex.EncodeToString(key), hex.EncodeToString(key2)) {
					t.Error("PBKDF2 should be deterministic - same inputs should produce same output")
				}
			}
		})
	}
}

func TestEncryptDecryptAES128CTR(t *testing.T) {
	tests := []struct {
		name      string
		plaintext []byte
		key       []byte
		iv        []byte
		wantError bool
		errorMsg  string
	}{
		{
			name:      "valid encryption",
			plaintext: []byte("Hello, World! This is a test message."),
			key:       []byte("1234567890123456"), // 16 bytes
			iv:        []byte("abcdefghijklmnop"), // 16 bytes
			wantError: false,
		},
		{
			name:      "empty plaintext",
			plaintext: []byte{},
			key:       []byte("1234567890123456"),
			iv:        []byte("abcdefghijklmnop"),
			wantError: true,
			errorMsg:  "plaintext cannot be empty",
		},
		{
			name:      "invalid key length",
			plaintext: []byte("Hello, World!"),
			key:       []byte("12345"), // Wrong length
			iv:        []byte("abcdefghijklmnop"),
			wantError: true,
			errorMsg:  "key must be 16 bytes for AES-128",
		},
		{
			name:      "invalid IV length",
			plaintext: []byte("Hello, World!"),
			key:       []byte("1234567890123456"),
			iv:        []byte("abc"), // Wrong length
			wantError: true,
			errorMsg:  "IV must be 16 bytes for AES-128-CTR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test encryption
			ciphertext, err := EncryptAES128CTR(tt.plaintext, tt.key, tt.iv)

			if tt.wantError {
				if err == nil {
					t.Error("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("Encryption failed: %v", err)
				return
			}

			if len(ciphertext) != len(tt.plaintext) {
				t.Errorf("Ciphertext length %d doesn't match plaintext length %d", len(ciphertext), len(tt.plaintext))
			}

			// Test decryption
			decrypted, err := DecryptAES128CTR(ciphertext, tt.key, tt.iv)
			if err != nil {
				t.Errorf("Decryption failed: %v", err)
				return
			}

			if !strings.EqualFold(string(decrypted), string(tt.plaintext)) {
				t.Errorf("Decrypted text doesn't match original. Expected: %s, Got: %s", string(tt.plaintext), string(decrypted))
			}
		})
	}
}

func TestDecryptAES128CTR_Errors(t *testing.T) {
	tests := []struct {
		name       string
		ciphertext []byte
		key        []byte
		iv         []byte
		wantError  bool
		errorMsg   string
	}{
		{
			name:       "empty ciphertext",
			ciphertext: []byte{},
			key:        []byte("1234567890123456"),
			iv:         []byte("abcdefghijklmnop"),
			wantError:  true,
			errorMsg:   "ciphertext cannot be empty",
		},
		{
			name:       "invalid key length",
			ciphertext: []byte("Hello, World!"),
			key:        []byte("12345"), // Wrong length
			iv:         []byte("abcdefghijklmnop"),
			wantError:  true,
			errorMsg:   "key must be 16 bytes for AES-128",
		},
		{
			name:       "invalid IV length",
			ciphertext: []byte("Hello, World!"),
			key:        []byte("1234567890123456"),
			iv:         []byte("abc"), // Wrong length
			wantError:  true,
			errorMsg:   "IV must be 16 bytes for AES-128-CTR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DecryptAES128CTR(tt.ciphertext, tt.key, tt.iv)

			if tt.wantError {
				if err == nil {
					t.Error("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestGenerateMAC(t *testing.T) {
	tests := []struct {
		name       string
		derivedKey []byte
		ciphertext []byte
		wantError  bool
		errorMsg   string
	}{
		{
			name:       "valid MAC generation",
			derivedKey: make([]byte, 32), // 32 bytes
			ciphertext: []byte("test ciphertext"),
			wantError:  false,
		},
		{
			name:       "derived key too short",
			derivedKey: make([]byte, 16), // Too short
			ciphertext: []byte("test ciphertext"),
			wantError:  true,
			errorMsg:   "derived key must be at least 32 bytes",
		},
		{
			name:       "empty ciphertext",
			derivedKey: make([]byte, 32),
			ciphertext: []byte{},
			wantError:  true,
			errorMsg:   "ciphertext cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mac, err := GenerateMAC(tt.derivedKey, tt.ciphertext)

			if tt.wantError {
				if err == nil {
					t.Error("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
				if len(mac) != 32 { // SHA256 produces 32 bytes
					t.Errorf("Expected MAC length 32, got %d", len(mac))
				}

				// Test deterministic behavior
				mac2, err2 := GenerateMAC(tt.derivedKey, tt.ciphertext)
				if err2 != nil {
					t.Errorf("Second MAC generation failed: %v", err2)
				}
				if !strings.EqualFold(hex.EncodeToString(mac), hex.EncodeToString(mac2)) {
					t.Error("MAC generation should be deterministic")
				}
			}
		})
	}
}

func TestVerifyMAC(t *testing.T) {
	derivedKey := make([]byte, 32)
	ciphertext := []byte("test ciphertext")

	// Generate a valid MAC
	validMAC, err := GenerateMAC(derivedKey, ciphertext)
	if err != nil {
		t.Fatalf("Failed to generate MAC for test: %v", err)
	}

	tests := []struct {
		name        string
		derivedKey  []byte
		ciphertext  []byte
		expectedMAC []byte
		wantValid   bool
		wantError   bool
		errorMsg    string
	}{
		{
			name:        "valid MAC verification",
			derivedKey:  derivedKey,
			ciphertext:  ciphertext,
			expectedMAC: validMAC,
			wantValid:   true,
			wantError:   false,
		},
		{
			name:        "invalid MAC",
			derivedKey:  derivedKey,
			ciphertext:  ciphertext,
			expectedMAC: []byte("invalid mac"),
			wantValid:   false,
			wantError:   false,
		},
		{
			name:        "empty expected MAC",
			derivedKey:  derivedKey,
			ciphertext:  ciphertext,
			expectedMAC: []byte{},
			wantValid:   false,
			wantError:   true,
			errorMsg:    "expected MAC cannot be empty",
		},
		{
			name:        "derived key too short",
			derivedKey:  make([]byte, 16),
			ciphertext:  ciphertext,
			expectedMAC: validMAC,
			wantValid:   false,
			wantError:   true,
			errorMsg:    "derived key must be at least 32 bytes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, err := VerifyMAC(tt.derivedKey, tt.ciphertext, tt.expectedMAC)

			if tt.wantError {
				if err == nil {
					t.Error("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
				if valid != tt.wantValid {
					t.Errorf("Expected valid=%v, got %v", tt.wantValid, valid)
				}
			}
		})
	}
}

func TestEncryptPrivateKey(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping encrypt private key test in short mode")
	}

	tests := []struct {
		name       string
		privateKey string
		password   string
		kdf        string
		wantError  bool
		errorMsg   string
	}{
		{
			name:       "valid scrypt encryption",
			privateKey: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			password:   "testpassword",
			kdf:        "scrypt",
			wantError:  false,
		},
		{
			name:       "valid pbkdf2 encryption",
			privateKey: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			password:   "testpassword",
			kdf:        "pbkdf2",
			wantError:  false,
		},
		{
			name:       "private key with 0x prefix",
			privateKey: "0x0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			password:   "testpassword",
			kdf:        "scrypt",
			wantError:  false,
		},
		{
			name:       "empty private key",
			privateKey: "",
			password:   "testpassword",
			kdf:        "scrypt",
			wantError:  true,
			errorMsg:   "private key cannot be empty",
		},
		{
			name:       "empty password",
			privateKey: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			password:   "",
			kdf:        "scrypt",
			wantError:  true,
			errorMsg:   "password cannot be empty",
		},
		{
			name:       "invalid private key length",
			privateKey: "0123456789abcdef", // Too short
			password:   "testpassword",
			kdf:        "scrypt",
			wantError:  true,
			errorMsg:   "private key must be 64 hex characters",
		},
		{
			name:       "invalid private key hex",
			privateKey: "gggggggggggggggggggggggggggggggggggggggggggggggggggggggggggggggg",
			password:   "testpassword",
			kdf:        "scrypt",
			wantError:  true,
			errorMsg:   "invalid private key hex",
		},
		{
			name:       "unsupported KDF",
			privateKey: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			password:   "testpassword",
			kdf:        "bcrypt",
			wantError:  true,
			errorMsg:   "KDF not supported", // Updated for Universal KDF integration
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ks, err := EncryptPrivateKey(tt.privateKey, tt.password, tt.kdf)

			if tt.wantError {
				if err == nil {
					t.Error("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}

				// Validate the keystore structure
				if err := ks.Validate(); err != nil {
					t.Errorf("Generated keystore is invalid: %v", err)
				}

				// Check KDF type
				if ks.Crypto.KDF != tt.kdf {
					t.Errorf("Expected KDF %s, got %s", tt.kdf, ks.Crypto.KDF)
				}

				// Check cipher
				if ks.Crypto.Cipher != "aes-128-ctr" {
					t.Errorf("Expected cipher aes-128-ctr, got %s", ks.Crypto.Cipher)
				}

				// Verify we can decrypt back to original
				decrypted, err := DecryptPrivateKey(ks, tt.password)
				if err != nil {
					t.Errorf("Failed to decrypt private key: %v", err)
				} else {
					expectedKey := strings.TrimPrefix(tt.privateKey, "0x")
					if hex.EncodeToString(decrypted) != expectedKey {
						t.Errorf("Decrypted key doesn't match original. Expected: %s, Got: %s", expectedKey, hex.EncodeToString(decrypted))
					}
				}
			}
		})
	}
}

func TestDecryptPrivateKey(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping decrypt private key test in short mode")
	}

	// Create a valid keystore for testing
	privateKey := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	password := "testpassword"

	validKS, err := EncryptPrivateKey(privateKey, password, "scrypt")
	if err != nil {
		t.Fatalf("Failed to create test keystore: %v", err)
	}

	tests := []struct {
		name      string
		keystore  *KeyStoreV3
		password  string
		wantError bool
		errorMsg  string
	}{
		{
			name:      "valid decryption",
			keystore:  validKS,
			password:  password,
			wantError: false,
		},
		{
			name:      "nil keystore",
			keystore:  nil,
			password:  password,
			wantError: true,
			errorMsg:  "keystore cannot be nil",
		},
		{
			name:      "empty password",
			keystore:  validKS,
			password:  "",
			wantError: true,
			errorMsg:  "password cannot be empty",
		},
		{
			name:      "wrong password",
			keystore:  validKS,
			password:  "wrongpassword",
			wantError: true,
			errorMsg:  "MAC verification failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decrypted, err := DecryptPrivateKey(tt.keystore, tt.password)

			if tt.wantError {
				if err == nil {
					t.Error("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
				if hex.EncodeToString(decrypted) != privateKey {
					t.Errorf("Decrypted key doesn't match original. Expected: %s, Got: %s", privateKey, hex.EncodeToString(decrypted))
				}
			}
		})
	}
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping encrypt/decrypt round trip test in short mode")
	}

	testCases := []struct {
		name       string
		privateKey string
		password   string
		kdf        string
	}{
		{
			name:       "scrypt round trip",
			privateKey: "fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210",
			password:   "complex_password_123!@#",
			kdf:        "scrypt",
		},
		{
			name:       "pbkdf2 round trip",
			privateKey: "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789",
			password:   "another_password_456$%^",
			kdf:        "pbkdf2",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Encrypt
			ks, err := EncryptPrivateKey(tc.privateKey, tc.password, tc.kdf)
			if err != nil {
				t.Fatalf("Encryption failed: %v", err)
			}

			// Decrypt
			decrypted, err := DecryptPrivateKey(ks, tc.password)
			if err != nil {
				t.Fatalf("Decryption failed: %v", err)
			}

			// Verify
			if hex.EncodeToString(decrypted) != tc.privateKey {
				t.Errorf("Round trip failed. Expected: %s, Got: %s", tc.privateKey, hex.EncodeToString(decrypted))
			}

			// Verify keystore can be serialized and deserialized
			jsonData, err := ks.ToJSON()
			if err != nil {
				t.Errorf("Failed to serialize keystore: %v", err)
			}

			ks2, err := FromJSON(jsonData)
			if err != nil {
				t.Errorf("Failed to deserialize keystore: %v", err)
			}

			// Decrypt from deserialized keystore
			decrypted2, err := DecryptPrivateKey(ks2, tc.password)
			if err != nil {
				t.Errorf("Failed to decrypt from deserialized keystore: %v", err)
			}

			if hex.EncodeToString(decrypted2) != tc.privateKey {
				t.Errorf("Deserialization round trip failed. Expected: %s, Got: %s", tc.privateKey, hex.EncodeToString(decrypted2))
			}
		})
	}
}

// KeyStore Service Tests

func TestNewKeyStoreService(t *testing.T) {
	tests := []struct {
		name           string
		config         KeyStoreConfig
		expectedCipher string
		expectedKDF    string
		expectedDir    string
	}{
		{
			name:           "default configuration",
			config:         KeyStoreConfig{},
			expectedCipher: "aes-128-ctr",
			expectedKDF:    "scrypt",
			expectedDir:    "./keystores",
		},
		{
			name: "custom configuration",
			config: KeyStoreConfig{
				OutputDirectory: "/tmp/test-keystores",
				Enabled:         true,
				Cipher:          "aes-128-ctr",
				KDF:             "pbkdf2",
			},
			expectedCipher: "aes-128-ctr",
			expectedKDF:    "pbkdf2",
			expectedDir:    "/tmp/test-keystores",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewKeyStoreService(tt.config)

			if service == nil {
				t.Fatal("NewKeyStoreService returned nil")
			}

			config := service.GetConfig()
			if config.Cipher != tt.expectedCipher {
				t.Errorf("Expected cipher %s, got %s", tt.expectedCipher, config.Cipher)
			}

			if config.KDF != tt.expectedKDF {
				t.Errorf("Expected KDF %s, got %s", tt.expectedKDF, config.KDF)
			}

			if config.OutputDirectory != tt.expectedDir {
				t.Errorf("Expected output directory %s, got %s", tt.expectedDir, config.OutputDirectory)
			}

			if service.passwordGen == nil {
				t.Error("Password generator should not be nil")
			}
		})
	}
}

func TestKeyStoreService_GenerateKeyStore(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping keystore service generate test in short mode")
	}

	tests := []struct {
		name        string
		enabled     bool
		privateKey  string
		address     string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid generation",
			enabled:     true,
			privateKey:  "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			address:     "0x1234567890abcdef1234567890abcdef12345678",
			expectError: false,
		},
		{
			name:        "disabled service",
			enabled:     false,
			privateKey:  "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			address:     "0x1234567890abcdef1234567890abcdef12345678",
			expectError: true,
			errorMsg:    "keystore generation is disabled",
		},
		{
			name:        "empty private key",
			enabled:     true,
			privateKey:  "",
			address:     "0x1234567890abcdef1234567890abcdef12345678",
			expectError: true,
			errorMsg:    "private key cannot be empty",
		},
		{
			name:        "empty address",
			enabled:     true,
			privateKey:  "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			address:     "",
			expectError: true,
			errorMsg:    "address cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := KeyStoreConfig{
				Enabled:         tt.enabled,
				OutputDirectory: "/tmp/test-keystores",
				KDF:             "scrypt",
			}
			service := NewKeyStoreService(config)

			keystore, password, err := service.GenerateKeyStore(tt.privateKey, tt.address, "ethereum")

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tt.errorMsg)
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if keystore == nil {
				t.Fatal("Generated keystore should not be nil")
			}

			if password == "" {
				t.Fatal("Generated password should not be empty")
			}

			// Validate keystore structure
			if err := keystore.Validate(); err != nil {
				t.Errorf("Generated keystore is invalid: %v", err)
			}

			// Check address is correctly set
			expectedAddress := strings.ToLower(strings.TrimPrefix(tt.address, "0x"))
			if keystore.Address != expectedAddress {
				t.Errorf("Expected address %s, got %s", expectedAddress, keystore.Address)
			}

			// Verify password complexity
			passwordGen := NewPasswordGenerator()
			if err := passwordGen.ValidatePassword(password); err != nil {
				t.Errorf("Generated password does not meet complexity requirements: %v", err)
			}

			// Verify we can decrypt the private key
			decryptedKey, err := DecryptPrivateKey(keystore, password)
			if err != nil {
				t.Errorf("Failed to decrypt private key: %v", err)
			}

			decryptedHex := hex.EncodeToString(decryptedKey)
			expectedPrivateKey := strings.TrimPrefix(tt.privateKey, "0x")
			if decryptedHex != expectedPrivateKey {
				t.Errorf("Decrypted private key mismatch: expected %s, got %s", expectedPrivateKey, decryptedHex)
			}
		})
	}
}

func TestKeyStoreService_SaveKeyStoreFiles(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "keystore-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	tests := []struct {
		name        string
		enabled     bool
		address     string
		keystore    *KeyStoreV3
		password    string
		expectError bool
		errorMsg    string
	}{
		{
			name:     "valid save operation",
			enabled:  true,
			address:  "0x1234567890abcdef1234567890abcdef12345678",
			keystore: createValidKeyStore("1234567890abcdef1234567890abcdef12345678"),
			password: "TestPassword123!",
		},
		{
			name:        "disabled service",
			enabled:     false,
			address:     "0x1234567890abcdef1234567890abcdef12345678",
			keystore:    createValidKeyStore("1234567890abcdef1234567890abcdef12345678"),
			password:    "TestPassword123!",
			expectError: true,
			errorMsg:    "keystore generation is disabled",
		},
		{
			name:        "nil keystore",
			enabled:     true,
			address:     "0x1234567890abcdef1234567890abcdef12345678",
			keystore:    nil,
			password:    "TestPassword123!",
			expectError: true,
			errorMsg:    "keystore cannot be nil",
		},
		{
			name:        "empty password",
			enabled:     true,
			address:     "0x1234567890abcdef1234567890abcdef12345678",
			keystore:    createValidKeyStore("1234567890abcdef1234567890abcdef12345678"),
			password:    "",
			expectError: true,
			errorMsg:    "password cannot be empty",
		},
		{
			name:        "invalid address length",
			enabled:     true,
			address:     "0x123",
			keystore:    createValidKeyStore("1234567890abcdef1234567890abcdef12345678"),
			password:    "TestPassword123!",
			expectError: true,
			errorMsg:    "invalid Ethereum address length",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := KeyStoreConfig{
				Enabled:         tt.enabled,
				OutputDirectory: tempDir,
				KDF:             "scrypt",
				// This test asserts on the password file itself, so it opts in.
				WritePasswordFile: true,
			}
			service := NewKeyStoreService(config)

			err := service.SaveKeyStoreFilesToDisk(tt.address, tt.keystore, tt.password, "ethereum", "")

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tt.errorMsg)
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Verify files were created (keystore uses the geth naming
			// convention UTC--<timestamp>--<address>.json)
			cleanAddress := strings.ToLower(strings.TrimPrefix(tt.address, "0x"))
			keystoreFile := ""
			entries, err := os.ReadDir(tempDir)
			if err != nil {
				t.Fatalf("Failed to read output directory: %v", err)
			}
			for _, entry := range entries {
				if strings.HasPrefix(entry.Name(), "UTC--") && strings.HasSuffix(entry.Name(), fmt.Sprintf("--0x%s.json", cleanAddress)) {
					keystoreFile = filepath.Join(tempDir, entry.Name())
					break
				}
			}
			if keystoreFile == "" {
				t.Errorf("Keystore file (UTC--*--0x%s.json) was not created", cleanAddress)
			}
			passwordFile := filepath.Join(tempDir, fmt.Sprintf("0x%s.pwd", cleanAddress))

			// Check keystore file exists and has correct content
			if _, err := os.Stat(keystoreFile); os.IsNotExist(err) {
				t.Errorf("Keystore file was not created: %s", keystoreFile)
			}

			// Check password file exists and has correct content
			if _, err := os.Stat(passwordFile); os.IsNotExist(err) {
				t.Errorf("Password file was not created: %s", passwordFile)
			}

			// Verify file contents
			keystoreData, err := os.ReadFile(keystoreFile)
			if err != nil {
				t.Errorf("Failed to read keystore file: %v", err)
			}

			passwordData, err := os.ReadFile(passwordFile)
			if err != nil {
				t.Errorf("Failed to read password file: %v", err)
			}

			if string(passwordData) != tt.password {
				t.Errorf("Password file content mismatch: expected %s, got %s", tt.password, string(passwordData))
			}

			// Verify keystore can be deserialized
			_, err = FromJSON(keystoreData)
			if err != nil {
				t.Errorf("Failed to deserialize saved keystore: %v", err)
			}

			// Check file permissions
			keystoreInfo, err := os.Stat(keystoreFile)
			if err != nil {
				t.Errorf("Failed to get keystore file info: %v", err)
			} else if keystoreInfo.Mode().Perm() != 0600 {
				t.Errorf("Keystore file has incorrect permissions: expected 0600, got %o", keystoreInfo.Mode().Perm())
			}

			passwordInfo, err := os.Stat(passwordFile)
			if err != nil {
				t.Errorf("Failed to get password file info: %v", err)
			} else if passwordInfo.Mode().Perm() != 0600 {
				t.Errorf("Password file has incorrect permissions: expected 0600, got %o", passwordInfo.Mode().Perm())
			}
		})
	}
}

func TestFormatAddressForFilenamePreservesEthereumCase(t *testing.T) {
	address := "0xDEaD70220f96970E686E56C7ed6b8376bADaAF6e"

	got := formatAddressForFilename(address, "ethereum")
	if got != address {
		t.Fatalf("expected %s, got %s", address, got)
	}
}

func TestKeyStoreService_SaveKeyStoreFilesPreservesEthereumFilenameCase(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "keystore-case-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	address := "0xDEaD70220f96970E686E56C7ed6b8376bADaAF6e"
	service := NewKeyStoreService(KeyStoreConfig{
		Enabled: true, OutputDirectory: tempDir, KDF: "scrypt",
		// This test asserts on the password filename, so it opts in.
		WritePasswordFile: true,
	})
	keystore := createValidKeyStore(strings.TrimPrefix(address, "0x"))

	if err := service.SaveKeyStoreFilesToDisk(address, keystore, "TestPassword123!", "ethereum", ""); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	entries, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("Failed to read temp directory: %v", err)
	}

	names := map[string]bool{}
	for _, entry := range entries {
		names[entry.Name()] = true
	}

	// The keystore file uses the geth naming convention with a UTC timestamp
	// prefix; verify the address suffix and case are preserved.
	foundKeystore := false
	for name := range names {
		if strings.HasPrefix(name, "UTC--") && strings.HasSuffix(name, "--"+address+".json") {
			foundKeystore = true
			break
		}
	}
	if !foundKeystore {
		t.Fatalf("expected keystore filename UTC--*--%s.json, got %v", address, names)
	}

	if !names[address+".pwd"] {
		t.Fatalf("expected password filename %s.pwd, got %v", address, names)
	}
}

func TestKeyStoreService_SaveMnemonicFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "mnemonic-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	validAddress := "0x1234567890abcdef1234567890abcdef12345678"
	mnemonic := "test walk nut penalty hip pave soap entry language right filter choice"

	t.Run("successfully saves mnemonic file", func(t *testing.T) {
		service := NewKeyStoreService(KeyStoreConfig{Enabled: true, OutputDirectory: tempDir, KDF: "scrypt"})

		if err := service.SaveMnemonicFile(validAddress, mnemonic, "ethereum"); err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		cleanAddress := strings.ToLower(strings.TrimPrefix(validAddress, "0x"))
		mnemonicPath := filepath.Join(tempDir, fmt.Sprintf("0x%s.mnemonic", cleanAddress))

		data, err := os.ReadFile(mnemonicPath)
		if err != nil {
			t.Fatalf("Failed to read mnemonic file: %v", err)
		}

		if string(data) != mnemonic {
			t.Errorf("Mnemonic file content mismatch: expected %q, got %q", mnemonic, string(data))
		}

		info, err := os.Stat(mnemonicPath)
		if err != nil {
			t.Fatalf("Failed to stat mnemonic file: %v", err)
		}

		if info.Mode().Perm() != 0o600 {
			t.Errorf("Mnemonic file permissions mismatch: expected 0600, got %o", info.Mode().Perm())
		}
	})

	t.Run("fails when service disabled", func(t *testing.T) {
		service := NewKeyStoreService(KeyStoreConfig{Enabled: false, OutputDirectory: tempDir, KDF: "scrypt"})

		err := service.SaveMnemonicFile(validAddress, mnemonic, "ethereum")
		if err == nil || !strings.Contains(err.Error(), "keystore generation is disabled") {
			t.Fatalf("Expected disabled service error, got %v", err)
		}
	})

	t.Run("fails with empty mnemonic", func(t *testing.T) {
		service := NewKeyStoreService(KeyStoreConfig{Enabled: true, OutputDirectory: tempDir, KDF: "scrypt"})

		err := service.SaveMnemonicFile(validAddress, "", "ethereum")
		if err == nil || !strings.Contains(err.Error(), "mnemonic cannot be empty") {
			t.Fatalf("Expected empty mnemonic error, got %v", err)
		}
	})

	t.Run("fails with invalid address", func(t *testing.T) {
		service := NewKeyStoreService(KeyStoreConfig{Enabled: true, OutputDirectory: tempDir, KDF: "scrypt"})

		err := service.SaveMnemonicFile("0x1234", mnemonic, "ethereum")
		if err == nil || (!strings.Contains(err.Error(), "invalid address length") && !strings.Contains(err.Error(), "invalid Ethereum address length")) {
			t.Fatalf("Expected invalid address error, got %v", err)
		}
	})
}

func TestKeyStoreService_Configuration(t *testing.T) {
	config := KeyStoreConfig{
		OutputDirectory: "/tmp/test",
		Enabled:         true,
		KDF:             "pbkdf2",
	}
	service := NewKeyStoreService(config)

	// Test GetConfig
	retrievedConfig := service.GetConfig()
	if retrievedConfig.OutputDirectory != config.OutputDirectory {
		t.Errorf("GetConfig output directory mismatch: expected %s, got %s", config.OutputDirectory, retrievedConfig.OutputDirectory)
	}

	// Test SetOutputDirectory
	newDir := "/tmp/new-test"
	service.SetOutputDirectory(newDir)
	if service.GetConfig().OutputDirectory != newDir {
		t.Errorf("SetOutputDirectory failed: expected %s, got %s", newDir, service.GetConfig().OutputDirectory)
	}

	// Test SetEnabled
	service.SetEnabled(false)
	if service.GetConfig().Enabled != false {
		t.Error("SetEnabled failed: expected false, got true")
	}

	// Test SetKDF
	err := service.SetKDF("scrypt")
	if err != nil {
		t.Errorf("SetKDF failed: %v", err)
	}
	if service.GetConfig().KDF != "scrypt" {
		t.Errorf("SetKDF failed: expected scrypt, got %s", service.GetConfig().KDF)
	}

	// Test invalid KDF
	err = service.SetKDF("invalid")
	if err == nil {
		t.Error("SetKDF should fail for invalid KDF")
	}
}

func TestKeyStoreService_EndToEndWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping end-to-end workflow test in short mode")
	}

	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "keystore-e2e-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Test both KDF algorithms
	kdfs := []string{"scrypt", "pbkdf2"}

	for _, kdf := range kdfs {
		t.Run(fmt.Sprintf("end_to_end_%s", kdf), func(t *testing.T) {
			// Use a dedicated directory per KDF so the UTC--* file lookup
			// cannot pick up a file from a previous subtest.
			subDir := filepath.Join(tempDir, kdf)
			if err := os.MkdirAll(subDir, 0755); err != nil {
				t.Fatalf("Failed to create sub directory: %v", err)
			}
			config := KeyStoreConfig{
				OutputDirectory: subDir,
				Enabled:         true,
				KDF:             kdf,
			}
			service := NewKeyStoreService(config)

			privateKey := "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"
			address := "0x1234567890abcdef1234567890abcdef12345678"

			// Generate keystore
			keystore, password, err := service.GenerateKeyStore(privateKey, address, "ethereum")
			if err != nil {
				t.Fatalf("GenerateKeyStore failed: %v", err)
			}

			// Save files
			err = service.SaveKeyStoreFilesToDisk(address, keystore, password, "ethereum", privateKey)
			if err != nil {
				t.Fatalf("SaveKeyStoreFiles failed: %v", err)
			}

			// Verify we can load and decrypt the keystore
			cleanAddress := strings.ToLower(strings.TrimPrefix(address, "0x"))
			keystoreFile := ""
			entries, err := os.ReadDir(subDir)
			if err != nil {
				t.Fatalf("Failed to read output directory: %v", err)
			}
			for _, entry := range entries {
				if strings.HasPrefix(entry.Name(), "UTC--") && strings.HasSuffix(entry.Name(), fmt.Sprintf("--0x%s.json", cleanAddress)) {
					keystoreFile = filepath.Join(subDir, entry.Name())
					break
				}
			}
			if keystoreFile == "" {
				t.Fatalf("Keystore file (UTC--*--0x%s.json) was not created", cleanAddress)
			}
			passwordFile := filepath.Join(subDir, fmt.Sprintf("0x%s.pwd", cleanAddress))

			// Load keystore from file
			keystoreData, err := os.ReadFile(keystoreFile)
			if err != nil {
				t.Fatalf("Failed to read keystore file: %v", err)
			}

			loadedKeystore, err := FromJSON(keystoreData)
			if err != nil {
				t.Fatalf("Failed to parse keystore JSON: %v", err)
			}

			// Load password from file
			passwordData, err := os.ReadFile(passwordFile)
			if err != nil {
				t.Fatalf("Failed to read password file: %v", err)
			}

			loadedPassword := string(passwordData)

			// Decrypt private key
			decryptedKey, err := DecryptPrivateKey(loadedKeystore, loadedPassword)
			if err != nil {
				t.Fatalf("Failed to decrypt private key: %v", err)
			}

			// Verify private key matches
			decryptedHex := hex.EncodeToString(decryptedKey)
			if decryptedHex != privateKey {
				t.Errorf("Private key mismatch: expected %s, got %s", privateKey, decryptedHex)
			}

			// Verify KDF was used correctly
			if loadedKeystore.Crypto.KDF != kdf {
				t.Errorf("KDF mismatch: expected %s, got %s", kdf, loadedKeystore.Crypto.KDF)
			}
		})
	}
}

// Helper function to create a valid keystore for testing
func createValidKeyStore(address string) *KeyStoreV3 {
	ks := NewKeyStoreV3(address, "ethereum")

	// Set some dummy but valid crypto parameters
	ks.SetScryptParams(262144, 8, 1, 32, []byte("test-salt-32-bytes-long-for-test"))
	ks.SetCipherParams([]byte("test-iv-16-bytes"), []byte("dummy-ciphertext-32-bytes-long!!"))
	ks.SetMAC([]byte("dummy-mac-32-bytes-long-for-test"))

	return ks
}

// File Management Tests

func TestKeyStoreService_EnsureOutputDirectory(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func() (*KeyStoreService, string, func())
		wantError bool
		errorMsg  string
	}{
		{
			name: "create new directory",
			setupFunc: func() (*KeyStoreService, string, func()) {
				tmpDir := filepath.Join(os.TempDir(), fmt.Sprintf("keystore-test-%d", time.Now().UnixNano()))
				config := KeyStoreConfig{
					OutputDirectory: tmpDir,
					Enabled:         true,
				}
				service := NewKeyStoreService(config)
				cleanup := func() { _ = os.RemoveAll(tmpDir) }
				return service, tmpDir, cleanup
			},
			wantError: false,
		},
		{
			name: "directory already exists",
			setupFunc: func() (*KeyStoreService, string, func()) {
				tmpDir := filepath.Join(os.TempDir(), fmt.Sprintf("keystore-test-%d", time.Now().UnixNano()))
				if err := os.MkdirAll(tmpDir, 0755); err != nil {
					panic(fmt.Sprintf("Failed to create directory: %v", err))
				}
				config := KeyStoreConfig{
					OutputDirectory: tmpDir,
					Enabled:         true,
				}
				service := NewKeyStoreService(config)
				cleanup := func() { _ = os.RemoveAll(tmpDir) }
				return service, tmpDir, cleanup
			},
			wantError: false,
		},
		{
			name: "path exists but is not directory",
			setupFunc: func() (*KeyStoreService, string, func()) {
				tmpFile := filepath.Join(os.TempDir(), fmt.Sprintf("keystore-test-file-%d", time.Now().UnixNano()))
				if err := os.WriteFile(tmpFile, []byte("test"), 0644); err != nil {
					panic(fmt.Sprintf("Failed to write file: %v", err))
				}
				config := KeyStoreConfig{
					OutputDirectory: tmpFile,
					Enabled:         true,
				}
				service := NewKeyStoreService(config)
				cleanup := func() { _ = os.Remove(tmpFile) }
				return service, tmpFile, cleanup
			},
			wantError: true,
			errorMsg:  "exists but is not a directory",
		},
		{
			name: "empty directory path after manual override",
			setupFunc: func() (*KeyStoreService, string, func()) {
				config := KeyStoreConfig{
					OutputDirectory: "./keystores", // NewKeyStoreService sets default
					Enabled:         true,
				}
				service := NewKeyStoreService(config)
				// Manually override to empty to test the ensureOutputDirectory validation
				service.config.OutputDirectory = ""
				return service, "", func() {}
			},
			wantError: true,
			errorMsg:  "output directory cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, expectedPath, cleanup := tt.setupFunc()
			defer cleanup()

			err := service.ensureOutputDirectory()

			if tt.wantError {
				if err == nil {
					t.Error("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
				// Verify directory exists
				if info, err := os.Stat(expectedPath); err != nil {
					t.Errorf("Directory should exist: %v", err)
				} else if !info.IsDir() {
					t.Error("Path should be a directory")
				}
			}
		})
	}
}

func TestKeyStoreService_WriteFileAtomic(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func() (*KeyStoreService, string, []byte, os.FileMode, func())
		wantError bool
		errorMsg  string
	}{
		{
			name: "successful write",
			setupFunc: func() (*KeyStoreService, string, []byte, os.FileMode, func()) {
				tmpDir := filepath.Join(os.TempDir(), fmt.Sprintf("keystore-test-%d", time.Now().UnixNano()))
				if err := os.MkdirAll(tmpDir, 0755); err != nil {
					panic(fmt.Sprintf("Failed to create directory: %v", err))
				}
				config := KeyStoreConfig{OutputDirectory: tmpDir, Enabled: true}
				service := NewKeyStoreService(config)
				filename := filepath.Join(tmpDir, "test.json")
				data := []byte(`{"test": "data"}`)
				cleanup := func() { _ = os.RemoveAll(tmpDir) }
				return service, filename, data, 0600, cleanup
			},
			wantError: false,
		},
		{
			name: "overwrite existing file",
			setupFunc: func() (*KeyStoreService, string, []byte, os.FileMode, func()) {
				tmpDir := filepath.Join(os.TempDir(), fmt.Sprintf("keystore-test-%d", time.Now().UnixNano()))
				if err := os.MkdirAll(tmpDir, 0755); err != nil {
					panic(fmt.Sprintf("Failed to create directory: %v", err))
				}
				filename := filepath.Join(tmpDir, "test.json")
				if err := os.WriteFile(filename, []byte("old data"), 0644); err != nil {
					panic(fmt.Sprintf("Failed to write file: %v", err))
				}
				config := KeyStoreConfig{OutputDirectory: tmpDir, Enabled: true}
				service := NewKeyStoreService(config)
				data := []byte(`{"test": "new data"}`)
				cleanup := func() { _ = os.RemoveAll(tmpDir) }
				return service, filename, data, 0600, cleanup
			},
			wantError: false,
		},
		{
			name: "empty filename",
			setupFunc: func() (*KeyStoreService, string, []byte, os.FileMode, func()) {
				config := KeyStoreConfig{Enabled: true}
				service := NewKeyStoreService(config)
				data := []byte(`{"test": "data"}`)
				return service, "", data, 0600, func() {}
			},
			wantError: true,
			errorMsg:  "filename cannot be empty",
		},
		{
			name: "nil data",
			setupFunc: func() (*KeyStoreService, string, []byte, os.FileMode, func()) {
				tmpDir := filepath.Join(os.TempDir(), fmt.Sprintf("keystore-test-%d", time.Now().UnixNano()))
				config := KeyStoreConfig{OutputDirectory: tmpDir, Enabled: true}
				service := NewKeyStoreService(config)
				filename := filepath.Join(tmpDir, "test.json")
				cleanup := func() { _ = os.RemoveAll(tmpDir) }
				return service, filename, nil, 0600, cleanup
			},
			wantError: true,
			errorMsg:  "data cannot be nil",
		},
		{
			name: "zero permissions",
			setupFunc: func() (*KeyStoreService, string, []byte, os.FileMode, func()) {
				tmpDir := filepath.Join(os.TempDir(), fmt.Sprintf("keystore-test-%d", time.Now().UnixNano()))
				config := KeyStoreConfig{OutputDirectory: tmpDir, Enabled: true}
				service := NewKeyStoreService(config)
				filename := filepath.Join(tmpDir, "test.json")
				data := []byte(`{"test": "data"}`)
				cleanup := func() { _ = os.RemoveAll(tmpDir) }
				return service, filename, data, 0, cleanup
			},
			wantError: true,
			errorMsg:  "file permissions cannot be zero",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, filename, data, perm, cleanup := tt.setupFunc()
			defer cleanup()

			err := service.writeFileAtomic(filename, data, perm)

			if tt.wantError {
				if err == nil {
					t.Error("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}

				// Verify file exists and has correct content
				if content, err := os.ReadFile(filename); err != nil {
					t.Errorf("Failed to read written file: %v", err)
				} else if !bytes.Equal(content, data) {
					t.Errorf("File content mismatch. Expected: %s, Got: %s", string(data), string(content))
				}

				// Verify file permissions
				if info, err := os.Stat(filename); err != nil {
					t.Errorf("Failed to stat written file: %v", err)
				} else if info.Mode().Perm() != perm {
					t.Errorf("File permissions mismatch. Expected: %o, Got: %o", perm, info.Mode().Perm())
				}
			}
		})
	}
}

func TestKeyStoreService_CheckDirectoryPermissions(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func() (*KeyStoreService, func())
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid writable directory",
			setupFunc: func() (*KeyStoreService, func()) {
				tmpDir := filepath.Join(os.TempDir(), fmt.Sprintf("keystore-test-%d", time.Now().UnixNano()))
				if err := os.MkdirAll(tmpDir, 0755); err != nil {
					panic(fmt.Sprintf("Failed to create directory: %v", err))
				}
				config := KeyStoreConfig{OutputDirectory: tmpDir, Enabled: true}
				service := NewKeyStoreService(config)
				cleanup := func() { _ = os.RemoveAll(tmpDir) }
				return service, cleanup
			},
			wantError: false,
		},
		{
			name: "disabled service",
			setupFunc: func() (*KeyStoreService, func()) {
				config := KeyStoreConfig{OutputDirectory: "/nonexistent", Enabled: false}
				service := NewKeyStoreService(config)
				return service, func() {}
			},
			wantError: false,
		},
		{
			name: "nonexistent directory",
			setupFunc: func() (*KeyStoreService, func()) {
				config := KeyStoreConfig{OutputDirectory: "/nonexistent/path", Enabled: true}
				service := NewKeyStoreService(config)
				return service, func() {}
			},
			wantError: true,
			errorMsg:  "directory does not exist",
		},
		{
			name: "path is not directory",
			setupFunc: func() (*KeyStoreService, func()) {
				tmpFile := filepath.Join(os.TempDir(), fmt.Sprintf("keystore-test-file-%d", time.Now().UnixNano()))
				if err := os.WriteFile(tmpFile, []byte("test"), 0644); err != nil {
					panic(fmt.Sprintf("Failed to write file: %v", err))
				}
				config := KeyStoreConfig{OutputDirectory: tmpFile, Enabled: true}
				service := NewKeyStoreService(config)
				cleanup := func() { _ = os.Remove(tmpFile) }
				return service, cleanup
			},
			wantError: true,
			errorMsg:  "path is not a directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, cleanup := tt.setupFunc()
			defer cleanup()

			err := service.CheckDirectoryPermissions()

			if tt.wantError {
				if err == nil {
					t.Error("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestKeyStoreService_FileExists(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), fmt.Sprintf("keystore-test-%d", time.Now().UnixNano()))
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	config := KeyStoreConfig{OutputDirectory: tmpDir, Enabled: true}
	service := NewKeyStoreService(config)

	existingFile := filepath.Join(tmpDir, "existing.json")
	if err := os.WriteFile(existingFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	nonExistentFile := filepath.Join(tmpDir, "nonexistent.json")

	tests := []struct {
		name       string
		filename   string
		wantExists bool
		wantError  bool
		errorMsg   string
	}{
		{
			name:       "existing file",
			filename:   existingFile,
			wantExists: true,
			wantError:  false,
		},
		{
			name:       "nonexistent file",
			filename:   nonExistentFile,
			wantExists: false,
			wantError:  false,
		},
		{
			name:      "empty filename",
			filename:  "",
			wantError: true,
			errorMsg:  "filename cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exists, err := service.FileExists(tt.filename)

			if tt.wantError {
				if err == nil {
					t.Error("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
				if exists != tt.wantExists {
					t.Errorf("Expected exists=%v, got %v", tt.wantExists, exists)
				}
			}
		})
	}
}

func TestKeyStoreService_GetFilePaths(t *testing.T) {
	tmpDir := t.TempDir()
	config := KeyStoreConfig{OutputDirectory: tmpDir, Enabled: true}
	service := NewKeyStoreService(config)

	tests := []struct {
		name             string
		address          string
		wantKeystoreName string
		wantPasswordPath string
		wantMnemonicPath string
		wantError        bool
		errorMsg         string
	}{
		{
			name:             "valid address with 0x prefix",
			address:          "0x1234567890abcdef1234567890abcdef12345678",
			wantKeystoreName: "UTC--2025-01-01T00-00-00.000000000Z--0x1234567890abcdef1234567890abcdef12345678.json",
			wantPasswordPath: filepath.Join(tmpDir, "0x1234567890abcdef1234567890abcdef12345678.pwd"),
			wantMnemonicPath: filepath.Join(tmpDir, "0x1234567890abcdef1234567890abcdef12345678.mnemonic"),
			wantError:        false,
		},
		{
			name:             "valid address without 0x prefix",
			address:          "1234567890abcdef1234567890abcdef12345678",
			wantKeystoreName: "UTC--2025-01-01T00-00-00.000000000Z--0x1234567890abcdef1234567890abcdef12345678.json",
			wantPasswordPath: filepath.Join(tmpDir, "0x1234567890abcdef1234567890abcdef12345678.pwd"),
			wantMnemonicPath: filepath.Join(tmpDir, "0x1234567890abcdef1234567890abcdef12345678.mnemonic"),
			wantError:        false,
		},
		{
			name:             "mixed case ethereum address",
			address:          "0xDEaD70220f96970E686E56C7ed6b8376bADaAF6e",
			wantKeystoreName: "UTC--2025-01-01T00-00-00.000000000Z--0xDEaD70220f96970E686E56C7ed6b8376bADaAF6e.json",
			wantPasswordPath: filepath.Join(tmpDir, "0xDEaD70220f96970E686E56C7ed6b8376bADaAF6e.pwd"),
			wantMnemonicPath: filepath.Join(tmpDir, "0xDEaD70220f96970E686E56C7ed6b8376bADaAF6e.mnemonic"),
			wantError:        false,
		},

		{
			name:      "empty address",
			address:   "",
			wantError: true,
			errorMsg:  "address cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a keystore file matching the new naming convention so the
			// directory scan can find it.
			if !tt.wantError {
				keystoreName := tt.wantKeystoreName
				if err := os.WriteFile(filepath.Join(tmpDir, keystoreName), []byte(`{}`), 0600); err != nil {
					t.Fatalf("Failed to write keystore file: %v", err)
				}
			}

			keystorePath, err := service.GetKeystoreFilePath(tt.address)
			if tt.wantError {
				if err == nil {
					t.Error("Expected error for keystore path but got none")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected keystore error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for keystore path but got: %v", err)
				}
				wantKeystorePath := filepath.Join(tmpDir, tt.wantKeystoreName)
				if keystorePath != wantKeystorePath {
					t.Errorf("Expected keystore path %s, got %s", wantKeystorePath, keystorePath)
				}
			}

			passwordPath, err := service.GetPasswordFilePath(tt.address)
			if tt.wantError {
				if err == nil {
					t.Error("Expected error for password path but got none")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected password error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for password path but got: %v", err)
				}
				if passwordPath != tt.wantPasswordPath {
					t.Errorf("Expected password path %s, got %s", tt.wantPasswordPath, passwordPath)
				}
			}

			mnemonicPath, err := service.GetMnemonicFilePath(tt.address)
			if tt.wantError {
				if err == nil {
					t.Error("Expected error for mnemonic path but got none")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected mnemonic error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for mnemonic path but got: %v", err)
				}
				if mnemonicPath != tt.wantMnemonicPath {
					t.Errorf("Expected mnemonic path %s, got %s", tt.wantMnemonicPath, mnemonicPath)
				}
			}
		})
	}
}

func TestKeyStoreService_RemoveKeystoreFiles(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), fmt.Sprintf("keystore-test-%d", time.Now().UnixNano()))
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	config := KeyStoreConfig{OutputDirectory: tmpDir, Enabled: true}
	service := NewKeyStoreService(config)

	address := "1234567890abcdef1234567890abcdef12345678"
	keystorePath := filepath.Join(tmpDir, "0x1234567890abcdef1234567890abcdef12345678.json")
	passwordPath := filepath.Join(tmpDir, "0x1234567890abcdef1234567890abcdef12345678.pwd")

	// Create test files
	if err := os.WriteFile(keystorePath, []byte(`{"test": "keystore"}`), 0600); err != nil {
		t.Fatalf("Failed to write keystore file: %v", err)
	}
	if err := os.WriteFile(passwordPath, []byte("testpassword"), 0600); err != nil {
		t.Fatalf("Failed to write password file: %v", err)
	}

	// Verify files exist
	if _, err := os.Stat(keystorePath); err != nil {
		t.Fatalf("Test keystore file should exist: %v", err)
	}
	if _, err := os.Stat(passwordPath); err != nil {
		t.Fatalf("Test password file should exist: %v", err)
	}

	// Remove files
	err := service.RemoveKeystoreFiles(address)
	if err != nil {
		t.Errorf("Expected no error but got: %v", err)
	}

	// Verify files are removed
	if _, err := os.Stat(keystorePath); !os.IsNotExist(err) {
		t.Error("Keystore file should be removed")
	}
	if _, err := os.Stat(passwordPath); !os.IsNotExist(err) {
		t.Error("Password file should be removed")
	}

	// Test removing non-existent files (should not error)
	err = service.RemoveKeystoreFiles(address)
	if err != nil {
		t.Errorf("Expected no error when removing non-existent files but got: %v", err)
	}

	// Test with disabled service
	disabledService := NewKeyStoreService(KeyStoreConfig{Enabled: false})
	err = disabledService.RemoveKeystoreFiles(address)
	if err == nil {
		t.Error("Expected error with disabled service but got none")
	}
}

func TestKeyStoreService_ValidateFilePermissions(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), fmt.Sprintf("keystore-test-%d", time.Now().UnixNano()))
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	config := KeyStoreConfig{OutputDirectory: tmpDir, Enabled: true}
	service := NewKeyStoreService(config)

	// Create test file with specific permissions
	testFile := filepath.Join(tmpDir, "test.json")
	if err := os.WriteFile(testFile, []byte("test"), 0600); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	tests := []struct {
		name         string
		filename     string
		expectedPerm os.FileMode
		wantError    bool
		errorMsg     string
	}{
		{
			name:         "correct permissions",
			filename:     testFile,
			expectedPerm: 0600,
			wantError:    false,
		},
		{
			name:         "incorrect permissions",
			filename:     testFile,
			expectedPerm: 0644,
			wantError:    true,
			errorMsg:     "incorrect permissions",
		},
		{
			name:         "nonexistent file",
			filename:     filepath.Join(tmpDir, "nonexistent.json"),
			expectedPerm: 0600,
			wantError:    true,
		},
		{
			name:         "empty filename",
			filename:     "",
			expectedPerm: 0600,
			wantError:    true,
			errorMsg:     "filename cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.ValidateFilePermissions(tt.filename, tt.expectedPerm)

			if tt.wantError {
				if err == nil {
					t.Error("Expected error but got none")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestFileOperationError(t *testing.T) {
	baseErr := fmt.Errorf("base error")
	fileErr := &FileOperationError{
		Operation: "test_operation",
		Path:      "/test/path",
		Err:       baseErr,
	}

	expectedMsg := "file operation 'test_operation' failed for path '/test/path': base error"
	if fileErr.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, fileErr.Error())
	}

	if fileErr.Unwrap() != baseErr {
		t.Errorf("Expected unwrapped error to be base error, got %v", fileErr.Unwrap())
	}
}

func TestKeyStoreService_SaveKeyStoreFiles_EnhancedErrorHandling(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), fmt.Sprintf("keystore-test-%d", time.Now().UnixNano()))
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	config := KeyStoreConfig{OutputDirectory: tmpDir, Enabled: true}
	service := NewKeyStoreService(config)

	// Create a valid keystore for testing
	address := "1234567890abcdef1234567890abcdef12345678"
	keystore := NewKeyStoreV3("0x"+address, "ethereum")
	keystore.SetScryptParams(262144, 8, 1, 32, bytes32(7))
	keystore.SetCipherParams(bytes16(8), bytes32(9))
	keystore.SetMAC(bytes32(10))
	password := "TestPassword123!"

	// Test successful save
	err := service.SaveKeyStoreFilesToDisk(address, keystore, password, "ethereum", "")
	if err != nil {
		t.Errorf("Expected no error but got: %v", err)
	}

	// Verify files exist with correct permissions (UTC-- naming convention)
	var keystorePath string
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to read output directory: %v", err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "UTC--") && strings.HasSuffix(entry.Name(), "--0x"+address+".json") {
			keystorePath = filepath.Join(tmpDir, entry.Name())
			break
		}
	}
	if keystorePath == "" {
		t.Fatalf("Keystore file (UTC--*--0x%s.json) was not created", address)
	}
	passwordPath := filepath.Join(tmpDir, "0x1234567890abcdef1234567890abcdef12345678.pwd")

	if info, err := os.Stat(keystorePath); err != nil {
		t.Errorf("Keystore file should exist: %v", err)
	} else if info.Mode().Perm() != 0600 {
		t.Errorf("Keystore file should have 0600 permissions, got %o", info.Mode().Perm())
	}

	// The password file is opt-in: by default it must NOT be written next to
	// the keystore it unlocks.
	if _, err := os.Stat(passwordPath); !os.IsNotExist(err) {
		t.Errorf("Password file should not be written without WritePasswordFile, stat error was: %v", err)
	}

	// Test overwriting existing files
	err = service.SaveKeyStoreFilesToDisk(address, keystore, password, "ethereum", "")
	if err != nil {
		t.Errorf("Expected no error when overwriting files but got: %v", err)
	}
}

// TestKeyStoreError tests the KeyStoreError type and its methods
func TestKeyStoreError(t *testing.T) {
	tests := []struct {
		name        string
		err         *KeyStoreError
		expectedMsg string
		recoverable bool
	}{
		{
			name: "basic error",
			err: &KeyStoreError{
				Operation:  "encrypt",
				Component:  "private_key",
				Underlying: fmt.Errorf("invalid key"),
			},
			expectedMsg: "keystore encrypt failed for private_key: invalid key",
			recoverable: false,
		},
		{
			name: "error with address",
			err: &KeyStoreError{
				Operation:  "save",
				Component:  "keystore",
				Address:    "0x1234567890abcdef1234567890abcdef12345678",
				Underlying: fmt.Errorf("disk full"),
			},
			expectedMsg: "keystore save failed for keystore (address: 0x1234567890abcdef1234567890abcdef12345678): disk full",
			recoverable: false,
		},
		{
			name: "error with path",
			err: &KeyStoreError{
				Operation:  "write",
				Component:  "file",
				Path:       "/tmp/keystore.json",
				Underlying: fmt.Errorf("permission denied"),
			},
			expectedMsg: "keystore write failed for file (path: /tmp/keystore.json): permission denied",
			recoverable: false,
		},
		{
			name: "recoverable error with user message",
			err: &KeyStoreError{
				Operation:   "save",
				Component:   "directory",
				Underlying:  fmt.Errorf("temporary failure"),
				Recoverable: true,
				UserMessage: "Directory is temporarily unavailable. Please try again.",
			},
			expectedMsg: "Directory is temporarily unavailable. Please try again.",
			recoverable: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.expectedMsg {
				t.Errorf("Expected error message '%s', got '%s'", tt.expectedMsg, tt.err.Error())
			}

			if tt.err.IsRecoverable() != tt.recoverable {
				t.Errorf("Expected recoverable=%v, got %v", tt.recoverable, tt.err.IsRecoverable())
			}

			if tt.err.Unwrap() != tt.err.Underlying {
				t.Errorf("Unwrap() should return the underlying error")
			}
		})
	}
}

// TestKeyStoreErrorConstructors tests the error constructor functions
func TestKeyStoreErrorConstructors(t *testing.T) {
	underlyingErr := fmt.Errorf("test error")

	// Test NewKeyStoreError
	err1 := NewKeyStoreError("test_op", "test_comp", underlyingErr)
	if err1.Operation != "test_op" || err1.Component != "test_comp" || err1.Underlying != underlyingErr {
		t.Errorf("NewKeyStoreError did not set fields correctly")
	}

	// Test NewKeyStoreErrorWithAddress
	err2 := NewKeyStoreErrorWithAddress("test_op", "test_comp", "0x123", underlyingErr)
	if err2.Address != "0x123" {
		t.Errorf("NewKeyStoreErrorWithAddress did not set address correctly")
	}

	// Test NewKeyStoreErrorWithPath
	err3 := NewKeyStoreErrorWithPath("test_op", "test_comp", "/tmp/test", underlyingErr)
	if err3.Path != "/tmp/test" {
		t.Errorf("NewKeyStoreErrorWithPath did not set path correctly")
	}

	// Test NewRecoverableKeyStoreError
	err4 := NewRecoverableKeyStoreError("test_op", "test_comp", underlyingErr, "user message")
	if !err4.Recoverable || err4.UserMessage != "user message" {
		t.Errorf("NewRecoverableKeyStoreError did not set recoverable fields correctly")
	}
}

// TestKeyStoreServiceRetryMechanism tests the retry functionality
func TestKeyStoreServiceRetryMechanism(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name          string
		config        KeyStoreConfig
		privateKey    string
		address       string
		setupFunc     func(string) error // Function to set up test conditions
		cleanupFunc   func(string) error // Function to clean up after test
		expectSuccess bool
	}{
		{
			name: "success on first attempt",
			config: KeyStoreConfig{
				Enabled:         true,
				OutputDirectory: tempDir,
				KDF:             "pbkdf2",
				MaxRetries:      3,
				RetryDelay:      10, // Short delay for tests
			},
			privateKey:    "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			address:       "0x1234567890abcdef1234567890abcdef12345678",
			expectSuccess: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewKeyStoreService(tt.config)

			// Set up test conditions
			if tt.setupFunc != nil {
				if err := tt.setupFunc(tt.config.OutputDirectory); err != nil {
					t.Fatalf("Setup failed: %v", err)
				}
			}

			// Start cleanup in background if needed
			if tt.cleanupFunc != nil {
				go func() {
					time.Sleep(20 * time.Millisecond) // Allow first attempt to start
					_ = tt.cleanupFunc(tt.config.OutputDirectory)
				}()
			}

			// Test the retry mechanism
			err := service.SaveKeyStoreFilesWithRetry(tt.privateKey, tt.address, "ethereum")

			if tt.expectSuccess && err != nil {
				t.Errorf("Expected success but got error: %v", err)
			} else if !tt.expectSuccess && err == nil {
				t.Errorf("Expected error but got success")
			}

			// Clean up
			if tt.cleanupFunc != nil {
				_ = tt.cleanupFunc(tt.config.OutputDirectory)
			}
		})
	}
}

// TestLogger is a test implementation of ProgressLogger
type TestLogger struct {
	InfoMessages    []string
	WarningMessages []string
	ErrorMessages   []string
	DebugMessages   []string
}

// Implement ProgressLogger interface methods
func (l *TestLogger) LogInfo(message string) {
	l.InfoMessages = append(l.InfoMessages, message)
}
func (l *TestLogger) LogWarning(message string) {
	l.WarningMessages = append(l.WarningMessages, message)
}
func (l *TestLogger) LogError(message string) {
	l.ErrorMessages = append(l.ErrorMessages, message)
}
func (l *TestLogger) LogDebug(message string) {
	l.DebugMessages = append(l.DebugMessages, message)
}

// TestKeyStoreServiceProgressLogging tests the progress logging functionality
func TestKeyStoreServiceProgressLogging(t *testing.T) {
	tempDir := t.TempDir()

	logger := &TestLogger{}

	config := KeyStoreConfig{
		Enabled:         true,
		OutputDirectory: tempDir,
		KDF:             "pbkdf2",
		MaxRetries:      1,
		RetryDelay:      10,
	}

	service := NewKeyStoreServiceWithLogger(config, logger)

	privateKey := "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"
	address := "0x1234567890abcdef1234567890abcdef12345678"

	// Test successful operation
	err := service.SaveKeyStoreFiles(privateKey, address, "ethereum")
	if err != nil {
		t.Fatalf("SaveKeyStoreFiles failed: %v", err)
	}

	// Check that appropriate log messages were generated
	if len(logger.InfoMessages) == 0 {
		t.Errorf("Expected info messages but got none")
	}

	if len(logger.DebugMessages) == 0 {
		t.Errorf("Expected debug messages but got none")
	}

	// Test verbose mode
	service.SetVerboseMode(true)

	// Clear previous messages
	logger.InfoMessages = nil
	logger.DebugMessages = nil

	// Generate another wallet
	privateKey2 := "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"
	address2 := "0xabcdef1234567890abcdef1234567890abcdef12"

	err = service.SaveKeyStoreFiles(privateKey2, address2, "ethereum")
	if err != nil {
		t.Fatalf("SaveKeyStoreFiles failed: %v", err)
	}

	// Should have more debug messages in verbose mode
	if len(logger.DebugMessages) == 0 {
		t.Errorf("Expected debug messages in verbose mode but got none")
	}
}

// TestKeyStoreServiceIntegrationWithGeth tests compatibility with geth format
func TestKeyStoreServiceIntegrationWithGeth(t *testing.T) {
	tempDir := t.TempDir()

	config := KeyStoreConfig{
		Enabled:         true,
		OutputDirectory: tempDir,
		KDF:             "scrypt", // Use scrypt for geth compatibility
		MaxRetries:      1,
		RetryDelay:      10,
	}

	service := NewKeyStoreService(config)

	privateKey := "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"
	address := "0x1234567890abcdef1234567890abcdef12345678"

	// Generate keystore
	err := service.SaveKeyStoreFiles(privateKey, address, "ethereum")
	if err != nil {
		t.Fatalf("SaveKeyStoreFiles failed: %v", err)
	}

	// Read the generated keystore file
	keystorePath, err := service.GetKeystoreFilePath(address)
	if err != nil {
		t.Fatalf("GetKeystoreFilePath failed: %v", err)
	}

	keystoreData, err := os.ReadFile(keystorePath)
	if err != nil {
		t.Fatalf("Failed to read keystore file: %v", err)
	}

	// Parse the keystore JSON
	var keystore map[string]interface{}
	if err := json.Unmarshal(keystoreData, &keystore); err != nil {
		t.Fatalf("Failed to parse keystore JSON: %v", err)
	}

	// Verify geth-compatible format
	requiredFields := []string{"address", "crypto", "id", "version"}
	for _, field := range requiredFields {
		if _, exists := keystore[field]; !exists {
			t.Errorf("Missing required field: %s", field)
		}
	}

	// Verify version is 3
	if version, ok := keystore["version"].(float64); !ok || version != 3 {
		t.Errorf("Expected version 3, got %v", keystore["version"])
	}

	// Verify crypto structure
	crypto, ok := keystore["crypto"].(map[string]interface{})
	if !ok {
		t.Fatalf("Crypto field is not a map")
	}

	cryptoFields := []string{"cipher", "ciphertext", "cipherparams", "kdf", "kdfparams", "mac"}
	for _, field := range cryptoFields {
		if _, exists := crypto[field]; !exists {
			t.Errorf("Missing crypto field: %s", field)
		}
	}

	// Verify cipher is AES-128-CTR
	if cipher, ok := crypto["cipher"].(string); !ok || cipher != "aes-128-ctr" {
		t.Errorf("Expected cipher 'aes-128-ctr', got %v", crypto["cipher"])
	}

	// Verify KDF is scrypt
	if kdf, ok := crypto["kdf"].(string); !ok || kdf != "scrypt" {
		t.Errorf("Expected KDF 'scrypt', got %v", crypto["kdf"])
	}
}

// TestKeyStoreServicePerformanceBenchmark benchmarks keystore operations
func TestKeyStoreServicePerformanceBenchmark(t *testing.T) {
	tempDir := t.TempDir()

	config := KeyStoreConfig{
		Enabled:         true,
		OutputDirectory: tempDir,
		KDF:             "pbkdf2", // Faster than scrypt for benchmarking
		MaxRetries:      1,
		RetryDelay:      10,
	}

	service := NewKeyStoreService(config)

	// Benchmark keystore generation with valid secp256k1 private keys
	start := time.Now()
	numOperations := 10

	for i := 0; i < numOperations; i++ {
		// Valid private key: 0x01..0x0a in the last byte, zero elsewhere.
		privateKeyBytes := bytes32(0)
		privateKeyBytes[31] = byte(i + 1)
		privateKey := hex.EncodeToString(privateKeyBytes)

		err := service.SaveKeyStoreFiles(privateKey, fmt.Sprintf("0x%040d", i+1), "ethereum")
		if err != nil {
			t.Fatalf("SaveKeyStoreFiles failed on iteration %d: %v", i, err)
		}
	}

	duration := time.Since(start)
	avgDuration := duration / time.Duration(numOperations)

	t.Logf("Generated %d keystores in %v (avg: %v per keystore)", numOperations, duration, avgDuration)

	// Verify all files were created
	files, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("Failed to read temp directory: %v", err)
	}

	// One keystore per operation; the password file is opt-in and this service
	// does not request it.
	expectedFiles := numOperations
	if len(files) != expectedFiles {
		t.Errorf("Expected %d files, got %d", expectedFiles, len(files))
	}

	// Performance threshold (adjust based on system capabilities)
	maxAvgDuration := 1000 * time.Millisecond
	if avgDuration > maxAvgDuration {
		t.Errorf("Average keystore generation time %v exceeds threshold %v", avgDuration, maxAvgDuration)
	}
}
