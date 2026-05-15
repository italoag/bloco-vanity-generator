package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"bloco-vgen/internal/engine"
	"bloco-vgen/internal/tui"
	"bloco-vgen/pkg/wallet"
)

const (
	benchmarkFormatText = "text"
	benchmarkFormatJSON = "json"
)

const (
	envBlocoEngine       = "BLOCO_ENGINE"
	envBlocoGPUBatchSize = "BLOCO_GPU_BATCH_SIZE"
	envBlocoGPUVerifyCPU = "BLOCO_GPU_VERIFY_CPU"
)

const (
	phase6SpeedupThreshold     = 1.5
	phase6StabilityCVThreshold = 0.20
)

type benchmarkOptions struct {
	Attempts        int
	Duration        time.Duration
	Detailed        bool
	Compare         bool
	ComparePatterns []string
	CompareBatches  []int
	CompareChecksum []bool
	Engine          string
	RequestedEngine string
	FallbackReason  string
	Network         string
	BatchSize       int
	Format          string
	Output          string
	UseTUI          bool
	MetalValidation string
	Criteria        wallet.GenerationCriteria
}

type benchmarkSample struct {
	Attempts int64
	Matches  int64
	Speed    float64
	Elapsed  time.Duration
	Result   *wallet.BenchmarkResult
}

type benchmarkComparisonResult struct {
	Phase                 string                    `json:"phase"`
	RequestedEngine       string                    `json:"requested_engine,omitempty"`
	DefaultEngineDecision string                    `json:"default_engine_decision"`
	DecisionReason        string                    `json:"decision_reason"`
	SpeedupThreshold      float64                   `json:"speedup_threshold"`
	StabilityCVThreshold  float64                   `json:"stability_cv_threshold"`
	Platform              string                    `json:"platform"`
	PowerMode             string                    `json:"power_mode,omitempty"`
	MetalAvailable        bool                      `json:"metal_available"`
	MetalDeviceName       string                    `json:"metal_device_name,omitempty"`
	Cases                 []benchmarkComparisonCase `json:"cases"`
}

type benchmarkComparisonCase struct {
	Pattern                     string                  `json:"pattern"`
	Checksum                    bool                    `json:"checksum"`
	BatchSize                   int                     `json:"batch_size"`
	CPU                         *wallet.BenchmarkResult `json:"cpu,omitempty"`
	Metal                       *wallet.BenchmarkResult `json:"metal,omitempty"`
	SpeedupVsCPU                float64                 `json:"speedup_vs_cpu,omitempty"`
	MetalStable                 bool                    `json:"metal_stable"`
	MetalCoefficientOfVariation float64                 `json:"metal_coefficient_of_variation,omitempty"`
	Decision                    string                  `json:"decision"`
	Error                       string                  `json:"error,omitempty"`
}

func (app *Application) getBenchmarkOptions(cmd *cobra.Command) (benchmarkOptions, error) {
	if err := app.parseFlags(cmd); err != nil {
		return benchmarkOptions{}, err
	}

	criteria, err := app.getGenerationCriteria(cmd)
	if err != nil {
		return benchmarkOptions{}, err
	}

	pattern, _ := cmd.Flags().GetString("pattern")
	if pattern != "" && criteria.Prefix == "" && criteria.Suffix == "" {
		criteria.Prefix = pattern
		if err := criteria.Validate(); err != nil {
			return benchmarkOptions{}, err
		}
	}

	attempts, _ := cmd.Flags().GetInt("attempts")
	duration, _ := cmd.Flags().GetDuration("duration")
	detailed, _ := cmd.Flags().GetBool("detailed")
	compare, _ := cmd.Flags().GetBool("compare")
	engineName := flagStringOrEnv(cmd, "engine", envBlocoEngine)
	batchSize, err := flagIntOrEnv(cmd, "batch-size", envBlocoGPUBatchSize)
	if err != nil {
		return benchmarkOptions{}, err
	}
	format, _ := cmd.Flags().GetString("format")
	output, _ := cmd.Flags().GetString("output")
	useTUI, _ := cmd.Flags().GetBool("tui")
	metalValidation, _ := cmd.Flags().GetString("metal-validation")
	comparePatterns, err := parseBenchmarkPatternList(flagString(cmd, "compare-patterns"))
	if err != nil {
		return benchmarkOptions{}, err
	}
	compareBatches, err := parseBenchmarkPositiveIntList(flagString(cmd, "compare-batch-sizes"))
	if err != nil {
		return benchmarkOptions{}, err
	}
	compareChecksums, err := parseBenchmarkBoolList(flagString(cmd, "compare-checksums"))
	if err != nil {
		return benchmarkOptions{}, err
	}

	if attempts <= 0 {
		return benchmarkOptions{}, fmt.Errorf("attempts must be positive, got %d", attempts)
	}
	if duration <= 0 {
		return benchmarkOptions{}, fmt.Errorf("duration must be positive, got %s", duration)
	}
	if batchSize <= 0 {
		return benchmarkOptions{}, fmt.Errorf("batch-size must be positive, got %d", batchSize)
	}

	network := strings.ToLower(strings.TrimSpace(criteria.Network))
	if network == "" {
		network = "ethereum"
		criteria.Network = network
	}
	if network != "ethereum" {
		return benchmarkOptions{}, fmt.Errorf("benchmark currently supports ethereum only, got %s", criteria.Network)
	}

	requestedEngine := strings.ToLower(strings.TrimSpace(engineName))
	if requestedEngine == "" {
		requestedEngine = engine.NameAuto
	}

	selection, err := engine.Resolve(requestedEngine)
	if err != nil {
		if !compare || requestedEngine != engine.NameMetal {
			return benchmarkOptions{}, err
		}
		selection = engine.Selection{Requested: requestedEngine, Resolved: engine.NameCPU, FallbackReason: engine.MetalUnavailableReason()}
	}

	format = strings.ToLower(strings.TrimSpace(format))
	if format == "" {
		format = benchmarkFormatText
	}
	if format != benchmarkFormatText && format != benchmarkFormatJSON {
		return benchmarkOptions{}, fmt.Errorf("unsupported benchmark format %q (expected text or json)", format)
	}
	metalValidation, err = resolveMetalValidationConfig(cmd, selection.Resolved, metalValidation)
	if err != nil {
		return benchmarkOptions{}, err
	}

	return benchmarkOptions{
		Attempts:        attempts,
		Duration:        duration,
		Detailed:        detailed,
		Compare:         compare,
		ComparePatterns: comparePatterns,
		CompareBatches:  compareBatches,
		CompareChecksum: compareChecksums,
		Engine:          selection.Resolved,
		RequestedEngine: selection.Requested,
		FallbackReason:  selection.FallbackReason,
		Network:         network,
		BatchSize:       batchSize,
		Format:          format,
		Output:          output,
		UseTUI:          useTUI,
		MetalValidation: metalValidation,
		Criteria:        criteria,
	}, nil
}

func (app *Application) runBenchmarkEngine(ctx context.Context, options benchmarkOptions, sampleInterval time.Duration, onSample func(benchmarkSample)) (*wallet.BenchmarkResult, error) {
	benchmarkEngine, err := engine.NewBenchmarkEngine(options.Engine)
	if err != nil {
		return nil, err
	}
	if closer, ok := benchmarkEngine.(interface{ Close() }); ok {
		defer closer.Close()
	}

	engineOptions := engine.BenchmarkOptions{
		Attempts:        options.Attempts,
		Duration:        options.Duration,
		BatchSize:       options.BatchSize,
		ThreadCount:     app.config.Worker.ThreadCount,
		Network:         options.Network,
		Criteria:        options.Criteria,
		RequestedEngine: options.RequestedEngine,
		FallbackReason:  options.FallbackReason,
		MetalValidation: options.MetalValidation,
	}

	var engineSample func(engine.Sample)
	if onSample != nil {
		engineSample = func(sample engine.Sample) {
			onSample(benchmarkSample{
				Attempts: sample.Attempts,
				Matches:  sample.Matches,
				Speed:    sample.Speed,
				Elapsed:  sample.Elapsed,
				Result:   sample.Result,
			})
		}
	}
	result, err := benchmarkEngine.RunBenchmark(ctx, engineOptions, sampleInterval, engineSample)
	if err != nil {
		return nil, err
	}
	annotateBenchmarkDiagnostics(result)
	return result, nil
}

func (app *Application) runBenchmarkComparison(ctx context.Context, options benchmarkOptions) (*benchmarkComparisonResult, error) {
	comparison := &benchmarkComparisonResult{
		Phase:                 "6",
		RequestedEngine:       options.RequestedEngine,
		DefaultEngineDecision: engine.NameCPU,
		DecisionReason:        "metal remains experimental until every local comparison case shows stable speedup >= 1.5x",
		SpeedupThreshold:      phase6SpeedupThreshold,
		StabilityCVThreshold:  phase6StabilityCVThreshold,
		MetalAvailable:        engine.MetalAvailable(),
		MetalDeviceName:       engine.MetalDeviceName(),
		Platform:              runtime.GOOS + "/" + runtime.GOARCH,
		PowerMode:             detectPowerMode(),
	}

	for _, pattern := range options.ComparePatterns {
		for _, checksum := range options.CompareChecksum {
			for _, batchSize := range options.CompareBatches {
				comparison.Cases = append(comparison.Cases, app.runBenchmarkComparisonCase(ctx, options, pattern, checksum, batchSize))
			}
		}
	}

	comparison.DefaultEngineDecision, comparison.DecisionReason = decideBenchmarkDefault(comparison.Cases)
	return comparison, nil
}

func (app *Application) runBenchmarkComparisonCase(ctx context.Context, options benchmarkOptions, pattern string, checksum bool, batchSize int) benchmarkComparisonCase {
	resultCase := benchmarkComparisonCase{
		Pattern:   pattern,
		Checksum:  checksum,
		BatchSize: batchSize,
		Decision:  engine.NameCPU,
	}
	criteria := options.Criteria
	criteria.Prefix = pattern
	criteria.Suffix = ""
	criteria.IsChecksum = checksum
	if !checksum {
		criteria.CaseSensitive = false
	}
	if err := criteria.Validate(); err != nil {
		resultCase.Error = err.Error()
		return resultCase
	}

	cpuOptions := options
	cpuOptions.Engine = engine.NameCPU
	cpuOptions.RequestedEngine = engine.NameCPU
	cpuOptions.FallbackReason = ""
	cpuOptions.BatchSize = batchSize
	cpuOptions.Criteria = criteria
	sampleInterval := benchmarkComparisonSampleInterval(options.Duration)
	cpuResult, err := app.runBenchmarkEngine(ctx, cpuOptions, sampleInterval, nil)
	if err != nil {
		resultCase.Error = err.Error()
		return resultCase
	}
	resultCase.CPU = cpuResult

	metalOptions := options
	metalOptions.Engine = engine.NameMetal
	metalOptions.RequestedEngine = engine.NameMetal
	metalOptions.FallbackReason = ""
	metalOptions.BatchSize = batchSize
	metalOptions.Criteria = criteria
	metalResult, err := app.runBenchmarkEngine(ctx, metalOptions, sampleInterval, nil)
	if err != nil {
		resultCase.Error = err.Error()
		return resultCase
	}
	resultCase.Metal = metalResult
	if cpuResult.AverageSpeed > 0 {
		resultCase.SpeedupVsCPU = metalResult.AverageSpeed / cpuResult.AverageSpeed
	}
	resultCase.MetalCoefficientOfVariation = benchmarkCoefficientOfVariation(metalResult.SpeedSamples)
	resultCase.MetalStable = len(metalResult.SpeedSamples) > 1 && resultCase.MetalCoefficientOfVariation <= phase6StabilityCVThreshold
	if resultCase.SpeedupVsCPU >= phase6SpeedupThreshold && resultCase.MetalStable {
		resultCase.Decision = engine.NameMetal
	}
	return resultCase
}

func resolveCommandEngine(cmd *cobra.Command) (engine.Selection, error) {
	return engine.Resolve(flagStringOrEnv(cmd, "engine", envBlocoEngine))
}

func renderBenchmarkResults(result *wallet.BenchmarkResult, detailed bool) string {
	var b strings.Builder
	fmt.Fprintf(&b, "\nBenchmark Results:\n")
	fmt.Fprintf(&b, "═══════════════════════════════════════\n")
	fmt.Fprintf(&b, "Engine: %s\n", result.Engine)
	if result.RequestedEngine != "" && result.RequestedEngine != result.Engine {
		fmt.Fprintf(&b, "Requested Engine: %s\n", result.RequestedEngine)
	}
	if result.FallbackReason != "" {
		fmt.Fprintf(&b, "Fallback: %s\n", result.FallbackReason)
	}
	if result.DeviceName != "" {
		fmt.Fprintf(&b, "Device: %s\n", result.DeviceName)
	} else if result.MetalDeviceName != "" {
		fmt.Fprintf(&b, "Metal Device: %s\n", result.MetalDeviceName)
	}
	fmt.Fprintf(&b, "Network: %s\n", result.Network)
	if result.IsSynthetic {
		fmt.Fprintf(&b, "Mode: synthetic address matching\n")
	}
	if result.IsHybrid {
		fmt.Fprintf(&b, "Mode: hybrid CPU private keys + Metal secp256k1/Keccak/matching\n")
		if result.MetalValidation != "" {
			fmt.Fprintf(&b, "Metal Validation: %s\n", result.MetalValidation)
		}
	}
	fmt.Fprintf(&b, "Pattern: %s\n", displayBenchmarkPattern(result.Pattern))
	fmt.Fprintf(&b, "Batch Size: %d\n", result.BatchSize)
	fmt.Fprintf(&b, "Total Attempts: %s\n", formatLargeNumber(result.TotalAttempts))
	fmt.Fprintf(&b, "Matches: %s\n", formatLargeNumber(result.Matches))
	fmt.Fprintf(&b, "Duration: %s\n", formatBenchmarkDuration(result.TotalDuration))
	fmt.Fprintf(&b, "Average Speed: %.0f addr/s\n", result.AverageSpeed)
	if result.CPUThroughput > 0 {
		fmt.Fprintf(&b, "CPU Throughput: %.0f addr/s\n", result.CPUThroughput)
	}
	if result.GPUThroughput > 0 {
		fmt.Fprintf(&b, "GPU Throughput: %.0f addr/s\n", result.GPUThroughput)
	}

	if result.MinSpeed > 0 && result.MaxSpeed > 0 {
		fmt.Fprintf(&b, "Speed Range: %.0f - %.0f addr/s\n", result.MinSpeed, result.MaxSpeed)
	}

	fmt.Fprintf(&b, "\nStage Timings:\n")
	fmt.Fprintf(&b, "  Entropy/private keys: %s\n", formatBenchmarkDuration(result.EntropyDuration))
	if result.IsHybrid {
		fmt.Fprintf(&b, "  CPU validation secp256k1: %s\n", formatBenchmarkDuration(result.ScalarBaseMultDuration))
		fmt.Fprintf(&b, "  CPU validation Keccak/address: %s\n", formatBenchmarkDuration(result.HashFormatDuration))
		fmt.Fprintf(&b, "  CPU validation matching: %s\n", formatBenchmarkDuration(result.MatchDuration))
	} else {
		fmt.Fprintf(&b, "  secp256k1 ScalarBaseMult: %s\n", formatBenchmarkDuration(result.ScalarBaseMultDuration))
		fmt.Fprintf(&b, "  Keccak/address format: %s\n", formatBenchmarkDuration(result.HashFormatDuration))
		fmt.Fprintf(&b, "  Pattern matching: %s\n", formatBenchmarkDuration(result.MatchDuration))
	}
	if result.MetalBufferDuration > 0 {
		fmt.Fprintf(&b, "  Metal buffer prep: %s\n", formatBenchmarkDuration(result.MetalBufferDuration))
	}
	if result.KernelDuration > 0 {
		fmt.Fprintf(&b, "  Metal kernel: %s\n", formatBenchmarkDuration(result.KernelDuration))
	}

	if result.ThreadCount > 1 {
		fmt.Fprintf(&b, "\nMulti-Threading Performance:\n")
		fmt.Fprintf(&b, "Threads Used: %d\n", result.ThreadCount)
		if result.SingleThreadSpeed > 0 {
			fmt.Fprintf(&b, "Estimated Single-Thread Speed: %.0f addr/s\n", result.SingleThreadSpeed)
			fmt.Fprintf(&b, "Estimated Multi-Thread Speedup: %.2fx\n", result.SpeedupVsSingleThread)
		}
	}

	if detailed && len(result.SpeedSamples) > 0 {
		fmt.Fprintf(&b, "\nDetailed Performance Samples:\n")
		samplesToShow := 5
		if len(result.SpeedSamples) <= samplesToShow*2 {
			for i, speed := range result.SpeedSamples {
				fmt.Fprintf(&b, "  Sample %d: %.0f addr/s\n", i+1, speed)
			}
		} else {
			for i := 0; i < samplesToShow; i++ {
				fmt.Fprintf(&b, "  Sample %d: %.0f addr/s\n", i+1, result.SpeedSamples[i])
			}
			fmt.Fprintf(&b, "  ... (%d samples omitted) ...\n", len(result.SpeedSamples)-samplesToShow*2)
			for i := len(result.SpeedSamples) - samplesToShow; i < len(result.SpeedSamples); i++ {
				fmt.Fprintf(&b, "  Sample %d: %.0f addr/s\n", i+1, result.SpeedSamples[i])
			}
		}

		mean, stdDev := benchmarkSampleStats(result.SpeedSamples)
		if mean > 0 {
			fmt.Fprintf(&b, "\nSpeed Statistics:\n")
			fmt.Fprintf(&b, "  Mean: %.0f addr/s\n", mean)
			fmt.Fprintf(&b, "  Std Dev: %.0f addr/s\n", stdDev)
			fmt.Fprintf(&b, "  Coefficient of Variation: %.1f%%\n", stdDev/mean*100)
		}
	}

	fmt.Fprintf(&b, "\nPerformance Analysis:\n")
	if result.AverageSpeed > 100000 {
		fmt.Fprintf(&b, "  Excellent performance (>100k addr/s)\n")
	} else if result.AverageSpeed > 50000 {
		fmt.Fprintf(&b, "  Good performance (>50k addr/s)\n")
	} else if result.AverageSpeed > 10000 {
		fmt.Fprintf(&b, "  Moderate performance (>10k addr/s)\n")
	} else {
		fmt.Fprintf(&b, "  Performance could be improved (<10k addr/s)\n")
	}

	return b.String()
}

func formatBenchmarkDuration(duration time.Duration) string {
	if duration < 0 {
		return formatDuration(duration)
	}
	if duration == 0 {
		return "0s"
	}
	if duration < time.Microsecond {
		return fmt.Sprintf("%dns", duration.Nanoseconds())
	}
	if duration < time.Millisecond {
		return fmt.Sprintf("%.1fus", float64(duration)/float64(time.Microsecond))
	}
	if duration < time.Second {
		return fmt.Sprintf("%.1fms", float64(duration)/float64(time.Millisecond))
	}
	return formatDuration(duration)
}

func displayBenchmarkPattern(pattern string) string {
	if pattern == "" {
		return "<none>"
	}
	return pattern
}

func flagString(cmd *cobra.Command, flagName string) string {
	value, _ := cmd.Flags().GetString(flagName)
	return value
}

func flagStringOrEnv(cmd *cobra.Command, flagName string, envName string) string {
	value, _ := cmd.Flags().GetString(flagName)
	if cmd.Flags().Changed(flagName) {
		return value
	}
	if envValue := strings.TrimSpace(os.Getenv(envName)); envValue != "" {
		return envValue
	}
	return value
}

func flagIntOrEnv(cmd *cobra.Command, flagName string, envName string) (int, error) {
	value, _ := cmd.Flags().GetInt(flagName)
	if cmd.Flags().Changed(flagName) {
		return value, nil
	}
	envValue := strings.TrimSpace(os.Getenv(envName))
	if envValue == "" {
		return value, nil
	}
	parsed, err := strconv.Atoi(envValue)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer, got %q", envName, envValue)
	}
	return parsed, nil
}

func resolveMetalValidationConfig(cmd *cobra.Command, resolvedEngine string, mode string) (string, error) {
	if cmd.Flags().Changed("metal-validation") {
		return engine.NormalizeMetalValidationMode(mode)
	}
	envValue := strings.TrimSpace(os.Getenv(envBlocoGPUVerifyCPU))
	if envValue == "" {
		return engine.NormalizeMetalValidationMode(mode)
	}
	verifyCPU, err := strconv.ParseBool(envValue)
	if err != nil {
		return "", fmt.Errorf("%s must be true or false, got %q", envBlocoGPUVerifyCPU, envValue)
	}
	if !verifyCPU && resolvedEngine == engine.NameMetal {
		return "", fmt.Errorf("%s=false is unsupported for metal benchmarks (Phase 4 requires full CPU verification)", envBlocoGPUVerifyCPU)
	}
	return engine.MetalValidationFull, nil
}

func annotateBenchmarkDiagnostics(result *wallet.BenchmarkResult) {
	result.MetalAvailable = engine.MetalAvailable()
	result.MetalDeviceName = result.DeviceName
	if result.MetalDeviceName == "" {
		result.MetalDeviceName = engine.MetalDeviceName()
	}
}

func parseBenchmarkPatternList(value string) ([]string, error) {
	parts := strings.Split(value, ",")
	patterns := make([]string, 0, len(parts))
	for _, part := range parts {
		pattern := strings.TrimSpace(part)
		if pattern == "" {
			continue
		}
		criteria := wallet.GenerationCriteria{Network: "ethereum", Prefix: pattern}
		if err := criteria.Validate(); err != nil {
			return nil, fmt.Errorf("invalid compare pattern %q: %w", pattern, err)
		}
		patterns = append(patterns, pattern)
	}
	if len(patterns) == 0 {
		return nil, fmt.Errorf("compare-patterns must include at least one pattern")
	}
	return patterns, nil
}

func parseBenchmarkPositiveIntList(value string) ([]int, error) {
	parts := strings.Split(value, ",")
	values := make([]int, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		parsed, err := strconv.Atoi(part)
		if err != nil {
			return nil, fmt.Errorf("compare-batch-sizes must contain integers, got %q", part)
		}
		if parsed <= 0 {
			return nil, fmt.Errorf("compare-batch-sizes must contain positive integers, got %d", parsed)
		}
		values = append(values, parsed)
	}
	if len(values) == 0 {
		return nil, fmt.Errorf("compare-batch-sizes must include at least one value")
	}
	return values, nil
}

func parseBenchmarkBoolList(value string) ([]bool, error) {
	parts := strings.Split(value, ",")
	values := make([]bool, 0, len(parts))
	for _, part := range parts {
		part = strings.ToLower(strings.TrimSpace(part))
		if part == "" {
			continue
		}
		switch part {
		case "on", "true", "1", "yes":
			values = append(values, true)
		case "off", "false", "0", "no":
			values = append(values, false)
		default:
			return nil, fmt.Errorf("compare-checksums must contain true/false or on/off, got %q", part)
		}
	}
	if len(values) == 0 {
		return nil, fmt.Errorf("compare-checksums must include at least one value")
	}
	return values, nil
}

func benchmarkComparisonSampleInterval(duration time.Duration) time.Duration {
	if duration <= 0 {
		return time.Second
	}
	interval := duration / 4
	if interval < time.Millisecond {
		return time.Millisecond
	}
	return interval
}

func benchmarkCoefficientOfVariation(samples []float64) float64 {
	mean, stdDev := benchmarkSampleStats(samples)
	if mean <= 0 {
		return 0
	}
	return stdDev / mean
}

func decideBenchmarkDefault(cases []benchmarkComparisonCase) (string, string) {
	if len(cases) == 0 {
		return engine.NameCPU, "no comparison cases were executed"
	}
	for _, resultCase := range cases {
		if resultCase.Error != "" {
			return engine.NameCPU, "cpu remains default because at least one comparison case failed"
		}
		if resultCase.Decision != engine.NameMetal {
			return engine.NameCPU, "cpu remains default because metal did not show stable speedup >= 1.5x in every case"
		}
	}
	return engine.NameMetal, "metal can become default on this Apple Silicon profile because every local case showed stable speedup >= 1.5x"
}

func detectPowerMode() string {
	if runtime.GOOS != "darwin" {
		return ""
	}
	output, err := exec.Command("pmset", "-g", "batt").Output()
	if err != nil {
		return ""
	}
	text := strings.ToLower(string(output))
	if strings.Contains(text, "battery power") {
		return "battery"
	}
	if strings.Contains(text, "ac power") {
		return "ac"
	}
	return ""
}

func benchmarkSampleStats(samples []float64) (float64, float64) {
	if len(samples) == 0 {
		return 0, 0
	}
	var sum float64
	for _, speed := range samples {
		sum += speed
	}
	mean := sum / float64(len(samples))
	if len(samples) == 1 {
		return mean, 0
	}
	var sumSquares float64
	for _, speed := range samples {
		delta := speed - mean
		sumSquares += delta * delta
	}
	return mean, math.Sqrt(sumSquares / float64(len(samples)-1))
}

func (app *Application) runBenchmarkComparisonText(ctx context.Context, options benchmarkOptions) error {
	if options.Format == benchmarkFormatText {
		fmt.Printf("Running Phase 6 CPU vs Metal comparison...\n")
		fmt.Printf("Patterns: %s\n", strings.Join(options.ComparePatterns, ", "))
		fmt.Printf("Batch Sizes: %s\n", formatIntList(options.CompareBatches))
		fmt.Printf("Checksum Modes: %s\n\n", formatBoolList(options.CompareChecksum))
	}

	comparison, err := app.runBenchmarkComparison(ctx, options)
	if err != nil {
		return err
	}
	return app.writeBenchmarkComparisonOutput(comparison, options)
}

func (app *Application) writeBenchmarkComparisonOutput(result *benchmarkComparisonResult, options benchmarkOptions) error {
	var output []byte
	var err error

	switch options.Format {
	case benchmarkFormatJSON:
		output, err = json.MarshalIndent(result, "", "  ")
		if err != nil {
			return err
		}
		output = append(output, '\n')
	case benchmarkFormatText:
		output = []byte(renderBenchmarkComparisonResults(result))
	default:
		return fmt.Errorf("unsupported benchmark format %q", options.Format)
	}

	if options.Output != "" {
		return os.WriteFile(options.Output, output, 0600)
	}

	_, err = os.Stdout.Write(output)
	return err
}

func renderBenchmarkComparisonResults(result *benchmarkComparisonResult) string {
	var b strings.Builder
	fmt.Fprintf(&b, "\nPhase 6 Benchmark Comparison:\n")
	fmt.Fprintf(&b, "═══════════════════════════════════════\n")
	fmt.Fprintf(&b, "Platform: %s\n", result.Platform)
	if result.PowerMode != "" {
		fmt.Fprintf(&b, "Power Mode: %s\n", result.PowerMode)
	}
	fmt.Fprintf(&b, "Metal Available: %s\n", formatBool(result.MetalAvailable))
	if result.MetalDeviceName != "" {
		fmt.Fprintf(&b, "Metal Device: %s\n", result.MetalDeviceName)
	}
	fmt.Fprintf(&b, "Speedup Threshold: %.2fx\n", result.SpeedupThreshold)
	fmt.Fprintf(&b, "Stability CV Threshold: %.0f%%\n\n", result.StabilityCVThreshold*100)

	for _, resultCase := range result.Cases {
		fmt.Fprintf(&b, "Pattern=%s checksum=%s batch=%d\n", displayBenchmarkPattern(resultCase.Pattern), formatBool(resultCase.Checksum), resultCase.BatchSize)
		if resultCase.CPU != nil {
			fmt.Fprintf(&b, "  CPU: %.0f addr/s\n", resultCase.CPU.AverageSpeed)
		}
		if resultCase.Metal != nil {
			fmt.Fprintf(&b, "  Metal: %.0f addr/s\n", resultCase.Metal.AverageSpeed)
			fmt.Fprintf(&b, "  Speedup vs CPU: %.2fx\n", resultCase.SpeedupVsCPU)
			fmt.Fprintf(&b, "  Metal Stability CV: %.1f%%\n", resultCase.MetalCoefficientOfVariation*100)
		}
		if resultCase.Error != "" {
			fmt.Fprintf(&b, "  Error: %s\n", resultCase.Error)
		}
		fmt.Fprintf(&b, "  Case Decision: %s\n", resultCase.Decision)
	}

	fmt.Fprintf(&b, "\nDefault Engine Decision: %s\n", result.DefaultEngineDecision)
	fmt.Fprintf(&b, "Reason: %s\n", result.DecisionReason)
	return b.String()
}

func formatIntList(values []int) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, strconv.Itoa(value))
	}
	return strings.Join(parts, ", ")
}

func formatBoolList(values []bool) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		if value {
			parts = append(parts, "on")
		} else {
			parts = append(parts, "off")
		}
	}
	return strings.Join(parts, ", ")
}

func (app *Application) writeBenchmarkOutput(result *wallet.BenchmarkResult, options benchmarkOptions) error {
	var output []byte
	var err error

	switch options.Format {
	case benchmarkFormatJSON:
		output, err = json.MarshalIndent(result, "", "  ")
		if err != nil {
			return err
		}
		output = append(output, '\n')
	case benchmarkFormatText:
		output = []byte(renderBenchmarkResults(result, options.Detailed))
	default:
		return fmt.Errorf("unsupported benchmark format %q", options.Format)
	}

	if options.Output != "" {
		return os.WriteFile(options.Output, output, 0600)
	}

	_, err = os.Stdout.Write(output)
	return err
}

// newBenchmarkTUIEngineInfo builds the engine diagnostics block consumed by
// the benchmark TUI. It mirrors the fields printed by runBenchmarkText so both
// modes expose the same engine, device, batch and validation context.
func newBenchmarkTUIEngineInfo(options benchmarkOptions, threadCount int) tui.EngineInfo {
	info := tui.EngineInfo{
		Engine:          options.Engine,
		RequestedEngine: options.RequestedEngine,
		FallbackReason:  options.FallbackReason,
		ThreadCount:     threadCount,
		Network:         options.Network,
		BatchSize:       options.BatchSize,
	}
	if options.Engine == engine.NameMetal {
		info.DeviceName = engine.MetalDeviceName()
		info.MetalValidation = options.MetalValidation
	}
	return info
}

func sendBenchmarkTUISample(program *tea.Program, criteria wallet.GenerationCriteria, sample benchmarkSample) {
	avgSpeed := sample.Speed
	if sample.Result != nil {
		avgSpeed = sample.Result.AverageSpeed
	}
	program.Send(tui.BenchmarkUpdateMsg{
		Running: sample.Result == nil,
		Progress: tui.ProgressMsg{
			Attempts:      sample.Attempts,
			Speed:         sample.Speed,
			Pattern:       criteria.GetPattern(),
			Difficulty:    calculateDifficulty(criteria),
			EstimatedTime: estimateBenchmarkRemaining(criteria, sample.Attempts, avgSpeed),
		},
	})
}

func estimateBenchmarkRemaining(criteria wallet.GenerationCriteria, attempts int64, speed float64) time.Duration {
	if speed <= 0 {
		return 0
	}
	difficulty := calculateDifficulty(criteria)
	remaining := int64(difficulty) - attempts
	if remaining <= 0 {
		return 0
	}
	return time.Duration(float64(remaining)/speed) * time.Second
}
