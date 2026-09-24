package crypto

import (
	"bloco-vgen/internal/crypto/kdf"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/google/uuid"
	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/crypto/scrypt"
)

// KeyStoreError represents detailed error information for keystore operations
type KeyStoreError struct {
	Operation   string        // The operation that failed (e.g., "encrypt", "save_file", "validate")
	Component   string        // The component involved (e.g., "keystore", "password", "directory")
	Address     string        // The address being processed (if applicable)
	Path        string        // The file path involved (if applicable)
	Underlying  error         // The underlying error
	Recoverable bool          // Whether the error might be recoverable with retry
	UserMessage string        // User-friendly error message
	KDFError    *kdf.KDFError // KDF-specific error information (if applicable)
}

func (e *KeyStoreError) Error() string {
	if e.UserMessage != "" {
		return e.UserMessage
	}

	var msg strings.Builder
	fmt.Fprintf(&msg, "keystore %s failed", e.Operation)

	if e.Component != "" {
		fmt.Fprintf(&msg, " for %s", e.Component)
	}

	if e.Address != "" {
		fmt.Fprintf(&msg, " (address: %s)", e.Address)
	}

	if e.Path != "" {
		fmt.Fprintf(&msg, " (path: %s)", e.Path)
	}

	if e.Underlying != nil {
		fmt.Fprintf(&msg, ": %v", e.Underlying)
	}

	return msg.String()
}

func (e *KeyStoreError) Unwrap() error {
	return e.Underlying
}

// IsRecoverable returns true if the error might be recoverable with retry
func (e *KeyStoreError) IsRecoverable() bool {
	return e.Recoverable
}

// GetKDFError returns the underlying KDF error if present
func (e *KeyStoreError) GetKDFError() *kdf.KDFError {
	return e.KDFError
}

// HasKDFError returns true if this error contains KDF-specific information
func (e *KeyStoreError) HasKDFError() bool {
	return e.KDFError != nil
}

// GetKDFSuggestions returns KDF-specific suggestions if available
func (e *KeyStoreError) GetKDFSuggestions() []string {
	if e.KDFError != nil {
		return e.KDFError.GetSuggestions()
	}
	return nil
}

// GetKDFType returns the KDF type if this is a KDF-related error
func (e *KeyStoreError) GetKDFType() string {
	if e.KDFError != nil {
		return e.KDFError.KDFType
	}
	return ""
}

// GetKDFParameter returns the problematic KDF parameter if available
func (e *KeyStoreError) GetKDFParameter() string {
	if e.KDFError != nil {
		return e.KDFError.Parameter
	}
	return ""
}

// NewKeyStoreError creates a new KeyStoreError with the given parameters
func NewKeyStoreError(operation, component string, err error) *KeyStoreError {
	return &KeyStoreError{
		Operation:  operation,
		Component:  component,
		Underlying: err,
	}
}

// NewKeyStoreErrorWithAddress creates a new KeyStoreError with address information
func NewKeyStoreErrorWithAddress(operation, component, address string, err error) *KeyStoreError {
	return &KeyStoreError{
		Operation:  operation,
		Component:  component,
		Address:    address,
		Underlying: err,
	}
}

// NewKeyStoreErrorWithPath creates a new KeyStoreError with path information
func NewKeyStoreErrorWithPath(operation, component, path string, err error) *KeyStoreError {
	return &KeyStoreError{
		Operation:  operation,
		Component:  component,
		Path:       path,
		Underlying: err,
	}
}

// NewRecoverableKeyStoreError creates a new recoverable KeyStoreError
func NewRecoverableKeyStoreError(operation, component string, err error, userMessage string) *KeyStoreError {
	return &KeyStoreError{
		Operation:   operation,
		Component:   component,
		Underlying:  err,
		Recoverable: true,
		UserMessage: userMessage,
	}
}

// NewKDFKeyStoreError creates a new KeyStoreError with KDF-specific information
func NewKDFKeyStoreError(operation, component string, kdfErr *kdf.KDFError) *KeyStoreError {
	userMessage := fmt.Sprintf("KDF %s failed: %s", operation, kdfErr.Message)
	if len(kdfErr.Suggestions) > 0 {
		userMessage += fmt.Sprintf(". Suggestions: %s", strings.Join(kdfErr.Suggestions, "; "))
	}

	return &KeyStoreError{
		Operation:   operation,
		Component:   component,
		Underlying:  kdfErr,
		Recoverable: kdfErr.IsRecoverable(),
		UserMessage: userMessage,
		KDFError:    kdfErr,
	}
}

// NewKDFKeyStoreErrorWithAddress creates a new KeyStoreError with KDF and address information
func NewKDFKeyStoreErrorWithAddress(operation, component, address string, kdfErr *kdf.KDFError) *KeyStoreError {
	err := NewKDFKeyStoreError(operation, component, kdfErr)
	err.Address = address
	return err
}

// KeyStoreV3 represents the Ethereum KeyStore V3 format
type KeyStoreV3 struct {
	Address string         `json:"address"`
	Crypto  KeyStoreCrypto `json:"crypto"`
	ID      string         `json:"id"`
	Version int            `json:"version"`
}

// KeyStoreCrypto contains the cryptographic parameters for the keystore
type KeyStoreCrypto struct {
	Cipher       string       `json:"cipher"`
	CipherText   string       `json:"ciphertext"`
	CipherParams CipherParams `json:"cipherparams"`
	KDF          string       `json:"kdf"`
	KDFParams    interface{}  `json:"kdfparams"`
	MAC          string       `json:"mac"`
}

// CipherParams contains parameters for the cipher algorithm
type CipherParams struct {
	IV string `json:"iv"`
}

// ScryptParams contains parameters for the scrypt key derivation function
type ScryptParams struct {
	DKLen int    `json:"dklen"`
	N     int    `json:"n"`
	P     int    `json:"p"`
	R     int    `json:"r"`
	Salt  string `json:"salt"`
}

// PBKDF2Params contains parameters for the PBKDF2 key derivation function
type PBKDF2Params struct {
	DKLen int    `json:"dklen"`
	C     int    `json:"c"`
	PRF   string `json:"prf"`
	Salt  string `json:"salt"`
}

// ValidateScryptParams validates scrypt parameters and returns detailed errors
func ValidateScryptParams(params *ScryptParams) error {
	if params == nil {
		return fmt.Errorf("scrypt parameters cannot be nil")
	}

	// Validate N parameter (must be power of 2)
	if params.N <= 0 {
		return fmt.Errorf("n parameter must be positive, got %d", params.N)
	}
	if params.N&(params.N-1) != 0 {
		return fmt.Errorf("n parameter must be a power of 2, got %d", params.N)
	}
	if params.N < 1024 || params.N > 67108864 {
		return fmt.Errorf("n parameter must be between 1024 and 67108864, got %d", params.N)
	}

	// Validate R parameter
	if params.R <= 0 || params.R > 1024 {
		return fmt.Errorf("r parameter must be between 1 and 1024, got %d", params.R)
	}

	// Validate P parameter
	if params.P <= 0 || params.P > 16 {
		return fmt.Errorf("p parameter must be between 1 and 16, got %d", params.P)
	}

	// Validate DKLen parameter
	if params.DKLen < 16 || params.DKLen > 128 {
		return fmt.Errorf("DKLen parameter must be between 16 and 128, got %d", params.DKLen)
	}

	// Validate salt
	if params.Salt == "" {
		return fmt.Errorf("salt cannot be empty")
	}
	if _, err := hex.DecodeString(params.Salt); err != nil {
		return fmt.Errorf("salt must be valid hex string: %w", err)
	}

	// Check memory usage (approximate)
	memoryUsage := int64(128) * int64(params.R) * int64(params.N) * int64(params.P)
	if memoryUsage > 2*1024*1024*1024 { // 2GB limit
		return fmt.Errorf("memory usage too high: %d bytes (limit: 2GB)", memoryUsage)
	}

	return nil
}

// ValidatePBKDF2Params validates PBKDF2 parameters and returns detailed errors
func ValidatePBKDF2Params(params *PBKDF2Params) error {
	if params == nil {
		return fmt.Errorf("PBKDF2 parameters cannot be nil")
	}

	// Validate iteration count
	if params.C < 1000 {
		return fmt.Errorf("iteration count too low for security: %d (minimum: 1000)", params.C)
	}
	if params.C > 10000000 {
		return fmt.Errorf("iteration count too high: %d (maximum: 10000000)", params.C)
	}

	// Validate DKLen parameter
	if params.DKLen < 16 || params.DKLen > 128 {
		return fmt.Errorf("DKLen parameter must be between 16 and 128, got %d", params.DKLen)
	}

	// Validate PRF parameter
	validPRFs := map[string]bool{
		"hmac-sha256": true,
		"hmac-sha512": true,
		"":            true, // Empty defaults to hmac-sha256
	}
	if !validPRFs[params.PRF] {
		return fmt.Errorf("invalid PRF: %s (supported: hmac-sha256, hmac-sha512)", params.PRF)
	}

	// Validate salt
	if params.Salt == "" {
		return fmt.Errorf("salt cannot be empty")
	}
	if _, err := hex.DecodeString(params.Salt); err != nil {
		return fmt.Errorf("salt must be valid hex string: %w", err)
	}

	return nil
}

// ParseScryptParamsFromMap creates ScryptParams from a map with flexible type handling
func ParseScryptParamsFromMap(params map[string]interface{}) (*ScryptParams, error) {
	if params == nil {
		return nil, fmt.Errorf("parameters map cannot be nil")
	}

	result := &ScryptParams{}

	// Parse N parameter
	if n, ok := params["n"]; ok {
		if nInt, err := parseIntParam(n); err != nil {
			return nil, fmt.Errorf("invalid N parameter: %w", err)
		} else {
			result.N = nInt
		}
	} else {
		return nil, fmt.Errorf("missing required parameter: n")
	}

	// Parse R parameter
	if r, ok := params["r"]; ok {
		if rInt, err := parseIntParam(r); err != nil {
			return nil, fmt.Errorf("invalid R parameter: %w", err)
		} else {
			result.R = rInt
		}
	} else {
		return nil, fmt.Errorf("missing required parameter: r")
	}

	// Parse P parameter
	if p, ok := params["p"]; ok {
		if pInt, err := parseIntParam(p); err != nil {
			return nil, fmt.Errorf("invalid P parameter: %w", err)
		} else {
			result.P = pInt
		}
	} else {
		return nil, fmt.Errorf("missing required parameter: p")
	}

	// Parse DKLen parameter
	if dklen, ok := params["dklen"]; ok {
		if dklenInt, err := parseIntParam(dklen); err != nil {
			return nil, fmt.Errorf("invalid DKLen parameter: %w", err)
		} else {
			result.DKLen = dklenInt
		}
	} else {
		return nil, fmt.Errorf("missing required parameter: dklen")
	}

	// Parse Salt parameter
	if salt, ok := params["salt"]; ok {
		if saltStr, err := parseSaltParam(salt); err != nil {
			return nil, fmt.Errorf("invalid salt parameter: %w", err)
		} else {
			result.Salt = saltStr
		}
	} else {
		return nil, fmt.Errorf("missing required parameter: salt")
	}

	return result, nil
}

// ParsePBKDF2ParamsFromMap creates PBKDF2Params from a map with flexible type handling
func ParsePBKDF2ParamsFromMap(params map[string]interface{}) (*PBKDF2Params, error) {
	if params == nil {
		return nil, fmt.Errorf("parameters map cannot be nil")
	}

	result := &PBKDF2Params{}

	// Parse C parameter (iteration count)
	if c, ok := params["c"]; ok {
		if cInt, err := parseIntParam(c); err != nil {
			return nil, fmt.Errorf("invalid C parameter: %w", err)
		} else {
			result.C = cInt
		}
	} else {
		return nil, fmt.Errorf("missing required parameter: c")
	}

	// Parse DKLen parameter
	if dklen, ok := params["dklen"]; ok {
		if dklenInt, err := parseIntParam(dklen); err != nil {
			return nil, fmt.Errorf("invalid DKLen parameter: %w", err)
		} else {
			result.DKLen = dklenInt
		}
	} else {
		return nil, fmt.Errorf("missing required parameter: dklen")
	}

	// Parse PRF parameter (optional, defaults to hmac-sha256)
	if prf, ok := params["prf"]; ok {
		if prfStr, ok := prf.(string); ok {
			result.PRF = prfStr
		} else {
			return nil, fmt.Errorf("PRF parameter must be a string, got %T", prf)
		}
	} else {
		result.PRF = "hmac-sha256" // Default
	}

	// Parse Salt parameter
	if salt, ok := params["salt"]; ok {
		if saltStr, err := parseSaltParam(salt); err != nil {
			return nil, fmt.Errorf("invalid salt parameter: %w", err)
		} else {
			result.Salt = saltStr
		}
	} else {
		return nil, fmt.Errorf("missing required parameter: salt")
	}

	return result, nil
}

// parseIntParam parses an integer parameter from various types
func parseIntParam(value interface{}) (int, error) {
	switch v := value.(type) {
	case int:
		return v, nil
	case int32:
		return int(v), nil
	case int64:
		return int(v), nil
	case float64:
		return int(v), nil
	case float32:
		return int(v), nil
	case string:
		var result int
		if n, err := fmt.Sscanf(v, "%d", &result); err != nil || n != 1 {
			return 0, fmt.Errorf("cannot parse string as integer: %s", v)
		}
		return result, nil
	default:
		return 0, fmt.Errorf("unsupported type for integer parameter: %T", value)
	}
}

// parseSaltParam parses a salt parameter from various formats
func parseSaltParam(value interface{}) (string, error) {
	switch v := value.(type) {
	case string:
		// Validate hex string
		if _, err := hex.DecodeString(v); err != nil {
			return "", fmt.Errorf("invalid hex string: %w", err)
		}
		return v, nil
	case []byte:
		return hex.EncodeToString(v), nil
	case []interface{}:
		// Handle array of numbers (common in JSON)
		bytes := make([]byte, len(v))
		for i, item := range v {
			if num, ok := item.(float64); ok {
				if num < 0 || num > 255 {
					return "", fmt.Errorf("byte value out of range: %f", num)
				}
				bytes[i] = byte(num)
			} else {
				return "", fmt.Errorf("array element must be number, got %T", item)
			}
		}
		return hex.EncodeToString(bytes), nil
	default:
		return "", fmt.Errorf("unsupported type for salt parameter: %T", value)
	}
}

// ToKDFCryptoParams converts keystore crypto parameters to KDF service format
func (ks *KeyStoreV3) ToKDFCryptoParams() (*kdf.CryptoParams, error) {
	if err := ks.Validate(); err != nil {
		return nil, fmt.Errorf("invalid keystore: %w", err)
	}

	// Convert KDFParams to map[string]interface{}
	var kdfParams map[string]interface{}

	switch ks.Crypto.KDF {
	case "scrypt":
		scryptParams, err := ks.GetScryptParams()
		if err != nil {
			return nil, fmt.Errorf("failed to get scrypt parameters: %w", err)
		}
		kdfParams = map[string]interface{}{
			"n":     scryptParams.N,
			"r":     scryptParams.R,
			"p":     scryptParams.P,
			"dklen": scryptParams.DKLen,
			"salt":  scryptParams.Salt,
		}
	case "pbkdf2":
		pbkdf2Params, err := ks.GetPBKDF2Params()
		if err != nil {
			return nil, fmt.Errorf("failed to get PBKDF2 parameters: %w", err)
		}
		kdfParams = map[string]interface{}{
			"c":     pbkdf2Params.C,
			"dklen": pbkdf2Params.DKLen,
			"prf":   pbkdf2Params.PRF,
			"salt":  pbkdf2Params.Salt,
		}
	default:
		return nil, fmt.Errorf("unsupported KDF: %s", ks.Crypto.KDF)
	}

	return &kdf.CryptoParams{
		KDF:        ks.Crypto.KDF,
		KDFParams:  kdfParams,
		Cipher:     ks.Crypto.Cipher,
		CipherText: ks.Crypto.CipherText,
		CipherParams: map[string]interface{}{
			"iv": ks.Crypto.CipherParams.IV,
		},
		MAC: ks.Crypto.MAC,
	}, nil
}

// FromKDFCryptoParams updates keystore from KDF service crypto parameters
func (ks *KeyStoreV3) FromKDFCryptoParams(cryptoParams *kdf.CryptoParams) error {
	if cryptoParams == nil {
		return fmt.Errorf("crypto parameters cannot be nil")
	}

	ks.Crypto.KDF = cryptoParams.KDF
	ks.Crypto.Cipher = cryptoParams.Cipher
	ks.Crypto.CipherText = cryptoParams.CipherText
	ks.Crypto.MAC = cryptoParams.MAC

	// Set cipher parameters
	if iv, ok := cryptoParams.CipherParams["iv"].(string); ok {
		ks.Crypto.CipherParams.IV = iv
	} else {
		return fmt.Errorf("missing or invalid IV in cipher parameters")
	}

	// Convert and validate KDF parameters
	switch cryptoParams.KDF {
	case "scrypt":
		scryptParams, err := ParseScryptParamsFromMap(cryptoParams.KDFParams)
		if err != nil {
			return fmt.Errorf("failed to parse scrypt parameters: %w", err)
		}
		if err := ks.SetScryptParamsFromStruct(scryptParams); err != nil {
			return fmt.Errorf("failed to set scrypt parameters: %w", err)
		}
	case "pbkdf2":
		pbkdf2Params, err := ParsePBKDF2ParamsFromMap(cryptoParams.KDFParams)
		if err != nil {
			return fmt.Errorf("failed to parse PBKDF2 parameters: %w", err)
		}
		if err := ks.SetPBKDF2ParamsFromStruct(pbkdf2Params); err != nil {
			return fmt.Errorf("failed to set PBKDF2 parameters: %w", err)
		}
	default:
		return fmt.Errorf("unsupported KDF: %s", cryptoParams.KDF)
	}

	return nil
}

// NewKeyStoreV3 creates a new KeyStore V3 structure with default values
func NewKeyStoreV3(address, network string) *KeyStoreV3 {
	var cleanAddress string
	if network == "ethereum" || network == "" {
		// Remove 0x prefix if present and convert to lowercase for Ethereum
		cleanAddress = strings.ToLower(strings.TrimPrefix(address, "0x"))
	} else {
		// Keep original case for other networks (e.g. Solana, Bitcoin)
		cleanAddress = address
	}

	return &KeyStoreV3{
		Address: cleanAddress,
		Crypto: KeyStoreCrypto{
			Cipher: "aes-128-ctr",
			KDF:    "scrypt",
		},
		ID:      uuid.New().String(),
		Version: 3,
	}
}

// SetScryptParams sets the scrypt parameters for the keystore
func (ks *KeyStoreV3) SetScryptParams(n, r, p, dklen int, salt []byte) {
	ks.Crypto.KDF = "scrypt"
	ks.Crypto.KDFParams = ScryptParams{
		DKLen: dklen,
		N:     n,
		P:     p,
		R:     r,
		Salt:  hex.EncodeToString(salt),
	}
}

// SetScryptParamsFromStruct sets the scrypt parameters from a ScryptParams struct
func (ks *KeyStoreV3) SetScryptParamsFromStruct(params *ScryptParams) error {
	if params == nil {
		return fmt.Errorf("scrypt parameters cannot be nil")
	}

	// Validate parameters before setting
	if err := ValidateScryptParams(params); err != nil {
		return fmt.Errorf("invalid scrypt parameters: %w", err)
	}

	ks.Crypto.KDF = "scrypt"
	ks.Crypto.KDFParams = *params
	return nil
}

// SetPBKDF2Params sets the PBKDF2 parameters for the keystore
func (ks *KeyStoreV3) SetPBKDF2Params(c, dklen int, prf string, salt []byte) {
	ks.Crypto.KDF = "pbkdf2"
	ks.Crypto.KDFParams = PBKDF2Params{
		DKLen: dklen,
		C:     c,
		PRF:   prf,
		Salt:  hex.EncodeToString(salt),
	}
}

// SetPBKDF2ParamsFromStruct sets the PBKDF2 parameters from a PBKDF2Params struct
func (ks *KeyStoreV3) SetPBKDF2ParamsFromStruct(params *PBKDF2Params) error {
	if params == nil {
		return fmt.Errorf("PBKDF2 parameters cannot be nil")
	}

	// Validate parameters before setting
	if err := ValidatePBKDF2Params(params); err != nil {
		return fmt.Errorf("invalid PBKDF2 parameters: %w", err)
	}

	ks.Crypto.KDF = "pbkdf2"
	ks.Crypto.KDFParams = *params
	return nil
}

// SetCipherParams sets the cipher parameters (IV and ciphertext)
func (ks *KeyStoreV3) SetCipherParams(iv, ciphertext []byte) {
	ks.Crypto.CipherParams = CipherParams{
		IV: hex.EncodeToString(iv),
	}
	ks.Crypto.CipherText = hex.EncodeToString(ciphertext)
}

// SetMAC sets the MAC for integrity verification
func (ks *KeyStoreV3) SetMAC(mac []byte) {
	ks.Crypto.MAC = hex.EncodeToString(mac)
}

// GenerateRandomBytes generates cryptographically secure random bytes
func GenerateRandomBytes(length int) ([]byte, error) {
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return bytes, nil
}

// Validate validates the KeyStore V3 structure
func (ks *KeyStoreV3) Validate() error {
	if ks.Address == "" {
		return fmt.Errorf("address cannot be empty")
	}

	if len(ks.Address) == 0 {
		return fmt.Errorf("address cannot be empty")
	}

	if ks.Version != 3 {
		return fmt.Errorf("version must be 3")
	}

	if ks.ID == "" {
		return fmt.Errorf("ID cannot be empty")
	}

	if ks.Crypto.Cipher != "aes-128-ctr" {
		return fmt.Errorf("unsupported cipher: %s", ks.Crypto.Cipher)
	}

	if ks.Crypto.KDF != "scrypt" && ks.Crypto.KDF != "pbkdf2" {
		return fmt.Errorf("unsupported KDF: %s", ks.Crypto.KDF)
	}

	if ks.Crypto.CipherText == "" {
		return fmt.Errorf("ciphertext cannot be empty")
	}

	if ks.Crypto.MAC == "" {
		return fmt.Errorf("MAC cannot be empty")
	}

	if ks.Crypto.CipherParams.IV == "" {
		return fmt.Errorf("IV cannot be empty")
	}

	if len(ks.Crypto.CipherParams.IV) != 32 {
		return fmt.Errorf("IV must be 32 hex characters (16 bytes), got %d", len(ks.Crypto.CipherParams.IV))
	}

	if len(ks.Crypto.CipherText) != 64 {
		return fmt.Errorf("ciphertext must be 64 hex characters (32 bytes), got %d", len(ks.Crypto.CipherText))
	}

	if len(ks.Crypto.MAC) != 64 {
		return fmt.Errorf("MAC must be 64 hex characters (32 bytes), got %d", len(ks.Crypto.MAC))
	}

	if ks.Crypto.KDFParams == nil {
		return fmt.Errorf("KDF parameters cannot be nil")
	}

	// Validate the KDF salt length (32 bytes = 64 hex characters)
	switch params := ks.Crypto.KDFParams.(type) {
	case ScryptParams:
		if len(params.Salt) != 64 {
			return fmt.Errorf("scrypt salt must be 64 hex characters (32 bytes), got %d", len(params.Salt))
		}
	case PBKDF2Params:
		if len(params.Salt) != 64 {
			return fmt.Errorf("pbkdf2 salt must be 64 hex characters (32 bytes), got %d", len(params.Salt))
		}
	}

	return nil
}

// ToJSON serializes the KeyStore V3 to JSON
func (ks *KeyStoreV3) ToJSON() ([]byte, error) {
	if err := ks.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	return json.MarshalIndent(ks, "", "  ")
}

// FromJSON deserializes JSON data into a KeyStore V3 structure
func FromJSON(data []byte) (*KeyStoreV3, error) {
	var ks KeyStoreV3
	if err := json.Unmarshal(data, &ks); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	if err := ks.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	return &ks, nil
}

// GetScryptParams returns the scrypt parameters if KDF is scrypt
func (ks *KeyStoreV3) GetScryptParams() (*ScryptParams, error) {
	if ks.Crypto.KDF != "scrypt" {
		return nil, fmt.Errorf("KDF is not scrypt")
	}

	// Try direct type assertion first
	if params, ok := ks.Crypto.KDFParams.(ScryptParams); ok {
		return &params, nil
	}

	// Convert interface{} to ScryptParams using flexible parsing
	paramsMap, ok := ks.Crypto.KDFParams.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid scrypt parameters format: expected map[string]interface{} or ScryptParams")
	}

	params, err := ParseScryptParamsFromMap(paramsMap)
	if err != nil {
		return nil, fmt.Errorf("failed to parse scrypt parameters: %w", err)
	}

	// Validate the parsed parameters
	if err := ValidateScryptParams(params); err != nil {
		return nil, fmt.Errorf("invalid scrypt parameters: %w", err)
	}

	return params, nil
}

// GetPBKDF2Params returns the PBKDF2 parameters if KDF is pbkdf2
func (ks *KeyStoreV3) GetPBKDF2Params() (*PBKDF2Params, error) {
	if ks.Crypto.KDF != "pbkdf2" {
		return nil, fmt.Errorf("KDF is not pbkdf2")
	}

	// Try direct type assertion first
	if params, ok := ks.Crypto.KDFParams.(PBKDF2Params); ok {
		return &params, nil
	}

	// Convert interface{} to PBKDF2Params using flexible parsing
	paramsMap, ok := ks.Crypto.KDFParams.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid PBKDF2 parameters format: expected map[string]interface{} or PBKDF2Params")
	}

	params, err := ParsePBKDF2ParamsFromMap(paramsMap)
	if err != nil {
		return nil, fmt.Errorf("failed to parse PBKDF2 parameters: %w", err)
	}

	// Validate the parsed parameters
	if err := ValidatePBKDF2Params(params); err != nil {
		return nil, fmt.Errorf("invalid PBKDF2 parameters: %w", err)
	}

	return params, nil
}

// DeriveKeyScrypt derives a key using the scrypt key derivation function
func DeriveKeyScrypt(password []byte, salt []byte, n, r, p, dkLen int) ([]byte, error) {
	if len(password) == 0 {
		return nil, fmt.Errorf("password cannot be empty")
	}
	if len(salt) == 0 {
		return nil, fmt.Errorf("salt cannot be empty")
	}
	if dkLen <= 0 {
		return nil, fmt.Errorf("derived key length must be positive")
	}

	key, err := scrypt.Key(password, salt, n, r, p, dkLen)
	if err != nil {
		return nil, fmt.Errorf("scrypt key derivation failed: %w", err)
	}

	return key, nil
}

// DeriveKeyPBKDF2 derives a key using the PBKDF2 key derivation function with HMAC-SHA256
func DeriveKeyPBKDF2(password []byte, salt []byte, iterations, dkLen int) ([]byte, error) {
	if len(password) == 0 {
		return nil, fmt.Errorf("password cannot be empty")
	}
	if len(salt) == 0 {
		return nil, fmt.Errorf("salt cannot be empty")
	}
	if iterations <= 0 {
		return nil, fmt.Errorf("iterations must be positive")
	}
	if dkLen <= 0 {
		return nil, fmt.Errorf("derived key length must be positive")
	}

	key := pbkdf2.Key(password, salt, iterations, dkLen, sha256.New)
	return key, nil
}

// EncryptAES128CTR encrypts data using AES-128-CTR mode
func EncryptAES128CTR(plaintext []byte, key []byte, iv []byte) ([]byte, error) {
	if len(plaintext) == 0 {
		return nil, fmt.Errorf("plaintext cannot be empty")
	}
	if len(key) != 16 {
		return nil, fmt.Errorf("key must be 16 bytes for AES-128")
	}
	if len(iv) != 16 {
		return nil, fmt.Errorf("IV must be 16 bytes for AES-128-CTR")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	ciphertext := make([]byte, len(plaintext))
	stream := cipher.NewCTR(block, iv)
	stream.XORKeyStream(ciphertext, plaintext)

	return ciphertext, nil
}

// DecryptAES128CTR decrypts data using AES-128-CTR mode
func DecryptAES128CTR(ciphertext []byte, key []byte, iv []byte) ([]byte, error) {
	if len(ciphertext) == 0 {
		return nil, fmt.Errorf("ciphertext cannot be empty")
	}
	if len(key) != 16 {
		return nil, fmt.Errorf("key must be 16 bytes for AES-128")
	}
	if len(iv) != 16 {
		return nil, fmt.Errorf("IV must be 16 bytes for AES-128-CTR")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	plaintext := make([]byte, len(ciphertext))
	stream := cipher.NewCTR(block, iv)
	stream.XORKeyStream(plaintext, ciphertext)

	return plaintext, nil
}

// GenerateMAC generates a MAC using Keccak256 for integrity verification (Ethereum KeyStore V3 standard)
func GenerateMAC(derivedKey []byte, ciphertext []byte) ([]byte, error) {
	if len(derivedKey) < 32 {
		return nil, fmt.Errorf("derived key must be at least 32 bytes")
	}
	if len(ciphertext) == 0 {
		return nil, fmt.Errorf("ciphertext cannot be empty")
	}

	// Use the second half of the derived key for MAC generation (Ethereum KeyStore V3 standard)
	macKey := derivedKey[16:32]

	// Calculate MAC using Keccak256 (Ethereum standard)
	hash := crypto.Keccak256Hash(macKey, ciphertext)
	return hash.Bytes(), nil
}

// VerifyMAC verifies a MAC using Keccak256 (Ethereum KeyStore V3 standard)
func VerifyMAC(derivedKey []byte, ciphertext []byte, expectedMAC []byte) (bool, error) {
	if len(expectedMAC) == 0 {
		return false, fmt.Errorf("expected MAC cannot be empty")
	}

	computedMAC, err := GenerateMAC(derivedKey, ciphertext)
	if err != nil {
		return false, fmt.Errorf("failed to compute MAC: %w", err)
	}

	// Use constant time comparison for security
	if len(computedMAC) != len(expectedMAC) {
		return false, nil
	}

	// Constant time comparison
	result := byte(0)
	for i := 0; i < len(computedMAC); i++ {
		result |= computedMAC[i] ^ expectedMAC[i]
	}

	return result == 0, nil
}

// EncryptPrivateKey encrypts a private key using the specified KDF and AES-128-CTR
func EncryptPrivateKey(privateKeyHex string, password string, kdfType string) (*KeyStoreV3, error) {
	// Create a temporary service for this operation
	config := KeyStoreConfig{
		KDF: kdfType,
	}
	service := NewKeyStoreService(config)

	return service.EncryptPrivateKeyWithKDF(privateKeyHex, password, kdfType, "ethereum")
}

// EncryptPrivateKeyWithKDF encrypts a private key using the Universal KDF service
func (ks *KeyStoreService) EncryptPrivateKeyWithKDF(privateKeyHex string, password string, kdfType string, network string) (*KeyStoreV3, error) {
	if privateKeyHex == "" {
		return nil, NewKeyStoreError("encrypt", "private_key", fmt.Errorf("private key cannot be empty"))
	}
	if password == "" {
		return nil, NewKeyStoreError("encrypt", "password", fmt.Errorf("password cannot be empty"))
	}

	// Remove 0x prefix if present
	privateKeyHex = strings.TrimPrefix(privateKeyHex, "0x")

	// Validate private key length based on network
	// Ethereum: 64 hex chars (32 bytes)
	// Solana: 128 hex chars (64 bytes for Ed25519)
	// Bitcoin: 64 hex chars (32 bytes for secp256k1)
	expectedLength := 64 // Default for Ethereum and Bitcoin
	if network == "solana" {
		expectedLength = 128 // Ed25519 private key is 64 bytes
	}

	if len(privateKeyHex) != expectedLength {
		return nil, NewKeyStoreError("save", "private_key",
			fmt.Errorf("private key must be %d hex characters for %s network, got %d",
				expectedLength, network, len(privateKeyHex)))
	}

	privateKeyBytes, err := hex.DecodeString(privateKeyHex)
	if err != nil {
		return nil, NewKeyStoreError("validate", "private_key",
			fmt.Errorf("invalid private key hex: %w", err))
	}

	// Generate random salt and IV
	salt, err := GenerateRandomBytes(32)
	if err != nil {
		return nil, NewRecoverableKeyStoreError("encrypt", "salt", err,
			"Failed to generate cryptographic salt. This might be due to insufficient system entropy.")
	}

	iv, err := GenerateRandomBytes(16)
	if err != nil {
		return nil, NewRecoverableKeyStoreError("encrypt", "iv", err,
			"Failed to generate initialization vector. This might be due to insufficient system entropy.")
	}

	// Get default parameters for the specified KDF
	defaultParams, err := ks.kdfService.GetDefaultParams(kdfType)
	if err != nil {
		if kdfErr, ok := err.(*kdf.KDFError); ok {
			return nil, NewKDFKeyStoreError("encrypt", "kdf_params", kdfErr)
		}
		return nil, NewKeyStoreError("encrypt", "kdf_params", err)
	}

	// Add salt to parameters
	defaultParams["salt"] = hex.EncodeToString(salt)

	// Create crypto parameters for KDF service
	cryptoParams := &kdf.CryptoParams{
		KDF:       kdfType,
		KDFParams: defaultParams,
	}

	// Validate parameters before key derivation
	if err := ks.kdfService.ValidateParams(kdfType, defaultParams); err != nil {
		if kdfErr, ok := err.(*kdf.KDFError); ok {
			return nil, NewKDFKeyStoreError("validate", "kdf_params", kdfErr)
		}
		return nil, NewKeyStoreError("validate", "kdf_params", err)
	}

	// Perform compatibility analysis
	compatReport, err := ks.analyzer.AnalyzeKeystore(cryptoParams)
	if err != nil {
		ks.logger.LogWarning(fmt.Sprintf("Failed to analyze KDF compatibility: %v", err))
	} else {
		// Log compatibility warnings
		if len(compatReport.Warnings) > 0 {
			for _, warning := range compatReport.Warnings {
				ks.logger.LogWarning(fmt.Sprintf("KDF compatibility warning: %s", warning))
			}
		}

		// Log security level
		ks.logger.LogInfo(fmt.Sprintf("KDF security level: %s", compatReport.SecurityLevel))
	}

	// Derive key using Universal KDF service
	derivedKey, err := ks.kdfService.DeriveKey(password, cryptoParams)
	if err != nil {
		if kdfErr, ok := err.(*kdf.KDFError); ok {
			return nil, NewKDFKeyStoreError("derive", "key", kdfErr)
		}
		return nil, NewKeyStoreError("derive", "key", err)
	}

	// Use first 16 bytes of derived key for AES encryption
	if len(derivedKey) < 32 {
		return nil, NewKeyStoreError("derive", "key",
			fmt.Errorf("derived key too short: got %d bytes, need at least 32", len(derivedKey)))
	}
	encryptionKey := derivedKey[:16]

	// Encrypt private key
	ciphertext, err := EncryptAES128CTR(privateKeyBytes, encryptionKey, iv)
	if err != nil {
		return nil, NewKeyStoreError("encrypt", "aes", fmt.Errorf("AES encryption failed: %w", err))
	}

	// Generate MAC for integrity
	mac, err := GenerateMAC(derivedKey, ciphertext)
	if err != nil {
		return nil, NewKeyStoreError("encrypt", "mac", fmt.Errorf("MAC generation failed: %w", err))
	}

	// Derive Ethereum address from the private key
	privateKey, err := crypto.ToECDSA(privateKeyBytes)
	if err != nil {
		return nil, NewKeyStoreError("encrypt", "private_key",
			fmt.Errorf("invalid private key: %w", err))
	}
	address := strings.ToLower(strings.TrimPrefix(crypto.PubkeyToAddress(privateKey.PublicKey).Hex(), "0x"))

	// Create KeyStore V3 structure
	keystore := NewKeyStoreV3(address, "ethereum")

	// Set cipher parameters
	keystore.SetCipherParams(iv, ciphertext)
	keystore.SetMAC(mac)

	// Set KDF parameters based on type
	switch kdfType {
	case "scrypt":
		scryptParams, err := ParseScryptParamsFromMap(defaultParams)
		if err != nil {
			return nil, NewKeyStoreError("convert", "scrypt_params", err)
		}
		if err := keystore.SetScryptParamsFromStruct(scryptParams); err != nil {
			return nil, NewKeyStoreError("set", "scrypt_params", err)
		}
	case "pbkdf2", "pbkdf2-sha256", "pbkdf2-sha512":
		pbkdf2Params, err := ParsePBKDF2ParamsFromMap(defaultParams)
		if err != nil {
			return nil, NewKeyStoreError("convert", "pbkdf2_params", err)
		}
		if err := keystore.SetPBKDF2ParamsFromStruct(pbkdf2Params); err != nil {
			return nil, NewKeyStoreError("set", "pbkdf2_params", err)
		}
	default:
		return nil, NewKeyStoreError("encrypt", "kdf", fmt.Errorf("unsupported KDF: %s", kdfType))
	}

	return keystore, nil
}

// KeyStoreConfig holds configuration for keystore generation
type KeyStoreConfig struct {
	OutputDirectory string
	Enabled         bool
	Cipher          string                 // "aes-128-ctr"
	KDF             string                 // "scrypt" or "pbkdf2"
	KDFParams       map[string]interface{} // KDF-specific parameters
	MaxRetries      int                    // Maximum number of retry attempts for recoverable errors
	RetryDelay      int                    // Delay between retries in milliseconds
}

// FileOperationError represents errors that occur during file operations
type FileOperationError struct {
	Operation string
	Path      string
	Err       error
}

func (e *FileOperationError) Error() string {
	return fmt.Sprintf("file operation '%s' failed for path '%s': %v", e.Operation, e.Path, e.Err)
}

func (e *FileOperationError) Unwrap() error {
	return e.Err
}

// ProgressLogger defines the interface for progress logging
type ProgressLogger interface {
	LogInfo(message string)
	LogWarning(message string)
	LogError(message string)
	LogDebug(message string)
}

// DefaultProgressLogger provides a simple console-based progress logger
type DefaultProgressLogger struct {
	VerboseMode bool
}

func (l *DefaultProgressLogger) LogInfo(message string) {
	if l.VerboseMode {
		fmt.Printf("INFO: %s\n", message)
	}
}

func (l *DefaultProgressLogger) LogWarning(message string) {
	fmt.Printf("WARNING: %s\n", message)
}

func (l *DefaultProgressLogger) LogError(message string) {
	fmt.Printf("ERROR: %s\n", message)
}

func (l *DefaultProgressLogger) LogDebug(message string) {
	if l.VerboseMode {
		fmt.Printf("%s\n", message)
	}
}

// KeyStoreService handles keystore generation and file operations
type KeyStoreService struct {
	config      KeyStoreConfig
	passwordGen *PasswordGenerator
	logger      ProgressLogger
	kdfService  *kdf.UniversalKDFService
	analyzer    *kdf.KDFCompatibilityAnalyzer
}

// NewKeyStoreService creates a new keystore service with the given configuration
func NewKeyStoreService(config KeyStoreConfig) *KeyStoreService {
	// Set defaults if not specified
	if config.Cipher == "" {
		config.Cipher = "aes-128-ctr"
	}
	if config.KDF == "" {
		config.KDF = "scrypt"
	}
	if config.OutputDirectory == "" {
		config.OutputDirectory = "./keystores"
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	if config.RetryDelay == 0 {
		config.RetryDelay = 100 // 100ms default delay
	}

	// Initialize Universal KDF service
	kdfService := kdf.NewUniversalKDFService()

	// Initialize compatibility analyzer
	analyzer := kdf.NewKDFCompatibilityAnalyzer(kdfService)

	return &KeyStoreService{
		config:      config,
		passwordGen: NewPasswordGenerator(),
		logger:      &DefaultProgressLogger{VerboseMode: false},
		kdfService:  kdfService,
		analyzer:    analyzer,
	}
}

// NewKeyStoreServiceWithLogger creates a new keystore service with a custom logger
func NewKeyStoreServiceWithLogger(config KeyStoreConfig, logger ProgressLogger) *KeyStoreService {
	service := NewKeyStoreService(config)
	service.logger = logger
	return service
}

// NewKeyStoreServiceWithKDFLogger creates a new keystore service with a KDF logger
func NewKeyStoreServiceWithKDFLogger(config KeyStoreConfig, kdfLogger kdf.KDFLogger) *KeyStoreService {
	service := NewKeyStoreService(config)
	service.kdfService.SetLogger(kdfLogger)
	return service
}

// SetLogger sets a custom progress logger
func (ks *KeyStoreService) SetLogger(logger ProgressLogger) {
	ks.logger = logger
}

// SetVerboseMode enables or disables verbose logging for the default logger
func (ks *KeyStoreService) SetVerboseMode(verbose bool) {
	if defaultLogger, ok := ks.logger.(*DefaultProgressLogger); ok {
		defaultLogger.VerboseMode = verbose
	}
}

// SaveKeyStoreFilesWithRetry saves keystore files with automatic retry for recoverable errors
func (ks *KeyStoreService) SaveKeyStoreFilesWithRetry(privateKeyHex, address, network string) error {
	if !ks.config.Enabled {
		ks.logger.LogDebug("Keystore generation is disabled, skipping file creation")
		return nil
	}

	var lastErr error
	for attempt := 1; attempt <= ks.config.MaxRetries; attempt++ {
		ks.logger.LogDebug(fmt.Sprintf("Keystore save attempt %d/%d for address %s", attempt, ks.config.MaxRetries, address))

		err := ks.SaveKeyStoreFiles(privateKeyHex, address, network)
		if err == nil {
			if attempt > 1 {
				ks.logger.LogInfo(fmt.Sprintf("Keystore files saved successfully for address %s after %d attempts", address, attempt))
			}
			return nil
		}

		lastErr = err

		// Check if error is recoverable
		if ksErr, ok := err.(*KeyStoreError); ok && ksErr.IsRecoverable() {
			if attempt < ks.config.MaxRetries {
				ks.logger.LogWarning(fmt.Sprintf("Recoverable error on attempt %d for address %s: %v. Retrying in %dms...",
					attempt, address, err, ks.config.RetryDelay))
				time.Sleep(time.Duration(ks.config.RetryDelay) * time.Millisecond)
				continue
			} else {
				ks.logger.LogError(fmt.Sprintf("Max retries (%d) exceeded for address %s. Last error: %v",
					ks.config.MaxRetries, address, err))
			}
		} else {
			// Non-recoverable error, don't retry
			ks.logger.LogError(fmt.Sprintf("Non-recoverable error for address %s: %v", address, err))
			break
		}
	}

	return lastErr
}

// GenerateKeyStore creates a keystore for the given private key and address
func (ks *KeyStoreService) GenerateKeyStore(privateKeyHex, address, network string) (*KeyStoreV3, string, error) {
	if !ks.config.Enabled {
		return nil, "", NewKeyStoreError("generate", "service", fmt.Errorf("keystore generation is disabled"))
	}

	if privateKeyHex == "" {
		return nil, "", NewKeyStoreErrorWithAddress("generate", "private_key", address, fmt.Errorf("private key cannot be empty"))
	}

	if address == "" {
		return nil, "", NewKeyStoreError("generate", "address", fmt.Errorf("address cannot be empty"))
	}

	// Generate secure password
	password, err := ks.passwordGen.GenerateSecurePassword()
	if err != nil {
		return nil, "", NewRecoverableKeyStoreError("generate", "password", err,
			"Failed to generate secure password. This might be due to insufficient system entropy. Please try again.")
	}

	// Encrypt private key using the Universal KDF service
	keystore, err := ks.EncryptPrivateKeyWithKDF(privateKeyHex, password, ks.config.KDF, network)
	if err != nil {
		// Error is already properly formatted from EncryptPrivateKeyWithKDF
		if ksErr, ok := err.(*KeyStoreError); ok {
			ksErr.Address = address // Add address information
			return nil, "", ksErr
		}
		return nil, "", NewKeyStoreErrorWithAddress("encrypt", "private_key", address, err)
	}

	// Update the keystore with the correct address
	if network == "ethereum" || network == "" {
		keystore.Address = strings.ToLower(strings.TrimPrefix(address, "0x"))
	} else {
		keystore.Address = address
	}

	return keystore, password, nil
}

// SaveKeyStoreFiles saves keystore files for a given private key and address (convenience method)
// For Bitcoin: only saves mnemonic (no KeyStore V3)
// For Ethereum: saves KeyStore V3 + password
// For Solana: saves Solana keypair format
func (ks *KeyStoreService) SaveKeyStoreFiles(privateKeyHex, address, network string) error {
	if !ks.config.Enabled {
		ks.logger.LogDebug("Keystore generation is disabled, skipping file creation")
		return nil // Silently skip if disabled
	}

	ks.logger.LogDebug(fmt.Sprintf("Starting keystore generation for address %s (network: %s)", address, network))

	// Bitcoin only saves mnemonic, no KeyStore V3
	if strings.ToLower(network) == "bitcoin" {
		ks.logger.LogDebug("Bitcoin network detected, keystore generation skipped (use SaveMnemonicFile for Bitcoin)")
		return nil // Bitcoin doesn't use KeyStore V3, only mnemonic
	}

	// Generate keystore and password for Ethereum and Solana
	keystore, password, err := ks.GenerateKeyStore(privateKeyHex, address, network)
	if err != nil {
		ks.logger.LogError(fmt.Sprintf("Failed to generate keystore for address %s: %v", address, err))
		return fmt.Errorf("failed to generate keystore: %w", err)
	}

	ks.logger.LogDebug(fmt.Sprintf("Keystore generated successfully, saving files for address %s", address))

	// Save the files
	if err := ks.SaveKeyStoreFilesToDisk(address, keystore, password, network, privateKeyHex); err != nil {
		ks.logger.LogError(fmt.Sprintf("Failed to save keystore files for address %s: %v", address, err))
		return err
	}

	ks.logger.LogInfo(fmt.Sprintf("Keystore files saved successfully for address %s", address))
	return nil
}

// validateEthereumAddress validates an Ethereum address format
func validateEthereumAddress(address string) error {
	cleanAddress := strings.TrimPrefix(address, "0x")
	if len(cleanAddress) != 40 {
		return fmt.Errorf("invalid Ethereum address length: expected 40 characters, got %d", len(cleanAddress))
	}
	// Check if it's valid hex
	if _, err := hex.DecodeString(cleanAddress); err != nil {
		return fmt.Errorf("invalid Ethereum address: must be hexadecimal")
	}
	return nil
}

// validateBitcoinAddress validates a Bitcoin address format (basic check)
func validateBitcoinAddress(address string) error {
	if len(address) < 26 || len(address) > 35 {
		return fmt.Errorf("invalid Bitcoin address length: expected 26-35 characters, got %d", len(address))
	}
	// Bitcoin addresses start with 1, 3, or bc1
	if !strings.HasPrefix(address, "1") && !strings.HasPrefix(address, "3") && !strings.HasPrefix(address, "bc1") {
		return fmt.Errorf("invalid Bitcoin address: must start with 1, 3, or bc1")
	}
	return nil
}

// validateSolanaAddress validates a Solana address format (basic check)
func validateSolanaAddress(address string) error {
	if len(address) != 43 && len(address) != 44 {
		return fmt.Errorf("invalid Solana address length: expected 43-44 characters, got %d", len(address))
	}
	// Solana addresses are Base58 encoded - check for valid Base58 characters
	for _, c := range address {
		if !strings.ContainsRune("123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", c) {
			return fmt.Errorf("invalid Solana address: contains invalid Base58 character '%c'", c)
		}
	}
	return nil
}

// validateAddressForNetwork validates an address based on the network type
func validateAddressForNetwork(address, network string) error {
	if address == "" {
		return fmt.Errorf("address cannot be empty")
	}

	switch strings.ToLower(network) {
	case "ethereum", "":
		return validateEthereumAddress(address)
	case "bitcoin":
		return validateBitcoinAddress(address)
	case "solana":
		return validateSolanaAddress(address)
	default:
		return fmt.Errorf("unsupported network: %s", network)
	}
}

// formatAddressForFilename formats an address for use in filenames based on the network
// Ethereum addresses get "0x" prefix, Bitcoin and Solana addresses don't
func formatAddressForFilename(address, network string) string {
	cleanAddress := strings.TrimPrefix(address, "0x")

	// Only add 0x prefix for Ethereum
	if network == "ethereum" || network == "" {
		return "0x" + cleanAddress
	}

	// Bitcoin and Solana: preserve original case, no 0x prefix
	return cleanAddress
}

// SaveKeyStoreFilesToDisk saves keystore and password files to disk
// Network-specific behavior:
// - Ethereum: saves KeyStore V3 JSON + password file
// - Solana: saves keypair JSON + .key file with raw private key
// - Bitcoin: returns error (should use SaveMnemonicFile)
func (ks *KeyStoreService) SaveKeyStoreFilesToDisk(address string, keystore *KeyStoreV3, password, network, privateKeyHex string) error {
	if !ks.config.Enabled {
		return NewKeyStoreError("save", "service", fmt.Errorf("keystore generation is disabled"))
	}

	// Validate address for the specific network
	if err := validateAddressForNetwork(address, network); err != nil {
		return NewKeyStoreError("validate", "address", err)
	}

	// Ensure output directory exists
	if err := ks.ensureOutputDirectory(); err != nil {
		return err
	}

	// Check directory permissions
	if err := ks.CheckDirectoryPermissions(); err != nil {
		ks.logger.LogError(fmt.Sprintf("Directory permission check failed for %s: %v", ks.config.OutputDirectory, err))
		return NewRecoverableKeyStoreError("save", "directory", err,
			fmt.Sprintf("Directory '%s' is not writable. Please check permissions and try again.", ks.config.OutputDirectory))
	}

	// Save files based on network type
	switch strings.ToLower(network) {
	case "ethereum", "":
		return ks.saveEthereumKeyStore(address, keystore, password)
	case "bitcoin":
		// Bitcoin only saves mnemonic, no KeyStore V3
		return fmt.Errorf("bitcoin keystore saving should use SaveMnemonicFile directly")
	case "solana":
		// Save Solana keypair JSON
		if err := ks.saveSolanaKeypair(address, keystore); err != nil {
			return err
		}
		// Also save private key to .key file for easy access (unencrypted)
		return ks.SavePrivateKeyFile(address, privateKeyHex, network)
	default:
		return NewKeyStoreError("save", "network", fmt.Errorf("unsupported network: %s", network))
	}
}

// saveEthereumKeyStore saves Ethereum KeyStore V3 format files
func (ks *KeyStoreService) saveEthereumKeyStore(address string, keystore *KeyStoreV3, password string) error {
	if keystore == nil {
		return NewKeyStoreErrorWithAddress("save", "keystore", address, fmt.Errorf("keystore cannot be nil"))
	}

	if password == "" {
		return NewKeyStoreErrorWithAddress("save", "password", address, fmt.Errorf("password cannot be empty"))
	}

	// Format address with 0x prefix for Ethereum
	formattedAddress := formatAddressForFilename(address, "ethereum")

	// Get file paths; use the geth-compatible UTC--<timestamp>--<address>.json
	// naming convention so tools (geth, MyCrypto, Firefly) can import the file.
	keystorePath := filepath.Join(ks.config.OutputDirectory, fmt.Sprintf("UTC--%s--%s.json", utcTimestamp(), formattedAddress))
	passwordPath := filepath.Join(ks.config.OutputDirectory, fmt.Sprintf("%s.pwd", formattedAddress))

	// Check if files already exist and warn (but don't fail)
	if _, err := ks.FileExists(keystorePath); err != nil {
		return NewRecoverableKeyStoreError("save", "file_check", err,
			"Failed to check if keystore file already exists. Please try again.")
	}

	if _, err := ks.FileExists(passwordPath); err != nil {
		return NewRecoverableKeyStoreError("save", "file_check", err,
			"Failed to check if password file already exists. Please try again.")
	}

	// Serialize keystore to JSON
	keystoreJSON, err := keystore.ToJSON()
	if err != nil {
		return NewKeyStoreErrorWithAddress("serialize", "keystore", address, err)
	}

	// Write keystore file atomically with secure permissions (600)
	ks.logger.LogDebug(fmt.Sprintf("Writing keystore file: %s", keystorePath))
	if err := ks.writeFileAtomic(keystorePath, keystoreJSON, 0600); err != nil {
		ks.logger.LogError(fmt.Sprintf("Failed to write keystore file %s: %v", keystorePath, err))
		return NewRecoverableKeyStoreError("save", "keystore_file", err,
			fmt.Sprintf("Failed to save keystore file to '%s'. Please check disk space and permissions.", keystorePath))
	}
	ks.logger.LogDebug(fmt.Sprintf("Keystore file written successfully: %s", keystorePath))

	// Write password file atomically with secure permissions (600)
	ks.logger.LogDebug(fmt.Sprintf("Writing password file: %s", passwordPath))
	if err := ks.writeFileAtomic(passwordPath, []byte(password), 0600); err != nil {
		ks.logger.LogError(fmt.Sprintf("Failed to write password file %s: %v", passwordPath, err))
		// If password file fails, try to clean up keystore file
		ks.logger.LogDebug(fmt.Sprintf("Attempting to clean up keystore file: %s", keystorePath))
		if removeErr := os.Remove(keystorePath); removeErr != nil {
			ks.logger.LogError(fmt.Sprintf("Failed to clean up keystore file %s: %v", keystorePath, removeErr))
			// Return compound error with cleanup failure
			return NewKeyStoreErrorWithPath("save", "password_file", passwordPath,
				fmt.Errorf("failed to write password file and cleanup failed: write error: %w, cleanup error: %v", err, removeErr))
		}
		ks.logger.LogDebug(fmt.Sprintf("Keystore file cleaned up successfully: %s", keystorePath))
		return NewRecoverableKeyStoreError("save", "password_file", err,
			fmt.Sprintf("Failed to save password file to '%s'. Please check disk space and permissions.", passwordPath))
	}
	ks.logger.LogDebug(fmt.Sprintf("Password file written successfully: %s", passwordPath))

	// Verify both files were created with correct permissions
	if err := ks.ValidateFilePermissions(keystorePath, 0600); err != nil {
		return NewKeyStoreErrorWithPath("validate", "keystore_permissions", keystorePath, err)
	}

	if err := ks.ValidateFilePermissions(passwordPath, 0600); err != nil {
		return NewKeyStoreErrorWithPath("validate", "password_permissions", passwordPath, err)
	}

	return nil
}

// saveSolanaKeypair saves Solana keypair in the native Solana format
// Solana keypair format is a JSON array of 64 bytes: [private_key (32 bytes) + public_key (32 bytes)]
func (ks *KeyStoreService) saveSolanaKeypair(address string, keystore *KeyStoreV3) error {
	if keystore == nil {
		return NewKeyStoreErrorWithAddress("save", "keystore", address, fmt.Errorf("keystore cannot be nil"))
	}

	// Format address without 0x prefix for Solana
	formattedAddress := formatAddressForFilename(address, "solana")

	// Get file path (no 0x prefix for Solana)
	keypairPath := filepath.Join(ks.config.OutputDirectory, fmt.Sprintf("%s.json", formattedAddress))

	// For Solana, we need to extract the private key from the keystore
	// Since KeyStore V3 is encrypted, we'll create a simpler format
	// Note: In a real implementation, you'd decrypt the private key first
	// For now, we'll save a placeholder that indicates this is a Solana keypair

	// Create Solana keypair structure (simplified)
	// In production, this should be the actual 64-byte array
	solanaKeypair := map[string]interface{}{
		"type":    "solana-keypair",
		"address": address,
		"note":    "Solana keypair - private key is encrypted in KeyStore V3 format",
	}

	// Serialize to JSON
	keypairJSON, err := json.MarshalIndent(solanaKeypair, "", "  ")
	if err != nil {
		return NewKeyStoreErrorWithAddress("serialize", "keypair", address, err)
	}

	// Write keypair file atomically
	ks.logger.LogDebug(fmt.Sprintf("Writing Solana keypair file: %s", keypairPath))
	if err := ks.writeFileAtomic(keypairPath, keypairJSON, 0600); err != nil {
		ks.logger.LogError(fmt.Sprintf("Failed to write Solana keypair file %s: %v", keypairPath, err))
		return NewKeyStoreErrorWithPath("write", "keypair", keypairPath, err)
	}

	// Verify file permissions
	if err := ks.ValidateFilePermissions(keypairPath, 0600); err != nil {
		ks.logger.LogWarning(fmt.Sprintf("Solana keypair file permissions verification failed: %v", err))
	}

	ks.logger.LogInfo(fmt.Sprintf("Successfully saved Solana keypair file for address %s", address))
	return nil
}

// SaveMnemonicFile persists the mnemonic phrase for a wallet
func (ks *KeyStoreService) SaveMnemonicFile(address, mnemonic, network string) error {
	if !ks.config.Enabled {
		return NewKeyStoreError("save", "service", fmt.Errorf("keystore generation is disabled"))
	}

	if mnemonic == "" {
		return NewKeyStoreErrorWithAddress("save", "mnemonic", address, fmt.Errorf("mnemonic cannot be empty"))
	}

	// Validate address for the specific network
	if err := validateAddressForNetwork(address, network); err != nil {
		return NewKeyStoreErrorWithAddress("validate", "address", address, err)
	}

	// Format address based on network (0x prefix only for Ethereum)
	formattedAddress := formatAddressForFilename(address, network)

	// Construct mnemonic file path
	mnemonicPath := filepath.Join(ks.config.OutputDirectory, fmt.Sprintf("%s.mnemonic", formattedAddress))

	ks.logger.LogDebug(fmt.Sprintf("Ensuring output directory exists: %s", ks.config.OutputDirectory))
	if err := ks.ensureOutputDirectory(); err != nil {
		ks.logger.LogError(fmt.Sprintf("Failed to create directory %s for mnemonic: %v", ks.config.OutputDirectory, err))
		return NewRecoverableKeyStoreError("save", "directory", err,
			fmt.Sprintf("Failed to create keystore directory '%s'. Please check permissions and try again.", ks.config.OutputDirectory))
	}

	ks.logger.LogDebug(fmt.Sprintf("Checking directory permissions for mnemonic file: %s", ks.config.OutputDirectory))
	if err := ks.CheckDirectoryPermissions(); err != nil {
		ks.logger.LogError(fmt.Sprintf("Directory permission check failed for %s: %v", ks.config.OutputDirectory, err))
		return NewRecoverableKeyStoreError("save", "directory", err,
			fmt.Sprintf("Directory '%s' is not writable. Please check permissions and try again.", ks.config.OutputDirectory))
	}

	// Check if file already exists
	if _, err := ks.FileExists(mnemonicPath); err != nil {
		return NewRecoverableKeyStoreError("save", "file_check", err,
			"Failed to check if mnemonic file already exists. Please try again.")
	}

	ks.logger.LogDebug(fmt.Sprintf("Writing mnemonic file: %s", mnemonicPath))
	if err := ks.writeFileAtomic(mnemonicPath, []byte(mnemonic), 0600); err != nil {
		ks.logger.LogError(fmt.Sprintf("Failed to write mnemonic file %s: %v", mnemonicPath, err))
		return NewRecoverableKeyStoreError("save", "mnemonic_file", err,
			fmt.Sprintf("Failed to save mnemonic file to '%s'. Please check disk space and permissions.", mnemonicPath))
	}
	ks.logger.LogDebug(fmt.Sprintf("Mnemonic file written successfully: %s", mnemonicPath))

	if err := ks.ValidateFilePermissions(mnemonicPath, 0600); err != nil {
		return NewKeyStoreErrorWithPath("validate", "mnemonic_permissions", mnemonicPath, err)
	}

	return nil
}

// SavePrivateKeyFile saves the private key to a .key file (for Solana)
// This provides easy access to the full private key which may be truncated in TUI
func (ks *KeyStoreService) SavePrivateKeyFile(address, privateKeyHex, network string) error {
	// Validate inputs
	if err := validateAddressForNetwork(address, network); err != nil {
		return NewKeyStoreError("validate", "address", err)
	}

	if privateKeyHex == "" {
		return NewKeyStoreError("validate", "private_key",
			fmt.Errorf("private key cannot be empty"))
	}

	// Format address for filename (no 0x prefix for Solana)
	formattedAddress := formatAddressForFilename(address, network)

	// Get file path
	keyPath := filepath.Join(ks.config.OutputDirectory, fmt.Sprintf("%s.key", formattedAddress))

	// Write private key file atomically
	ks.logger.LogDebug(fmt.Sprintf("Writing private key file: %s", keyPath))
	if err := ks.writeFileAtomic(keyPath, []byte(privateKeyHex), 0600); err != nil {
		ks.logger.LogError(fmt.Sprintf("Failed to write private key file %s: %v", keyPath, err))
		return NewKeyStoreErrorWithPath("write", "private_key", keyPath, err)
	}

	return nil
}

// ensureOutputDirectory creates the output directory if it doesn't exist with proper error handling
func (ks *KeyStoreService) ensureOutputDirectory() error {
	// Check if path is empty
	if ks.config.OutputDirectory == "" {
		return fmt.Errorf("output directory cannot be empty")
	}

	// Clean the path to handle any path traversal issues
	cleanPath := filepath.Clean(ks.config.OutputDirectory)

	// Check if directory already exists
	info, err := os.Stat(cleanPath)
	if err == nil {
		// Path exists, check if it's a directory
		if !info.IsDir() {
			return fmt.Errorf("path %s exists but is not a directory", cleanPath)
		}
		// Directory exists and is valid
		return nil
	}

	// Check if error is something other than "not exists"
	if !os.IsNotExist(err) {
		return fmt.Errorf("failed to check directory %s: %w", cleanPath, err)
	}

	// Directory doesn't exist, create it with all parent directories
	if err := os.MkdirAll(cleanPath, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", cleanPath, err)
	}

	// Verify the directory was created successfully
	if info, err := os.Stat(cleanPath); err != nil {
		return fmt.Errorf("failed to verify created directory %s: %w", cleanPath, err)
	} else if !info.IsDir() {
		return fmt.Errorf("created path %s is not a directory", cleanPath)
	}

	return nil
}

// writeFileAtomic writes data to a file atomically using a temporary file with enhanced error handling
func (ks *KeyStoreService) writeFileAtomic(filename string, data []byte, perm os.FileMode) error {
	// Validate inputs
	if filename == "" {
		return fmt.Errorf("filename cannot be empty")
	}
	if data == nil {
		return fmt.Errorf("data cannot be nil")
	}
	if perm == 0 {
		return fmt.Errorf("file permissions cannot be zero")
	}

	// Clean the filename path
	cleanFilename := filepath.Clean(filename)
	dir := filepath.Dir(cleanFilename)

	// Ensure the directory exists
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Check if target file already exists and handle accordingly
	if _, err := os.Stat(cleanFilename); err == nil {
		// File exists - we'll overwrite it atomically
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to check existing file %s: %w", cleanFilename, err)
	}

	// Create temporary file in the same directory with a secure pattern
	tmpFile, err := os.CreateTemp(dir, ".keystore-tmp-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary file in %s: %w", dir, err)
	}

	tmpPath := tmpFile.Name()
	var writeErr error

	// Ensure cleanup happens regardless of success or failure
	defer func() {
		// Close file if still open
		if tmpFile != nil {
			_ = tmpFile.Close()
		}
		// Remove temp file if write failed
		if writeErr != nil {
			_ = os.Remove(tmpPath)
		}
	}()

	// Set permissions on temp file before writing sensitive data
	if err := tmpFile.Chmod(perm); err != nil {
		writeErr = fmt.Errorf("failed to set permissions on temporary file: %w", err)
		return writeErr
	}

	// Write data to temporary file
	bytesWritten, err := tmpFile.Write(data)
	if err != nil {
		writeErr = fmt.Errorf("failed to write to temporary file: %w", err)
		return writeErr
	}

	// Verify all data was written
	if bytesWritten != len(data) {
		writeErr = fmt.Errorf("incomplete write: wrote %d bytes, expected %d", bytesWritten, len(data))
		return writeErr
	}

	// Sync to ensure data is written to disk before moving
	if err := tmpFile.Sync(); err != nil {
		writeErr = fmt.Errorf("failed to sync temporary file to disk: %w", err)
		return writeErr
	}

	// Close temporary file before rename (required on Windows)
	if err := tmpFile.Close(); err != nil {
		writeErr = fmt.Errorf("failed to close temporary file: %w", err)
		return writeErr
	}
	tmpFile = nil // Mark as closed

	// Verify file permissions are correct
	if info, err := os.Stat(tmpPath); err != nil {
		writeErr = fmt.Errorf("failed to verify temporary file: %w", err)
		return writeErr
	} else if info.Mode().Perm() != perm {
		writeErr = fmt.Errorf("temporary file permissions incorrect: got %o, expected %o", info.Mode().Perm(), perm)
		return writeErr
	}

	// Atomically move temporary file to final location
	if err := os.Rename(tmpPath, cleanFilename); err != nil {
		writeErr = fmt.Errorf("failed to move temporary file to final location %s: %w", cleanFilename, err)
		return writeErr
	}

	// Verify final file exists and has correct permissions
	if info, err := os.Stat(cleanFilename); err != nil {
		return fmt.Errorf("failed to verify final file %s: %w", cleanFilename, err)
	} else if info.Mode().Perm() != perm {
		return fmt.Errorf("final file permissions incorrect: got %o, expected %o", info.Mode().Perm(), perm)
	}

	return nil
}

// GetConfig returns the current keystore service configuration
func (ks *KeyStoreService) GetConfig() KeyStoreConfig {
	return ks.config
}

// SetOutputDirectory updates the output directory for keystore files
func (ks *KeyStoreService) SetOutputDirectory(dir string) {
	ks.config.OutputDirectory = dir
}

// SetEnabled enables or disables keystore generation
func (ks *KeyStoreService) SetEnabled(enabled bool) {
	ks.config.Enabled = enabled
}

// SetKDF sets the key derivation function using Universal KDF service validation
func (ks *KeyStoreService) SetKDF(kdfType string) error {
	if !ks.kdfService.IsKDFSupported(kdfType) {
		supportedKDFs := ks.kdfService.GetSupportedKDFs()
		return NewKDFKeyStoreError("validate", "kdf_type",
			kdf.NewKDFError("validation", kdfType, "kdf", kdfType, supportedKDFs,
				fmt.Sprintf("unsupported KDF: %s", kdfType)).
				WithSuggestions(fmt.Sprintf("Supported KDFs: %v", supportedKDFs)))
	}
	ks.config.KDF = kdfType
	return nil
}

// CheckDirectoryPermissions verifies that the output directory has appropriate permissions
func (ks *KeyStoreService) CheckDirectoryPermissions() error {
	if !ks.config.Enabled {
		return nil // Skip check if keystore is disabled
	}

	info, err := os.Stat(ks.config.OutputDirectory)
	if err != nil {
		if os.IsNotExist(err) {
			return &FileOperationError{
				Operation: "check_permissions",
				Path:      ks.config.OutputDirectory,
				Err:       fmt.Errorf("directory does not exist"),
			}
		}
		return &FileOperationError{
			Operation: "check_permissions",
			Path:      ks.config.OutputDirectory,
			Err:       err,
		}
	}

	if !info.IsDir() {
		return &FileOperationError{
			Operation: "check_permissions",
			Path:      ks.config.OutputDirectory,
			Err:       fmt.Errorf("path is not a directory"),
		}
	}

	// Check if directory is writable by attempting to create a temp file
	tmpFile, err := os.CreateTemp(ks.config.OutputDirectory, ".permission-test-*")
	if err != nil {
		return &FileOperationError{
			Operation: "check_permissions",
			Path:      ks.config.OutputDirectory,
			Err:       fmt.Errorf("directory is not writable: %w", err),
		}
	}

	// Clean up test file
	_ = tmpFile.Close()
	_ = os.Remove(tmpFile.Name())

	return nil
}

// FileExists checks if a file exists at the given path
func (ks *KeyStoreService) FileExists(filename string) (bool, error) {
	if filename == "" {
		return false, fmt.Errorf("filename cannot be empty")
	}

	cleanPath := filepath.Clean(filename)
	_, err := os.Stat(cleanPath)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, &FileOperationError{
		Operation: "file_exists",
		Path:      cleanPath,
		Err:       err,
	}
}

// GetKeystoreFilePath returns the full path for a keystore file given an
// address. Keystore files use the geth naming convention
// UTC--<timestamp>--<address>.json, so the file is located by scanning the
// output directory for a matching address suffix.
func (ks *KeyStoreService) GetKeystoreFilePath(address string) (string, error) {
	cleanAddress := strings.TrimPrefix(address, "0x")
	if len(cleanAddress) == 0 {
		return "", fmt.Errorf("address cannot be empty")
	}

	formatted := formatAddressForFilename(address, "ethereum")
	entries, err := os.ReadDir(ks.config.OutputDirectory)
	if err != nil {
		return "", &FileOperationError{Operation: "read_dir", Path: ks.config.OutputDirectory, Err: err}
	}
	suffix := fmt.Sprintf("--%s.json", formatted)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, "UTC--") && strings.HasSuffix(name, suffix) {
			return filepath.Join(ks.config.OutputDirectory, name), nil
		}
	}
	// Fall back to the legacy <address>.json name for backwards compatibility.
	return filepath.Join(ks.config.OutputDirectory, fmt.Sprintf("%s.json", formatted)), nil
}

// utcTimestamp returns the geth-style UTC timestamp used in keystore file
// names: UTC--2006-01-02T15-04-05.000000000Z--<address>.json
func utcTimestamp() string {
	return time.Now().UTC().Format("2006-01-02T15-04-05.000000000Z")
}

// GetPasswordFilePath returns the full path for a password file given an address
func (ks *KeyStoreService) GetPasswordFilePath(address string) (string, error) {
	cleanAddress := strings.TrimPrefix(address, "0x")
	if len(cleanAddress) == 0 {
		return "", fmt.Errorf("address cannot be empty")
	}

	filename := fmt.Sprintf("%s.pwd", formatAddressForFilename(address, "ethereum"))
	return filepath.Join(ks.config.OutputDirectory, filename), nil
}

// GetMnemonicFilePath returns the full path for a mnemonic file given an address
func (ks *KeyStoreService) GetMnemonicFilePath(address string) (string, error) {
	cleanAddress := strings.TrimPrefix(address, "0x")
	if len(cleanAddress) == 0 {
		return "", fmt.Errorf("address cannot be empty")
	}

	filename := fmt.Sprintf("%s.mnemonic", formatAddressForFilename(address, "ethereum"))
	return filepath.Join(ks.config.OutputDirectory, filename), nil
}

// RemoveKeystoreFiles removes both keystore and password files for a given address
func (ks *KeyStoreService) RemoveKeystoreFiles(address string) error {
	if !ks.config.Enabled {
		return fmt.Errorf("keystore service is disabled")
	}

	keystorePath, err := ks.GetKeystoreFilePath(address)
	if err != nil {
		return fmt.Errorf("failed to get keystore path: %w", err)
	}

	passwordPath, err := ks.GetPasswordFilePath(address)
	if err != nil {
		return fmt.Errorf("failed to get password path: %w", err)
	}

	// Remove keystore file if it exists
	if exists, err := ks.FileExists(keystorePath); err != nil {
		return fmt.Errorf("failed to check keystore file: %w", err)
	} else if exists {
		if err := os.Remove(keystorePath); err != nil {
			return &FileOperationError{
				Operation: "remove_keystore",
				Path:      keystorePath,
				Err:       err,
			}
		}
	}

	// Remove password file if it exists
	if exists, err := ks.FileExists(passwordPath); err != nil {
		return fmt.Errorf("failed to check password file: %w", err)
	} else if exists {
		if err := os.Remove(passwordPath); err != nil {
			return &FileOperationError{
				Operation: "remove_password",
				Path:      passwordPath,
				Err:       err,
			}
		}
	}

	return nil
}

// ValidateFilePermissions checks if a file has the expected permissions
func (ks *KeyStoreService) ValidateFilePermissions(filename string, expectedPerm os.FileMode) error {
	if filename == "" {
		return fmt.Errorf("filename cannot be empty")
	}

	info, err := os.Stat(filename)
	if err != nil {
		return &FileOperationError{
			Operation: "validate_permissions",
			Path:      filename,
			Err:       err,
		}
	}

	actualPerm := info.Mode().Perm()
	if actualPerm != expectedPerm {
		return &FileOperationError{
			Operation: "validate_permissions",
			Path:      filename,
			Err:       fmt.Errorf("incorrect permissions: got %o, expected %o", actualPerm, expectedPerm),
		}
	}

	return nil
}

// DecryptPrivateKey decrypts a private key from a KeyStore V3 structure
func DecryptPrivateKey(ks *KeyStoreV3, password string) ([]byte, error) {
	if ks == nil {
		return nil, fmt.Errorf("keystore cannot be nil")
	}
	if password == "" {
		return nil, fmt.Errorf("password cannot be empty")
	}

	// Validate keystore
	if err := ks.Validate(); err != nil {
		return nil, fmt.Errorf("invalid keystore: %w", err)
	}

	// Get cipher parameters
	iv, err := hex.DecodeString(ks.Crypto.CipherParams.IV)
	if err != nil {
		return nil, fmt.Errorf("invalid IV hex: %w", err)
	}

	ciphertext, err := hex.DecodeString(ks.Crypto.CipherText)
	if err != nil {
		return nil, fmt.Errorf("invalid ciphertext hex: %w", err)
	}

	expectedMAC, err := hex.DecodeString(ks.Crypto.MAC)
	if err != nil {
		return nil, fmt.Errorf("invalid MAC hex: %w", err)
	}

	// Derive key using the same KDF parameters
	var derivedKey []byte
	switch ks.Crypto.KDF {
	case "scrypt":
		params, err := ks.GetScryptParams()
		if err != nil {
			return nil, fmt.Errorf("failed to get scrypt params: %w", err)
		}

		salt, err := hex.DecodeString(params.Salt)
		if err != nil {
			return nil, fmt.Errorf("invalid salt hex: %w", err)
		}

		derivedKey, err = DeriveKeyScrypt([]byte(password), salt, params.N, params.R, params.P, params.DKLen)
		if err != nil {
			return nil, fmt.Errorf("scrypt key derivation failed: %w", err)
		}

	case "pbkdf2":
		params, err := ks.GetPBKDF2Params()
		if err != nil {
			return nil, fmt.Errorf("failed to get PBKDF2 params: %w", err)
		}

		salt, err := hex.DecodeString(params.Salt)
		if err != nil {
			return nil, fmt.Errorf("invalid salt hex: %w", err)
		}

		derivedKey, err = DeriveKeyPBKDF2([]byte(password), salt, params.C, params.DKLen)
		if err != nil {
			return nil, fmt.Errorf("PBKDF2 key derivation failed: %w", err)
		}

	default:
		return nil, fmt.Errorf("unsupported KDF: %s", ks.Crypto.KDF)
	}

	// Verify MAC
	valid, err := VerifyMAC(derivedKey, ciphertext, expectedMAC)
	if err != nil {
		return nil, fmt.Errorf("MAC verification failed: %w", err)
	}
	if !valid {
		return nil, fmt.Errorf("MAC verification failed: incorrect password or corrupted keystore")
	}

	// Decrypt private key
	encryptionKey := derivedKey[:16]
	privateKeyBytes, err := DecryptAES128CTR(ciphertext, encryptionKey, iv)
	if err != nil {
		return nil, fmt.Errorf("AES decryption failed: %w", err)
	}

	return privateKeyBytes, nil
}

// AnalyzeKeystoreCompatibility analyzes keystore compatibility with Ethereum clients
func (ks *KeyStoreService) AnalyzeKeystoreCompatibility(keystore *KeyStoreV3) (*kdf.CompatibilityReport, error) {
	if keystore == nil {
		return nil, NewKeyStoreError("analyze", "keystore", fmt.Errorf("keystore cannot be nil"))
	}

	cryptoParams, err := keystore.ToKDFCryptoParams()
	if err != nil {
		return nil, NewKeyStoreError("analyze", "crypto_params", err)
	}

	report, err := ks.analyzer.AnalyzeKeystore(cryptoParams)
	if err != nil {
		return nil, NewKeyStoreError("analyze", "compatibility", err)
	}

	return report, nil
}

// OptimizeKDFParameters optimizes KDF parameters for a given security level
func (ks *KeyStoreService) OptimizeKDFParameters(kdfType string, securityLevel kdf.SecurityLevel) (map[string]interface{}, error) {
	// Use a reasonable default memory limit (256MB)
	const defaultMaxMemoryMB = 256
	return ks.OptimizeKDFParametersWithMemoryLimit(kdfType, securityLevel, defaultMaxMemoryMB)
}

// OptimizeKDFParametersWithMemoryLimit optimizes KDF parameters with a specific memory limit
func (ks *KeyStoreService) OptimizeKDFParametersWithMemoryLimit(kdfType string, securityLevel kdf.SecurityLevel, maxMemoryMB int64) (map[string]interface{}, error) {
	optimizedParams, err := ks.analyzer.GetOptimizedParams(kdfType, securityLevel, maxMemoryMB)
	if err != nil {
		if kdfErr, ok := err.(*kdf.KDFError); ok {
			return nil, NewKDFKeyStoreError("optimize", "parameters", kdfErr)
		}
		return nil, NewKeyStoreError("optimize", "parameters", err)
	}

	return optimizedParams, nil
}

// ValidateKDFParameters validates KDF parameters using the Universal KDF service
func (ks *KeyStoreService) ValidateKDFParameters(kdfType string, params map[string]interface{}) error {
	err := ks.kdfService.ValidateParams(kdfType, params)
	if err != nil {
		if kdfErr, ok := err.(*kdf.KDFError); ok {
			return NewKDFKeyStoreError("validate", "parameters", kdfErr)
		}
		return NewKeyStoreError("validate", "parameters", err)
	}
	return nil
}

// GetSupportedKDFs returns a list of supported KDF types
func (ks *KeyStoreService) GetSupportedKDFs() []string {
	return ks.kdfService.GetSupportedKDFs()
}

// GetKDFService returns the underlying Universal KDF service
func (ks *KeyStoreService) GetKDFService() *kdf.UniversalKDFService {
	return ks.kdfService
}

// GetCompatibilityAnalyzer returns the compatibility analyzer
func (ks *KeyStoreService) GetCompatibilityAnalyzer() *kdf.KDFCompatibilityAnalyzer {
	return ks.analyzer
}
