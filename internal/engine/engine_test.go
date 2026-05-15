package engine

import (
	"context"
	"strings"
	"testing"
	"time"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"golang.org/x/crypto/sha3"

	"bloco-vgen/pkg/wallet"
)

func TestResolveAutoFallsBackToAvailableEngine(t *testing.T) {
	selection, err := Resolve(NameAuto)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if selection.Requested != NameAuto {
		t.Fatalf("expected requested auto, got %q", selection.Requested)
	}

	if selection.Resolved != NameCPU && selection.Resolved != NameMetal {
		t.Fatalf("unexpected resolved engine %q", selection.Resolved)
	}
}

func TestResolveMetalReturnsClearUnavailableError(t *testing.T) {
	if MetalAvailable() {
		t.Skip("metal backend available in this build")
	}

	_, err := Resolve(NameMetal)
	if err == nil {
		t.Fatalf("expected metal unavailable error")
	}

	if !strings.Contains(err.Error(), "metal engine is not available yet") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNormalizeMetalValidationMode(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{name: "empty defaults to full", input: "", expected: MetalValidationFull},
		{name: "full", input: "full", expected: MetalValidationFull},
		{name: "sampled rejected", input: " SAMPLED ", wantErr: true},
		{name: "none rejected", input: "none", wantErr: true},
		{name: "invalid", input: "fast", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := NormalizeMetalValidationMode(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if actual != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, actual)
			}
		})
	}
}

func TestCPUEngineRunBenchmarkProcessesAttempts(t *testing.T) {
	engine := NewCPUEngine()
	result, err := engine.RunBenchmark(context.Background(), BenchmarkOptions{
		Attempts:        3,
		Duration:        time.Second,
		BatchSize:       1,
		ThreadCount:     1,
		Network:         "ethereum",
		RequestedEngine: NameCPU,
		Criteria: wallet.GenerationCriteria{
			Network: "ethereum",
		},
	}, time.Hour, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.Engine != NameCPU {
		t.Fatalf("expected cpu engine, got %q", result.Engine)
	}

	if result.ThreadCount != 1 {
		t.Fatalf("expected one benchmark thread, got %d", result.ThreadCount)
	}

	if result.TotalAttempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", result.TotalAttempts)
	}

	if result.AverageSpeed <= 0 {
		t.Fatalf("expected positive average speed, got %f", result.AverageSpeed)
	}

	if result.ScalarBaseMultDuration <= 0 {
		t.Fatalf("expected scalar timing to be recorded")
	}
}

func TestValidSecp256k1PrivateKeyRange(t *testing.T) {
	zero := make([]byte, 32)
	if validSecp256k1PrivateKey(zero) {
		t.Fatalf("expected zero private key to be invalid")
	}

	one := make([]byte, 32)
	one[31] = 1
	if !validSecp256k1PrivateKey(one) {
		t.Fatalf("expected private key one to be valid")
	}

	order := secp256k1Order.FillBytes(make([]byte, 32))
	if validSecp256k1PrivateKey(order) {
		t.Fatalf("expected secp256k1 order to be invalid")
	}
}

func TestZeroBytesClearsSensitiveBuffer(t *testing.T) {
	values := []byte{1, 2, 3, 4}
	zeroBytes(values)
	for i, value := range values {
		if value != 0 {
			t.Fatalf("expected byte %d to be zero, got %d", i, value)
		}
	}
}

func TestMetalEngineRunBenchmarkWhenAvailable(t *testing.T) {
	if !MetalAvailable() {
		t.Skip("metal backend unavailable in this build")
	}

	benchmarkEngine, err := NewMetalEngine()
	if err != nil {
		t.Fatalf("expected metal engine, got %v", err)
	}

	result, err := benchmarkEngine.RunBenchmark(context.Background(), BenchmarkOptions{
		Attempts:        16,
		Duration:        time.Second,
		BatchSize:       16,
		ThreadCount:     1,
		Network:         "ethereum",
		RequestedEngine: NameMetal,
		Criteria: wallet.GenerationCriteria{
			Network: "ethereum",
			Prefix:  "ab",
			Suffix:  "cd",
		},
	}, time.Hour, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.Engine != NameMetal {
		t.Fatalf("expected metal engine, got %q", result.Engine)
	}

	if !result.IsHybrid {
		t.Fatalf("expected hybrid metal benchmark")
	}

	if result.DeviceName == "" {
		t.Fatalf("expected metal device name")
	}

	if result.KernelDuration <= 0 {
		t.Fatalf("expected kernel duration to be recorded")
	}

	if result.EntropyDuration <= 0 {
		t.Fatalf("expected entropy timing to be recorded")
	}

	if result.ScalarBaseMultDuration <= 0 {
		t.Fatalf("expected scalar timing to be recorded")
	}

	if result.HashFormatDuration <= 0 {
		t.Fatalf("expected hash timing to be recorded")
	}
}

func TestMetalEngineRunBenchmarkSamplesBatchesWhenAvailable(t *testing.T) {
	if !MetalAvailable() {
		t.Skip("metal backend unavailable in this build")
	}

	benchmarkEngine, err := NewMetalEngine()
	if err != nil {
		t.Fatalf("expected metal engine, got %v", err)
	}

	var samples []Sample
	result, err := benchmarkEngine.RunBenchmark(context.Background(), BenchmarkOptions{
		Attempts:        6,
		Duration:        time.Second,
		BatchSize:       2,
		ThreadCount:     1,
		Network:         "ethereum",
		RequestedEngine: NameMetal,
		Criteria: wallet.GenerationCriteria{
			Network: "ethereum",
		},
	}, time.Nanosecond, func(sample Sample) {
		samples = append(samples, sample)
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.TotalAttempts != 6 {
		t.Fatalf("expected 6 attempts, got %d", result.TotalAttempts)
	}

	if result.BatchSize != 2 {
		t.Fatalf("expected batch size 2, got %d", result.BatchSize)
	}

	if len(samples) < 2 {
		t.Fatalf("expected multiple batch samples, got %d", len(samples))
	}

	if samples[len(samples)-1].Result == nil {
		t.Fatalf("expected final sample to include result")
	}
}

func TestMetalEngineRunBenchmarkRejectsNonFullValidation(t *testing.T) {
	if !MetalAvailable() {
		t.Skip("metal backend unavailable in this build")
	}

	benchmarkEngine, err := NewMetalEngine()
	if err != nil {
		t.Fatalf("expected metal engine, got %v", err)
	}

	_, err = benchmarkEngine.RunBenchmark(context.Background(), BenchmarkOptions{
		Attempts:        2,
		Duration:        time.Second,
		BatchSize:       2,
		ThreadCount:     1,
		Network:         "ethereum",
		RequestedEngine: NameMetal,
		MetalValidation: "none",
		Criteria: wallet.GenerationCriteria{
			Network: "ethereum",
		},
	}, time.Hour, nil)
	if err == nil {
		t.Fatalf("expected non-full metal validation to be rejected")
	}
	if !strings.Contains(err.Error(), "Phase 4 requires full CPU verification") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSyntheticMatchingCountsExpectedCriteria(t *testing.T) {
	addresses := generateSyntheticAddresses(16, wallet.GenerationCriteria{
		Prefix: "ab",
		Suffix: "cd",
	})
	prefix, err := patternNibbles("ab")
	if err != nil {
		t.Fatalf("unexpected prefix error: %v", err)
	}
	suffix, err := patternNibbles("cd")
	if err != nil {
		t.Fatalf("unexpected suffix error: %v", err)
	}

	if matches := countAddressMatches(addresses, prefix, suffix); matches != 1 {
		t.Fatalf("expected one synthetic full match, got %d", matches)
	}

	if matches := countAddressMatches(addresses, prefix, nil); matches != 2 {
		t.Fatalf("expected two synthetic prefix matches, got %d", matches)
	}
}

func TestEthereumAddressFromCoordinatesMatchesGoEthereum(t *testing.T) {
	privateKeyBytes := make([]byte, 32)
	privateKeyBytes[31] = 1

	privateKey, err := ethcrypto.ToECDSA(privateKeyBytes)
	if err != nil {
		t.Fatalf("failed to create private key: %v", err)
	}

	expected := ethcrypto.PubkeyToAddress(privateKey.PublicKey).Hex()
	actual := EthereumAddressFromCoordinates(
		privateKey.PublicKey.X.FillBytes(make([]byte, 32)),
		privateKey.PublicKey.Y.FillBytes(make([]byte, 32)),
		sha3.NewLegacyKeccak256(),
	)

	if !strings.EqualFold(expected, actual) {
		t.Fatalf("expected %s, got %s", expected, actual)
	}
}
