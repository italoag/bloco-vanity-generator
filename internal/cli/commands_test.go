package cli

import (
	"context"
	"strings"
	"testing"
	"time"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/sha3"

	"bloco-vgen/internal/config"
	"bloco-vgen/internal/engine"
	"bloco-vgen/pkg/wallet"
)

func TestGetGenerationCriteriaReadsCaseSensitiveFlag(t *testing.T) {
	app := NewApplication(config.DefaultConfig(), "test", "test", "test")
	cmd := app.rootCmd

	if err := cmd.ParseFlags([]string{"--prefix", "DEAD", "--checksum", "--case-sensitive"}); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}

	criteria, err := app.getGenerationCriteria(cmd)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !criteria.CaseSensitive {
		t.Fatalf("expected CaseSensitive to be true")
	}

	if !criteria.IsChecksum {
		t.Fatalf("expected IsChecksum to be true")
	}
}

func TestGetGenerationCriteriaRejectsCaseSensitiveWithoutChecksum(t *testing.T) {
	app := NewApplication(config.DefaultConfig(), "test", "test", "test")
	cmd := app.rootCmd

	if err := cmd.ParseFlags([]string{"--prefix", "DEAD", "--case-sensitive"}); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}

	_, err := app.getGenerationCriteria(cmd)
	if err == nil {
		t.Fatalf("expected validation error")
	}

	if !strings.Contains(err.Error(), "case-sensitive matching requires checksum validation") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetBenchmarkOptionsResolvesAutoToCPU(t *testing.T) {
	app := NewApplication(config.DefaultConfig(), "test", "test", "test")
	cmd := benchmarkCommandForTest(t, app)

	if err := cmd.ParseFlags([]string{
		"--attempts", "10",
		"--duration", "1s",
		"--engine", "auto",
		"--batch-size", "2",
		"--network", "ethereum",
		"--prefix", "ab",
		"--format", "json",
		"--tui=false",
	}); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}

	options, err := app.getBenchmarkOptions(cmd)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if options.Engine != engine.NameCPU {
		t.Fatalf("expected resolved engine cpu, got %q", options.Engine)
	}

	if options.RequestedEngine != engine.NameAuto {
		t.Fatalf("expected requested engine auto, got %q", options.RequestedEngine)
	}

	if options.Criteria.Prefix != "ab" {
		t.Fatalf("expected prefix ab, got %q", options.Criteria.Prefix)
	}

	if options.Format != benchmarkFormatJSON {
		t.Fatalf("expected json format, got %q", options.Format)
	}

	if options.UseTUI {
		t.Fatalf("expected TUI disabled")
	}
}

func TestGetBenchmarkOptionsHandlesMetalAvailability(t *testing.T) {
	app := NewApplication(config.DefaultConfig(), "test", "test", "test")
	cmd := benchmarkCommandForTest(t, app)

	if err := cmd.ParseFlags([]string{"--engine", "metal"}); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}

	_, err := app.getBenchmarkOptions(cmd)
	if engine.MetalAvailable() {
		if err != nil {
			t.Fatalf("expected metal to resolve, got %v", err)
		}
		return
	}

	if err == nil {
		t.Fatalf("expected metal availability error")
	}
	if !strings.Contains(err.Error(), "metal engine is not available yet") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunBenchmarkEngineProcessesRealAttempts(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Worker.ThreadCount = 1
	cfg.Logging.Enabled = false
	app := NewApplication(cfg, "test", "test", "test")

	options := benchmarkOptions{
		Attempts:        3,
		Duration:        time.Second,
		Engine:          engine.NameCPU,
		RequestedEngine: engine.NameCPU,
		Network:         "ethereum",
		BatchSize:       1,
		Format:          benchmarkFormatJSON,
		Criteria: wallet.GenerationCriteria{
			Network: "ethereum",
		},
	}

	result, err := app.runBenchmarkEngine(context.Background(), options, time.Hour, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.TotalAttempts != int64(options.Attempts) {
		t.Fatalf("expected %d attempts, got %d", options.Attempts, result.TotalAttempts)
	}

	if result.Engine != engine.NameCPU {
		t.Fatalf("expected cpu engine, got %q", result.Engine)
	}

	if result.AverageSpeed <= 0 {
		t.Fatalf("expected positive average speed, got %f", result.AverageSpeed)
	}

	if result.ScalarBaseMultDuration <= 0 {
		t.Fatalf("expected scalar timing to be recorded")
	}
}

func TestBenchmarkEthereumAddressMatchesGoEthereum(t *testing.T) {
	privateKeyBytes := make([]byte, 32)
	privateKeyBytes[31] = 1

	privateKey, err := ethcrypto.ToECDSA(privateKeyBytes)
	if err != nil {
		t.Fatalf("failed to create private key: %v", err)
	}

	expected := ethcrypto.PubkeyToAddress(privateKey.PublicKey).Hex()
	actual := engine.EthereumAddressFromCoordinates(
		privateKey.PublicKey.X.FillBytes(make([]byte, 32)),
		privateKey.PublicKey.Y.FillBytes(make([]byte, 32)),
		sha3.NewLegacyKeccak256(),
	)

	if !strings.EqualFold(expected, actual) {
		t.Fatalf("expected %s, got %s", expected, actual)
	}
}

func benchmarkCommandForTest(t *testing.T, app *Application) *cobra.Command {
	t.Helper()

	for _, cmd := range app.rootCmd.Commands() {
		if cmd.Name() == "benchmark" {
			return cmd
		}
	}

	t.Fatalf("benchmark command not found")
	return nil
}
