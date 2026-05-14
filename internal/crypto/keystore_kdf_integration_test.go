package crypto

import (
	"testing"

	"bloco-vgen/internal/crypto/kdf"
)

// TestKeyStoreService_UniversalKDFIntegration tests the integration with Universal KDF service
func TestKeyStoreService_UniversalKDFIntegration(t *testing.T) {
	config := KeyStoreConfig{
		OutputDirectory: t.TempDir(),
		Enabled:         true,
		KDF:             "scrypt",
	}
	service := NewKeyStoreService(config)

	t.Run("supported_kdfs", func(t *testing.T) {
		supportedKDFs := service.GetSupportedKDFs()
		expectedKDFs := []string{"scrypt", "pbkdf2", "pbkdf2-sha256", "pbkdf2-sha512"}

		for _, expected := range expectedKDFs {
			found := false
			for _, supported := range supportedKDFs {
				if supported == expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected KDF %s not found in supported KDFs: %v", expected, supportedKDFs)
			}
		}
	})

	t.Run("kdf_parameter_validation", func(t *testing.T) {
		// Test valid scrypt parameters
		validScryptParams := map[string]interface{}{
			"n":     16384,
			"r":     8,
			"p":     1,
			"dklen": 32,
			"salt":  "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		}

		err := service.ValidateKDFParameters("scrypt", validScryptParams)
		if err != nil {
			t.Errorf("Valid scrypt parameters should not produce error: %v", err)
		}

		// Test invalid scrypt parameters (N not power of 2)
		invalidScryptParams := map[string]interface{}{
			"n":     12345, // Not a power of 2
			"r":     8,
			"p":     1,
			"dklen": 32,
			"salt":  "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		}

		err = service.ValidateKDFParameters("scrypt", invalidScryptParams)
		if err == nil {
			t.Error("Invalid scrypt parameters should produce error")
		}

		// Check if it's a KDF error with suggestions
		if ksErr, ok := err.(*KeyStoreError); ok && ksErr.HasKDFError() {
			kdfErr := ksErr.GetKDFError()
			if kdfErr.Type != "validation" {
				t.Errorf("Expected validation error, got %s", kdfErr.Type)
			}
		} else {
			t.Error("Expected KeyStoreError with KDF error information")
		}
	})

	t.Run("parameter_optimization", func(t *testing.T) {
		// Test parameter optimization for different security levels
		securityLevels := []kdf.SecurityLevel{
			kdf.SecurityLevelLow,
			kdf.SecurityLevelMedium,
			kdf.SecurityLevelHigh,
			kdf.SecurityLevelVeryHigh,
		}

		for _, level := range securityLevels {
			params, err := service.OptimizeKDFParameters("scrypt", level)
			if err != nil {
				t.Errorf("Failed to optimize parameters for %s: %v", level, err)
				continue
			}

			// Verify required parameters are present
			requiredParams := []string{"n", "r", "p", "dklen"}
			for _, param := range requiredParams {
				if _, ok := params[param]; !ok {
					t.Errorf("Missing required parameter %s in optimized params for %s", param, level)
				}
			}
		}
	})

	t.Run("keystore_generation_with_different_kdfs", func(t *testing.T) {
		privateKey := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
		address := "1234567890abcdef1234567890abcdef12345678"

		kdfTypes := []string{"scrypt", "pbkdf2", "pbkdf2-sha256", "pbkdf2-sha512"}

		for _, kdfType := range kdfTypes {
			t.Run(kdfType, func(t *testing.T) {
				// Update service KDF type
				err := service.SetKDF(kdfType)
				if err != nil {
					t.Fatalf("Failed to set KDF to %s: %v", kdfType, err)
				}

				// Generate keystore
				keystore, password, err := service.GenerateKeyStore(privateKey, address, "ethereum")
				if err != nil {
					t.Fatalf("Failed to generate keystore with %s: %v", kdfType, err)
				}

				if keystore == nil {
					t.Fatal("Generated keystore is nil")
				}
				if password == "" {
					t.Fatal("Generated password is empty")
				}

				// Verify KDF type in keystore (normalize for Ethereum KeyStore V3 format)
				expectedKDF := kdfType
				if kdfType == "pbkdf2-sha256" || kdfType == "pbkdf2-sha512" {
					expectedKDF = "pbkdf2" // Ethereum KeyStore V3 uses "pbkdf2" with PRF parameter
				}
				if keystore.Crypto.KDF != expectedKDF {
					t.Errorf("Expected KDF %s, got %s", expectedKDF, keystore.Crypto.KDF)
				}

				// Test compatibility analysis
				report, err := service.AnalyzeKeystoreCompatibility(keystore)
				if err != nil {
					t.Errorf("Failed to analyze keystore compatibility: %v", err)
				} else {
					// The report should show the normalized KDF name
					expectedReportKDF := expectedKDF
					if report.NormalizedKDF != expectedReportKDF {
						t.Errorf("Expected normalized KDF %s in report, got %s", expectedReportKDF, report.NormalizedKDF)
					}
					if !report.Compatible {
						t.Errorf("Keystore should be compatible, but report says it's not")
					}
				}
			})
		}
	})

	t.Run("enhanced_error_handling", func(t *testing.T) {
		// Test unsupported KDF
		err := service.SetKDF("unsupported-kdf")
		if err == nil {
			t.Error("Setting unsupported KDF should produce error")
		}

		// Check if it's a proper KDF error
		if ksErr, ok := err.(*KeyStoreError); ok && ksErr.HasKDFError() {
			kdfErr := ksErr.GetKDFError()
			if kdfErr.Type != "validation" {
				t.Errorf("Expected validation error, got %s", kdfErr.Type)
			}
			if len(kdfErr.GetSuggestions()) == 0 {
				t.Error("Expected suggestions in KDF error")
			}
		} else {
			t.Error("Expected KeyStoreError with KDF error information")
		}
	})

	t.Run("parameter_conversion", func(t *testing.T) {
		// Create a keystore with scrypt
		err := service.SetKDF("scrypt")
		if err != nil {
			t.Fatalf("Failed to set KDF to scrypt: %v", err)
		}
		privateKey := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
		address := "1234567890abcdef1234567890abcdef12345678"

		keystore, _, err := service.GenerateKeyStore(privateKey, address, "ethereum")
		if err != nil {
			t.Fatalf("Failed to generate keystore: %v", err)
		}

		// Test conversion to KDF crypto params
		cryptoParams, err := keystore.ToKDFCryptoParams()
		if err != nil {
			t.Fatalf("Failed to convert to KDF crypto params: %v", err)
		}

		if cryptoParams.KDF != "scrypt" {
			t.Errorf("Expected KDF scrypt, got %s", cryptoParams.KDF)
		}

		// Verify required parameters are present
		requiredParams := []string{"n", "r", "p", "dklen", "salt"}
		for _, param := range requiredParams {
			if _, ok := cryptoParams.KDFParams[param]; !ok {
				t.Errorf("Missing required parameter %s in KDF params", param)
			}
		}

		// Test conversion back from KDF crypto params
		newKeystore := NewKeyStoreV3(address, "ethereum")
		err = newKeystore.FromKDFCryptoParams(cryptoParams)
		if err != nil {
			t.Fatalf("Failed to convert from KDF crypto params: %v", err)
		}

		if newKeystore.Crypto.KDF != "scrypt" {
			t.Errorf("Expected KDF scrypt after conversion, got %s", newKeystore.Crypto.KDF)
		}
	})
}

// TestKeyStoreService_KDFCompatibilityAnalysis tests the compatibility analysis features
func TestKeyStoreService_KDFCompatibilityAnalysis(t *testing.T) {
	config := KeyStoreConfig{
		OutputDirectory: t.TempDir(),
		Enabled:         true,
		KDF:             "scrypt",
	}
	service := NewKeyStoreService(config)

	privateKey := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	address := "1234567890abcdef1234567890abcdef12345678"

	t.Run("compatibility_analysis", func(t *testing.T) {
		keystore, _, err := service.GenerateKeyStore(privateKey, address, "ethereum")
		if err != nil {
			t.Fatalf("Failed to generate keystore: %v", err)
		}

		report, err := service.AnalyzeKeystoreCompatibility(keystore)
		if err != nil {
			t.Fatalf("Failed to analyze compatibility: %v", err)
		}

		// Verify report structure
		if report.KDFType == "" {
			t.Error("KDF type should not be empty in report")
		}
		if report.NormalizedKDF == "" {
			t.Error("Normalized KDF should not be empty in report")
		}
		if report.SecurityLevel == "" {
			t.Error("Security level should not be empty in report")
		}
		if len(report.Parameters) == 0 {
			t.Error("Parameters should not be empty in report")
		}
	})

	t.Run("memory_limit_optimization", func(t *testing.T) {
		// Test optimization with different memory limits
		memoryLimits := []int64{64, 128, 256, 512}

		for _, limit := range memoryLimits {
			params, err := service.OptimizeKDFParametersWithMemoryLimit("scrypt", kdf.SecurityLevelMedium, limit)
			if err != nil {
				t.Errorf("Failed to optimize with memory limit %d MB: %v", limit, err)
				continue
			}

			// Verify N parameter is reasonable for the memory limit
			if n, ok := params["n"].(int); ok {
				// Rough memory calculation: 128 * r * N * p
				if r, ok := params["r"].(int); ok {
					if p, ok := params["p"].(int); ok {
						memoryUsage := int64(128) * int64(r) * int64(n) * int64(p) / (1024 * 1024) // MB
						if memoryUsage > limit*2 {                                                 // Allow some tolerance
							t.Errorf("Memory usage %d MB exceeds limit %d MB for N=%d, r=%d, p=%d",
								memoryUsage, limit, n, r, p)
						}
					}
				}
			}
		}
	})
}
