package worker

import (
	"context"
	"strings"
	"testing"
	"time"

	"bloco-vgen/internal/config"
	"bloco-vgen/pkg/wallet"
)

// TestPool_SecureLoggingIntegration tests that the pool correctly uses SecureLogger
func TestPool_SecureLoggingIntegration(t *testing.T) {
	// Create a config with logging enabled
	cfg := config.DefaultConfig()
	cfg.Logging.Enabled = true
	cfg.Logging.Level = "debug"
	cfg.Logging.Format = "text"
	cfg.Logging.OutputFile = "" // Use stdout for testing

	// Create pool with secure logging
	pool := NewPoolWithConfig(2, cfg, "ethereum")
	defer func() { _ = pool.Shutdown() }()

	// Start the pool
	err := pool.Start()
	if err != nil {
		t.Fatalf("Failed to start pool: %v", err)
	}

	// Test wallet generation with logging
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	criteria := wallet.GenerationCriteria{
		Prefix:     "a",
		Suffix:     "",
		IsChecksum: false,
	}

	result, err := pool.GenerateWalletWithContext(ctx, criteria)
	if err != nil {
		t.Fatalf("Failed to generate wallet: %v", err)
	}

	// Verify result
	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if result.Wallet == nil {
		t.Fatal("Expected non-nil wallet")
	}

	if !strings.HasPrefix(strings.ToLower(result.Wallet.Address), "0xa") {
		t.Errorf("Expected address to start with 'a', got: %s", result.Wallet.Address)
	}

	// Verify that the logger is properly configured
	if pool.logger == nil {
		t.Fatal("Expected pool to have a logger")
	}

	// Test that the logger is enabled for INFO level
	if !pool.logger.IsEnabled(1) { // INFO level
		t.Error("Expected logger to be enabled for INFO level")
	}
}

// TestPool_SecureLoggingDisabled tests that logging can be disabled
func TestPool_SecureLoggingDisabled(t *testing.T) {
	// Create a config with logging disabled
	cfg := config.DefaultConfig()
	cfg.Logging.Enabled = false

	// Create pool with disabled logging
	pool := NewPoolWithConfig(1, cfg, "ethereum")
	defer func() { _ = pool.Shutdown() }()

	// Start the pool
	err := pool.Start()
	if err != nil {
		t.Fatalf("Failed to start pool: %v", err)
	}

	// Test wallet generation still works with disabled logging
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	criteria := wallet.GenerationCriteria{
		Prefix:     "b",
		Suffix:     "",
		IsChecksum: false,
	}

	result, err := pool.GenerateWalletWithContext(ctx, criteria)
	if err != nil {
		t.Fatalf("Failed to generate wallet: %v", err)
	}

	// Verify result
	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if result.Wallet == nil {
		t.Fatal("Expected non-nil wallet")
	}

	if !strings.HasPrefix(strings.ToLower(result.Wallet.Address), "0xb") {
		t.Errorf("Expected address to start with 'b', got: %s", result.Wallet.Address)
	}

	// Verify that the logger exists but is disabled
	if pool.logger == nil {
		t.Fatal("Expected pool to have a logger")
	}

	// Test that the logger is disabled
	if pool.logger.IsEnabled(1) { // INFO level
		t.Error("Expected logger to be disabled")
	}
}

// TestPool_SecureLoggingErrorHandling tests error logging
func TestPool_SecureLoggingErrorHandling(t *testing.T) {
	// Create a config with logging enabled
	cfg := config.DefaultConfig()
	cfg.Logging.Enabled = true
	cfg.Logging.Level = "debug"
	cfg.Logging.Format = "text"
	cfg.Logging.OutputFile = "" // Use stdout for testing

	// Create pool with secure logging
	pool := NewPoolWithConfig(1, cfg, "ethereum")
	defer func() { _ = pool.Shutdown() }()

	// Start the pool
	err := pool.Start()
	if err != nil {
		t.Fatalf("Failed to start pool: %v", err)
	}

	// Test cancellation error logging
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	criteria := wallet.GenerationCriteria{
		Prefix:     "aaaaaaaaaaaaaaaaaaaa", // Very difficult pattern to force timeout
		Suffix:     "",
		IsChecksum: false,
	}

	result, err := pool.GenerateWalletWithContext(ctx, criteria)

	// Should get a cancellation error
	if err == nil {
		t.Fatal("Expected cancellation error")
	}

	if result != nil {
		t.Error("Expected nil result on cancellation")
	}

	// Verify error message
	if !strings.Contains(err.Error(), "generation cancelled") {
		t.Errorf("Expected cancellation error message, got: %v", err)
	}
}

// TestPool_BackwardCompatibility tests that the old NewPool function still works
func TestPool_BackwardCompatibility(t *testing.T) {
	// Test that the old NewPool function still works
	pool := NewPool(2, "ethereum")
	defer func() { _ = pool.Shutdown() }()

	// Verify pool was created successfully
	if pool == nil {
		t.Fatal("Expected non-nil pool")
	}

	if pool.threadCount != 2 {
		t.Errorf("Expected thread count 2, got %d", pool.threadCount)
	}

	// Verify logger was created (should use default config)
	if pool.logger == nil {
		t.Fatal("Expected pool to have a logger")
	}

	// Should be enabled by default
	if !pool.logger.IsEnabled(1) { // INFO level
		t.Error("Expected logger to be enabled by default")
	}
}
