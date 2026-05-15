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

func TestBuildStatsTextRowsIncludesDetailedStatistics(t *testing.T) {
	criteria := wallet.GenerationCriteria{
		Prefix:        "BAD",
		Suffix:        "DAD",
		IsChecksum:    true,
		CaseSensitive: true,
	}
	difficulty := calculateDifficulty(criteria)
	probability50 := calculateProbability50(difficulty)

	rows := buildStatsTextRows(criteria, difficulty, probability50)
	values := make(map[string]string, len(rows))
	for _, row := range rows {
		values[row.metric] = row.value
	}

	wantValues := map[string]string{
		"Pattern":             "BADDAD",
		"Pattern Length":      "6 characters",
		"Base Difficulty":     formatLargeNumber(16777216),
		"Total Difficulty":    formatLargeNumber(1073741824),
		"Checksum Multiplier": "64.0x",
		"50% Probability":     formatLargeNumber(744261117),
		"Expected Attempts":   formatLargeNumber(1073741824),
		"Success Rate":        "9.31e-08%",
	}
	for metric, want := range wantValues {
		if got := values[metric]; got != want {
			t.Fatalf("metric %q = %q, want %q; rows=%v", metric, got, want, values)
		}
	}
}

func TestGetGenerationEngineOptionsReadsEnvironment(t *testing.T) {
	t.Setenv(envBlocoEngine, engine.NameCPU)
	t.Setenv(envBlocoGPUBatchSize, "7")

	app := NewApplication(config.DefaultConfig(), "test", "test", "test")
	cmd := app.rootCmd
	if err := cmd.ParseFlags([]string{"--prefix", "ab"}); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}
	criteria, err := app.getGenerationCriteria(cmd)
	if err != nil {
		t.Fatalf("expected criteria, got %v", err)
	}

	selection, options, err := app.getGenerationEngineOptions(cmd, criteria)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if selection.Requested != engine.NameCPU {
		t.Fatalf("expected env requested engine cpu, got %q", selection.Requested)
	}
	if selection.Resolved != engine.NameCPU {
		t.Fatalf("expected cpu engine, got %q", selection.Resolved)
	}
	if options.BatchSize != 7 {
		t.Fatalf("expected env batch size 7, got %d", options.BatchSize)
	}
}

func TestGetGenerationEngineOptionsFlagOverridesEnvironment(t *testing.T) {
	t.Setenv(envBlocoEngine, engine.NameCPU)
	t.Setenv(envBlocoGPUBatchSize, "7")

	app := NewApplication(config.DefaultConfig(), "test", "test", "test")
	cmd := app.rootCmd
	if err := cmd.ParseFlags([]string{"--engine", "auto", "--gpu-batch-size", "3", "--prefix", "ab"}); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}
	criteria, err := app.getGenerationCriteria(cmd)
	if err != nil {
		t.Fatalf("expected criteria, got %v", err)
	}

	selection, options, err := app.getGenerationEngineOptions(cmd, criteria)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if selection.Requested != engine.NameAuto {
		t.Fatalf("expected flag requested engine auto, got %q", selection.Requested)
	}
	if options.BatchSize != 3 {
		t.Fatalf("expected flag batch size 3, got %d", options.BatchSize)
	}
}

func TestGetGenerationEngineOptionsRejectsDisabledCPUVerificationForMetal(t *testing.T) {
	t.Setenv(envBlocoGPUVerifyCPU, "false")

	app := NewApplication(config.DefaultConfig(), "test", "test", "test")
	cmd := app.rootCmd
	if err := cmd.ParseFlags([]string{"--engine", "metal", "--prefix", "ab"}); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}
	criteria, err := app.getGenerationCriteria(cmd)
	if err != nil {
		t.Fatalf("expected criteria, got %v", err)
	}

	_, _, err = app.getGenerationEngineOptions(cmd, criteria)
	if engine.MetalAvailable() {
		if err == nil {
			t.Fatalf("expected disabled CPU verification error")
		}
		if !strings.Contains(err.Error(), "Phase 4 requires full CPU verification") {
			t.Fatalf("unexpected error: %v", err)
		}
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
		"--metal-validation", "full",
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

	if options.MetalValidation != engine.MetalValidationFull {
		t.Fatalf("expected metal validation full, got %q", options.MetalValidation)
	}

	if options.UseTUI {
		t.Fatalf("expected TUI disabled")
	}
}

func TestGetBenchmarkOptionsReadsPhase5Environment(t *testing.T) {
	t.Setenv(envBlocoEngine, engine.NameCPU)
	t.Setenv(envBlocoGPUBatchSize, "7")
	t.Setenv(envBlocoGPUVerifyCPU, "true")

	app := NewApplication(config.DefaultConfig(), "test", "test", "test")
	cmd := benchmarkCommandForTest(t, app)

	if err := cmd.ParseFlags([]string{
		"--attempts", "10",
		"--duration", "1s",
		"--network", "ethereum",
		"--format", "json",
		"--tui=false",
	}); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}

	options, err := app.getBenchmarkOptions(cmd)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if options.RequestedEngine != engine.NameCPU {
		t.Fatalf("expected env engine cpu, got %q", options.RequestedEngine)
	}
	if options.BatchSize != 7 {
		t.Fatalf("expected env batch size 7, got %d", options.BatchSize)
	}
	if options.MetalValidation != engine.MetalValidationFull {
		t.Fatalf("expected safe full validation, got %q", options.MetalValidation)
	}
}

func TestGetBenchmarkOptionsFlagOverridesPhase5Environment(t *testing.T) {
	t.Setenv(envBlocoEngine, engine.NameCPU)
	t.Setenv(envBlocoGPUBatchSize, "7")

	app := NewApplication(config.DefaultConfig(), "test", "test", "test")
	cmd := benchmarkCommandForTest(t, app)

	if err := cmd.ParseFlags([]string{
		"--attempts", "10",
		"--duration", "1s",
		"--engine", "auto",
		"--batch-size", "3",
		"--network", "ethereum",
		"--format", "json",
		"--tui=false",
	}); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}

	options, err := app.getBenchmarkOptions(cmd)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if options.RequestedEngine != engine.NameAuto {
		t.Fatalf("expected flag engine auto, got %q", options.RequestedEngine)
	}
	if options.BatchSize != 3 {
		t.Fatalf("expected flag batch size 3, got %d", options.BatchSize)
	}
}

func TestResolveMetalValidationConfigRejectsDisabledCPUVerificationForMetal(t *testing.T) {
	t.Setenv(envBlocoGPUVerifyCPU, "false")

	app := NewApplication(config.DefaultConfig(), "test", "test", "test")
	cmd := benchmarkCommandForTest(t, app)

	_, err := resolveMetalValidationConfig(cmd, engine.NameMetal, engine.MetalValidationFull)
	if err == nil {
		t.Fatalf("expected disabled CPU verification error")
	}
	if !strings.Contains(err.Error(), "Phase 4 requires full CPU verification") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetBenchmarkOptionsRejectsInvalidPhase5Environment(t *testing.T) {
	t.Setenv(envBlocoGPUBatchSize, "large")

	app := NewApplication(config.DefaultConfig(), "test", "test", "test")
	cmd := benchmarkCommandForTest(t, app)

	if err := cmd.ParseFlags([]string{"--attempts", "10", "--duration", "1s"}); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}

	_, err := app.getBenchmarkOptions(cmd)
	if err == nil {
		t.Fatalf("expected invalid env error")
	}
	if !strings.Contains(err.Error(), envBlocoGPUBatchSize) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetBenchmarkOptionsReadsPhase6ComparisonFlags(t *testing.T) {
	app := NewApplication(config.DefaultConfig(), "test", "test", "test")
	cmd := benchmarkCommandForTest(t, app)

	if err := cmd.ParseFlags([]string{
		"--compare",
		"--compare-patterns", "ab,abcd",
		"--compare-batch-sizes", "2,4",
		"--compare-checksums", "off,on",
		"--attempts", "4",
		"--duration", "1s",
		"--format", "json",
		"--tui=false",
	}); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}

	options, err := app.getBenchmarkOptions(cmd)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !options.Compare {
		t.Fatalf("expected compare mode")
	}
	if len(options.ComparePatterns) != 2 || options.ComparePatterns[0] != "ab" || options.ComparePatterns[1] != "abcd" {
		t.Fatalf("unexpected compare patterns: %#v", options.ComparePatterns)
	}
	if len(options.CompareBatches) != 2 || options.CompareBatches[0] != 2 || options.CompareBatches[1] != 4 {
		t.Fatalf("unexpected compare batches: %#v", options.CompareBatches)
	}
	if len(options.CompareChecksum) != 2 || options.CompareChecksum[0] || !options.CompareChecksum[1] {
		t.Fatalf("unexpected compare checksums: %#v", options.CompareChecksum)
	}
}

func TestGetBenchmarkOptionsRejectsInvalidPhase6ComparisonFlags(t *testing.T) {
	app := NewApplication(config.DefaultConfig(), "test", "test", "test")
	cmd := benchmarkCommandForTest(t, app)

	if err := cmd.ParseFlags([]string{"--compare-batch-sizes", "0"}); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}

	_, err := app.getBenchmarkOptions(cmd)
	if err == nil {
		t.Fatalf("expected invalid comparison flag error")
	}
	if !strings.Contains(err.Error(), "compare-batch-sizes") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetBenchmarkOptionsRejectsInvalidMetalValidation(t *testing.T) {
	app := NewApplication(config.DefaultConfig(), "test", "test", "test")
	cmd := benchmarkCommandForTest(t, app)

	if err := cmd.ParseFlags([]string{"--metal-validation", "fast"}); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}

	_, err := app.getBenchmarkOptions(cmd)
	if err == nil {
		t.Fatalf("expected metal validation error")
	}
	if !strings.Contains(err.Error(), "unsupported metal validation mode") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDecideBenchmarkDefaultKeepsCPUUnlessAllCasesAreStable(t *testing.T) {
	decision, reason := decideBenchmarkDefault([]benchmarkComparisonCase{
		{Decision: engine.NameMetal},
		{Decision: engine.NameCPU},
	})
	if decision != engine.NameCPU {
		t.Fatalf("expected cpu decision, got %q", decision)
	}
	if !strings.Contains(reason, "cpu remains default") {
		t.Fatalf("unexpected reason: %s", reason)
	}

	decision, reason = decideBenchmarkDefault([]benchmarkComparisonCase{
		{Decision: engine.NameMetal},
		{Decision: engine.NameMetal},
	})
	if decision != engine.NameMetal {
		t.Fatalf("expected metal decision, got %q", decision)
	}
	if !strings.Contains(reason, "metal can become default") {
		t.Fatalf("unexpected reason: %s", reason)
	}
}

func TestRunBenchmarkComparisonReportsDefaultDecision(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Worker.ThreadCount = 1
	cfg.Logging.Enabled = false
	app := NewApplication(cfg, "test", "test", "test")

	result, err := app.runBenchmarkComparison(context.Background(), benchmarkOptions{
		Attempts:        2,
		Duration:        time.Second,
		Engine:          engine.NameCPU,
		RequestedEngine: engine.NameAuto,
		Network:         "ethereum",
		BatchSize:       1,
		Format:          benchmarkFormatJSON,
		MetalValidation: engine.MetalValidationFull,
		ComparePatterns: []string{"ab"},
		CompareBatches:  []int{1},
		CompareChecksum: []bool{false},
		Criteria: wallet.GenerationCriteria{
			Network: "ethereum",
		},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Phase != "6" {
		t.Fatalf("expected phase 6, got %q", result.Phase)
	}
	if result.DefaultEngineDecision == "" {
		t.Fatalf("expected default engine decision")
	}
	if len(result.Cases) != 1 {
		t.Fatalf("expected one comparison case, got %d", len(result.Cases))
	}
	if result.Cases[0].CPU == nil {
		t.Fatalf("expected cpu result")
	}
	if engine.MetalAvailable() && result.Cases[0].Metal == nil {
		t.Fatalf("expected metal result when metal is available: %s", result.Cases[0].Error)
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

	if result.MetalAvailable != engine.MetalAvailable() {
		t.Fatalf("expected metal availability diagnostic to be populated")
	}

	if result.CPUThroughput <= 0 {
		t.Fatalf("expected cpu throughput diagnostic")
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

func TestNewTUIEngineInfoCPU(t *testing.T) {
	selection := engine.Selection{Requested: engine.NameCPU, Resolved: engine.NameCPU}
	options := engine.GenerationOptions{
		Network:         "ethereum",
		BatchSize:       1024,
		MetalValidation: engine.MetalValidationFull,
	}

	info := newTUIEngineInfo(selection, options, 4)
	if info.Engine != engine.NameCPU {
		t.Fatalf("expected cpu engine, got %q", info.Engine)
	}
	if info.ThreadCount != 4 {
		t.Fatalf("expected thread count 4, got %d", info.ThreadCount)
	}
	if info.Network != "ethereum" {
		t.Fatalf("expected network ethereum, got %q", info.Network)
	}
	if info.DeviceName != "" || info.BatchSize != 0 || info.MetalValidation != "" {
		t.Fatalf("cpu engine should not carry metal-specific fields, got %+v", info)
	}
}

func TestNewTUIEngineInfoMetal(t *testing.T) {
	if !engine.MetalAvailable() {
		t.Skip("metal backend unavailable in this build")
	}

	selection := engine.Selection{Requested: engine.NameMetal, Resolved: engine.NameMetal}
	options := engine.GenerationOptions{
		Network:         "ethereum",
		BatchSize:       2048,
		MetalValidation: engine.MetalValidationFull,
	}

	info := newTUIEngineInfo(selection, options, 8)
	if info.Engine != engine.NameMetal {
		t.Fatalf("expected metal engine, got %q", info.Engine)
	}
	if info.BatchSize != 2048 {
		t.Fatalf("expected batch size 2048, got %d", info.BatchSize)
	}
	if info.MetalValidation != engine.MetalValidationFull {
		t.Fatalf("expected full validation, got %q", info.MetalValidation)
	}
	if info.DeviceName == "" {
		t.Fatalf("expected metal device name, got empty string")
	}
}

func TestNewBenchmarkTUIEngineInfoCPU(t *testing.T) {
	options := benchmarkOptions{
		Engine:          engine.NameCPU,
		RequestedEngine: engine.NameCPU,
		BatchSize:       5000,
		Network:         "ethereum",
		MetalValidation: engine.MetalValidationFull,
	}

	info := newBenchmarkTUIEngineInfo(options, 6)
	if info.Engine != engine.NameCPU {
		t.Fatalf("expected cpu engine, got %q", info.Engine)
	}
	if info.BatchSize != 5000 {
		t.Fatalf("expected batch size 5000, got %d", info.BatchSize)
	}
	if info.ThreadCount != 6 {
		t.Fatalf("expected thread count 6, got %d", info.ThreadCount)
	}
	if info.DeviceName != "" || info.MetalValidation != "" {
		t.Fatalf("cpu benchmark info should not carry metal-only fields, got %+v", info)
	}
}

func TestWillUseTUIRespectsQuietAndProgress(t *testing.T) {
	app := NewApplication(config.DefaultConfig(), "test", "test", "test")

	if app.willUseTUI(false) {
		t.Fatalf("willUseTUI should be false when showProgress is false")
	}

	app.config.CLI.QuietMode = true
	if app.willUseTUI(true) {
		t.Fatalf("willUseTUI should be false when quiet mode is enabled")
	}

	app = NewApplication(config.DefaultConfig(), "test", "test", "test")
	app.config.TUI.Enabled = false
	if app.willUseTUI(true) {
		t.Fatalf("willUseTUI should be false when TUI is disabled in config")
	}
}
