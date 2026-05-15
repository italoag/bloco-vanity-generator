package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
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

type benchmarkOptions struct {
	Attempts        int
	Duration        time.Duration
	Detailed        bool
	Engine          string
	RequestedEngine string
	FallbackReason  string
	Network         string
	BatchSize       int
	Format          string
	Output          string
	UseTUI          bool
	Criteria        wallet.GenerationCriteria
}

type benchmarkSample struct {
	Attempts int64
	Matches  int64
	Speed    float64
	Elapsed  time.Duration
	Result   *wallet.BenchmarkResult
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
	engineName, _ := cmd.Flags().GetString("engine")
	batchSize, _ := cmd.Flags().GetInt("batch-size")
	format, _ := cmd.Flags().GetString("format")
	output, _ := cmd.Flags().GetString("output")
	useTUI, _ := cmd.Flags().GetBool("tui")

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
		return benchmarkOptions{}, err
	}

	format = strings.ToLower(strings.TrimSpace(format))
	if format == "" {
		format = benchmarkFormatText
	}
	if format != benchmarkFormatText && format != benchmarkFormatJSON {
		return benchmarkOptions{}, fmt.Errorf("unsupported benchmark format %q (expected text or json)", format)
	}

	return benchmarkOptions{
		Attempts:        attempts,
		Duration:        duration,
		Detailed:        detailed,
		Engine:          selection.Resolved,
		RequestedEngine: selection.Requested,
		FallbackReason:  selection.FallbackReason,
		Network:         network,
		BatchSize:       batchSize,
		Format:          format,
		Output:          output,
		UseTUI:          useTUI,
		Criteria:        criteria,
	}, nil
}

func (app *Application) runBenchmarkEngine(ctx context.Context, options benchmarkOptions, sampleInterval time.Duration, onSample func(benchmarkSample)) (*wallet.BenchmarkResult, error) {
	benchmarkEngine, err := engine.NewBenchmarkEngine(options.Engine)
	if err != nil {
		return nil, err
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
	return benchmarkEngine.RunBenchmark(ctx, engineOptions, sampleInterval, engineSample)
}

func resolveCommandEngine(cmd *cobra.Command) (engine.Selection, error) {
	engineName, _ := cmd.Flags().GetString("engine")
	return engine.Resolve(engineName)
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
	}
	fmt.Fprintf(&b, "Network: %s\n", result.Network)
	if result.IsSynthetic {
		fmt.Fprintf(&b, "Mode: synthetic address matching\n")
	}
	if result.IsHybrid {
		fmt.Fprintf(&b, "Mode: hybrid CPU private keys + Metal secp256k1/Keccak/matching\n")
	}
	fmt.Fprintf(&b, "Pattern: %s\n", displayBenchmarkPattern(result.Pattern))
	fmt.Fprintf(&b, "Batch Size: %d\n", result.BatchSize)
	fmt.Fprintf(&b, "Total Attempts: %s\n", formatLargeNumber(result.TotalAttempts))
	fmt.Fprintf(&b, "Matches: %s\n", formatLargeNumber(result.Matches))
	fmt.Fprintf(&b, "Duration: %s\n", formatBenchmarkDuration(result.TotalDuration))
	fmt.Fprintf(&b, "Average Speed: %.0f addr/s\n", result.AverageSpeed)

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
