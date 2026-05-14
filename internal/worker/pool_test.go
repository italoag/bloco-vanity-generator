package worker

import (
	"context"
	"fmt"
	"testing"
	"time"

	"bloco-vgen/pkg/wallet"
)

func TestNewPool(t *testing.T) {
	tests := []struct {
		name        string
		threadCount int
		expectError bool
	}{
		{
			name:        "valid thread count",
			threadCount: 4,
			expectError: false,
		},
		{
			name:        "single thread",
			threadCount: 1,
			expectError: false,
		},
		{
			name:        "zero threads",
			threadCount: 0,
			expectError: false, // Should handle gracefully (converts to 1)
		},
		{
			name:        "negative threads",
			threadCount: -1,
			expectError: false, // Should handle gracefully (converts to 1)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool := NewPool(tt.threadCount, "ethereum")

			if pool == nil {
				t.Fatal("NewPool() returned nil")
			}

			expectedThreadCount := tt.threadCount
			if tt.threadCount <= 0 {
				expectedThreadCount = 1 // Pool converts invalid values to 1
			}
			if pool.threadCount != expectedThreadCount {
				t.Errorf("NewPool() threadCount = %v, expected %v", pool.threadCount, expectedThreadCount)
			}

			if pool.isRunning {
				t.Error("New pool should not be running by default")
			}
		})
	}
}

func TestPool_StartShutdown(t *testing.T) {
	pool := NewPool(2, "ethereum")

	// Test initial state
	if pool.isRunning {
		t.Error("Pool should not be running initially")
	}

	// Test start
	err := pool.Start()
	if err != nil {
		t.Fatalf("Start() returned error: %v", err)
	}

	if !pool.isRunning {
		t.Error("Pool should be running after Start()")
	}

	// Test shutdown
	err = pool.Shutdown()
	if err != nil {
		t.Fatalf("Shutdown() returned error: %v", err)
	}

	if pool.isRunning {
		t.Error("Pool should not be running after Shutdown()")
	}
}

func TestPool_GetStatsCollector(t *testing.T) {
	pool := NewPool(2, "ethereum")

	collector := pool.GetStatsCollector()
	if collector == nil {
		t.Fatal("GetStatsCollector() returned nil")
	}
}

func TestPool_GenerateWalletWithContext_SimplePattern(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping wallet generation test in short mode")
	}

	pool := NewPool(2, "ethereum")
	err := pool.Start()
	if err != nil {
		t.Fatalf("Failed to start pool: %v", err)
	}
	defer func() { _ = pool.Shutdown() }()

	// Test with a simple pattern that should be found quickly
	criteria := wallet.GenerationCriteria{
		Prefix:     "a", // Single character should be found quickly
		Suffix:     "",
		IsChecksum: false,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := pool.GenerateWalletWithContext(ctx, criteria)
	if err != nil {
		t.Fatalf("GenerateWalletWithContext() returned error: %v", err)
	}

	if result == nil {
		t.Fatal("GenerateWalletWithContext() returned nil result")
	}

	if result.Wallet == nil {
		t.Fatal("Generated wallet is nil")
	}

	// Verify the address starts with the prefix
	address := result.Wallet.Address
	if len(address) < 4 { // "0x" + at least one char
		t.Fatalf("Address too short: %s", address)
	}

	// Remove 0x prefix and check
	addrWithoutPrefix := address[2:]
	if len(addrWithoutPrefix) > 0 && addrWithoutPrefix[0] != 'a' && addrWithoutPrefix[0] != 'A' {
		t.Errorf("Address %s does not start with prefix 'a'", address)
	}

	// Verify private key is not empty
	if result.Wallet.PrivateKey == "" {
		t.Error("Private key should not be empty")
	}

	// Verify attempts is positive
	if result.Attempts <= 0 {
		t.Error("Attempts should be positive")
	}

	// Verify duration is positive
	if result.Duration <= 0 {
		t.Error("Duration should be positive")
	}
}

func TestPool_GenerateWalletWithContext_Mnemonic(t *testing.T) {
	pool := NewPool(1, "ethereum")
	if err := pool.Start(); err != nil {
		t.Fatalf("Failed to start pool: %v", err)
	}
	defer func() { _ = pool.Shutdown() }()

	criteria := wallet.GenerationCriteria{
		Prefix:      "",
		Suffix:      "",
		IsChecksum:  false,
		UseMnemonic: true,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := pool.GenerateWalletWithContext(ctx, criteria)
	if err != nil {
		t.Fatalf("GenerateWalletWithContext() returned error: %v", err)
	}

	if result == nil || result.Wallet == nil {
		t.Fatal("Expected a wallet result when using mnemonic generation")
	}

	if result.Wallet.Mnemonic == "" {
		t.Error("Expected mnemonic to be populated when using mnemonic generation")
	}

	if result.Wallet.PrivateKey == "" {
		t.Error("Private key should not be empty when using mnemonic generation")
	}
}

func TestPool_GenerateWalletWithContext_Cancellation(t *testing.T) {
	pool := NewPool(1, "ethereum")
	err := pool.Start()
	if err != nil {
		t.Fatalf("Failed to start pool: %v", err)
	}
	defer func() { _ = pool.Shutdown() }()

	// Use a very difficult pattern that would take a long time
	criteria := wallet.GenerationCriteria{
		Prefix:     "aaaaaaaaaaaaaaaaaaaa", // This should take a while to find
		Suffix:     "",
		IsChecksum: false,
	}

	// Create a context that will be cancelled quickly
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	start := time.Now()
	result, err := pool.GenerateWalletWithContext(ctx, criteria)
	duration := time.Since(start)

	// Should return an error due to cancellation
	if err == nil {
		t.Error("Expected context cancellation error, but got nil")
	}

	if result != nil {
		t.Error("Expected nil result on cancellation")
	}

	// Should not take much longer than the timeout
	if duration > 2*time.Second {
		t.Errorf("Cancellation took too long: %v", duration)
	}
}

func TestPool_GenerateWalletWithContext_MultipleWorkers(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping multi-worker test in short mode")
	}

	// Test with multiple threads
	threadCounts := []int{1, 2, 4}

	for _, threads := range threadCounts {
		t.Run(fmt.Sprintf("threads_%d", threads), func(t *testing.T) {
			pool := NewPool(threads, "ethereum")
			err := pool.Start()
			if err != nil {
				t.Fatalf("Failed to start pool: %v", err)
			}
			defer func() { _ = pool.Shutdown() }()

			criteria := wallet.GenerationCriteria{
				Prefix:     "b",
				Suffix:     "",
				IsChecksum: false,
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			result, err := pool.GenerateWalletWithContext(ctx, criteria)
			if err != nil {
				t.Fatalf("GenerateWalletWithContext() with %d threads returned error: %v", threads, err)
			}

			if result == nil {
				t.Fatalf("GenerateWalletWithContext() with %d threads returned nil result", threads)
			}

			t.Logf("Generated with %d threads in %d attempts (%v)",
				threads, result.Attempts, result.Duration)
		})
	}
}

func TestPool_GenerateWalletWithContext_InvalidCriteria(t *testing.T) {
	pool := NewPool(2, "ethereum")
	err := pool.Start()
	if err != nil {
		t.Fatalf("Failed to start pool: %v", err)
	}
	defer func() { _ = pool.Shutdown() }()

	// Test with empty criteria (should still work)
	criteria := wallet.GenerationCriteria{
		Prefix:     "",
		Suffix:     "",
		IsChecksum: false,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	result, err := pool.GenerateWalletWithContext(ctx, criteria)
	if err != nil {
		t.Fatalf("GenerateWalletWithContext() with empty criteria returned error: %v", err)
	}

	if result == nil {
		t.Fatal("GenerateWalletWithContext() with empty criteria returned nil result")
	}

	// Any address should be valid for empty criteria
	if result.Wallet == nil || result.Wallet.Address == "" {
		t.Error("Should generate valid wallet even with empty criteria")
	}
}

func BenchmarkIsValidBlocoAddress(b *testing.B) {
	address := "0x71C7656EC7ab88b098defB751B7401B5f6d8976F"
	prefix := "71"
	suffix := "6f"

	b.Run("NoChecksum", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			matchesCriteria(address, prefix, suffix, false, "ethereum")
		}
	})

	b.Run("WithChecksum", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			matchesCriteria(address, prefix, suffix, true, "ethereum")
		}
	})

	b.Run("WithChecksum_NoMatch", func(b *testing.B) {
		// Address that doesn't match prefix, so optimization should skip checksum
		badPrefix := "FF"
		for i := 0; i < b.N; i++ {
			matchesCriteria(address, badPrefix, suffix, true, "ethereum")
		}
	})
}
